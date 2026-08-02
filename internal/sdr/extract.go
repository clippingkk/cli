package sdr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type bookCandidate struct {
	bookPath   string
	sidecarDir string
}

// ExtractPath discovers Kindle book/sidecar pairs beneath path and extracts
// all supported text annotations. It never writes to the source tree.
func ExtractPath(path string) (Report, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Report{}, fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return Report{}, fmt.Errorf("inspect %s: %w", path, err)
	}

	candidates, direct, discoveryWarnings, err := discoverCandidates(absPath, info)
	if err != nil {
		return Report{}, err
	}
	report := Report{Warnings: discoveryWarnings}
	for _, candidate := range candidates {
		result, err := extractCandidate(candidate)
		if err != nil {
			if direct {
				return Report{}, err
			}
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s: %v", candidate.sidecarDir, err))
			continue
		}
		report.Decoded++
		report.Books = append(report.Books, result)
	}
	if report.Decoded == 0 {
		if len(report.Warnings) > 0 {
			return report, fmt.Errorf("no supported Kindle sidecar pairs could be decoded")
		}
		return report, fmt.Errorf("no supported Kindle sidecar pairs found under %s", path)
	}
	return report, nil
}

func discoverCandidates(path string, info os.FileInfo) ([]bookCandidate, bool, []string, error) {
	if !info.IsDir() {
		if !supportedBookExtension(filepath.Ext(path)) {
			return nil, true, nil, fmt.Errorf("unsupported input file %s", path)
		}
		sidecar := strings.TrimSuffix(path, filepath.Ext(path)) + ".sdr"
		if sidecarInfo, err := os.Stat(sidecar); err != nil || !sidecarInfo.IsDir() {
			return nil, true, nil, fmt.Errorf("sidecar directory %s was not found", sidecar)
		}
		return []bookCandidate{{bookPath: path, sidecarDir: sidecar}}, true, nil, nil
	}
	if strings.EqualFold(filepath.Ext(path), ".sdr") {
		book, err := findSiblingBook(path)
		if err != nil {
			return nil, true, nil, err
		}
		return []bookCandidate{{bookPath: book, sidecarDir: path}}, true, nil, nil
	}

	var candidates []bookCandidate
	var warnings []string
	err := filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", current, walkErr))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sdr") {
			return nil
		}
		book, err := findSiblingBook(current)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", current, err))
			return filepath.SkipDir
		}
		candidates = append(candidates, bookCandidate{bookPath: book, sidecarDir: current})
		return filepath.SkipDir
	})
	if err != nil {
		return nil, false, warnings, fmt.Errorf("scan %s: %w", path, err)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].bookPath < candidates[j].bookPath
	})
	return candidates, false, warnings, nil
}

func findSiblingBook(sidecarDir string) (string, error) {
	base := strings.TrimSuffix(sidecarDir, filepath.Ext(sidecarDir))
	parent := filepath.Dir(sidecarDir)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", fmt.Errorf("read book directory: %w", err)
	}
	baseName := filepath.Base(base)
	hasAZW3R, hasYJR := sidecarFormats(sidecarDir)
	priority := map[string]int{".azw3": 0, ".azw": 1, ".mobi": 2, ".kfx": 3}
	type match struct {
		path     string
		priority int
	}
	var matches []match
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		order, ok := priority[ext]
		if !ok || !strings.EqualFold(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), baseName) {
			continue
		}
		if ext == ".kfx" && !hasYJR {
			continue
		}
		if ext != ".kfx" && !hasAZW3R {
			continue
		}
		matches = append(matches, match{path: filepath.Join(parent, entry.Name()), priority: order})
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no compatible sibling AZW3/KF8 or KFX book found")
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].priority != matches[j].priority {
			return matches[i].priority < matches[j].priority
		}
		return matches[i].path < matches[j].path
	})
	return matches[0].path, nil
}

func sidecarFormats(directory string) (hasAZW3R, hasYJR bool) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false, false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".azw3r":
			hasAZW3R = true
		case ".yjr":
			hasYJR = true
		}
	}
	return hasAZW3R, hasYJR
}

func supportedBookExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".azw3", ".azw", ".mobi", ".kfx":
		return true
	default:
		return false
	}
}

func extractCandidate(candidate bookCandidate) (BookResult, error) {
	bookExtension := strings.ToLower(filepath.Ext(candidate.bookPath))
	sidecarPaths, err := findSidecarFiles(candidate.sidecarDir, bookExtension)
	if err != nil {
		return BookResult{}, err
	}
	merged := Sidecar{}
	seenAnnotations := make(map[string]struct{})
	for _, path := range sidecarPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return BookResult{}, fmt.Errorf("read %s: %w", filepath.Base(path), err)
		}
		decoded, err := DecodeSidecar(data)
		if err != nil {
			return BookResult{}, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
		}
		if len(merged.PageMap.Positions) == 0 && len(decoded.PageMap.Positions) > 0 {
			merged.PageMap = decoded.PageMap
		}
		for _, annotation := range decoded.Annotations {
			key := fmt.Sprintf("%s\x00%d\x00%d\x00%d\x00%s", annotation.Type,
				annotation.StartPosition, annotation.EndPosition, annotation.CreationTime.UnixMilli(), annotation.Note)
			if _, duplicate := seenAnnotations[key]; duplicate {
				continue
			}
			seenAnnotations[key] = struct{}{}
			merged.Annotations = append(merged.Annotations, annotation)
		}
	}

	bookData, err := os.ReadFile(candidate.bookPath)
	if err != nil {
		return BookResult{}, fmt.Errorf("read book: %w", err)
	}
	var title string
	var assembled []byte
	var kfx kfxBook
	if bookExtension == ".kfx" {
		kfx, err = assembleKFX(bookData)
		if err != nil {
			return BookResult{}, fmt.Errorf("assemble %s: %w", filepath.Base(candidate.bookPath), err)
		}
		title = kfx.title
	} else {
		title, assembled, err = AssembleBook(bookData)
		if err != nil {
			return BookResult{}, fmt.Errorf("assemble %s: %w", filepath.Base(candidate.bookPath), err)
		}
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(candidate.bookPath), filepath.Ext(candidate.bookPath))
	}

	sort.SliceStable(merged.Annotations, func(i, j int) bool {
		if merged.Annotations[i].StartPosition != merged.Annotations[j].StartPosition {
			return merged.Annotations[i].StartPosition < merged.Annotations[j].StartPosition
		}
		if !merged.Annotations[i].CreationTime.Equal(merged.Annotations[j].CreationTime) {
			return merged.Annotations[i].CreationTime.Before(merged.Annotations[j].CreationTime)
		}
		return merged.Annotations[i].Type < merged.Annotations[j].Type
	})
	result := BookResult{BookPath: candidate.bookPath, SidecarDir: candidate.sidecarDir, Title: title}
	annotations := merged.Annotations
	if bookExtension == ".kfx" {
		annotations = associateNotes(annotations)
	}
	for _, annotation := range annotations {
		var text, exact string
		var page string
		if bookExtension == ".kfx" {
			var found bool
			text, found = kfx.textAt(annotation.StartPosition, annotation.EndPosition)
			if found {
				exact = text
			}
			page = kfx.pageAt(annotation.StartPosition)
		} else {
			text, exact = recoverText(assembled, annotation.StartPosition, annotation.EndPosition)
			page = pageAt(merged.PageMap, annotation.StartPosition)
		}
		if text == "" {
			continue
		}
		result.Highlights = append(result.Highlights, Highlight{
			Title: title, Text: text, ExactText: exact, Type: annotation.Type,
			StartPosition: annotation.StartPosition, EndPosition: annotation.EndPosition,
			CreatedAt: annotation.CreationTime.UTC(), Note: annotation.Note,
			PageAt: page,
		})
	}
	return result, nil
}

func associateNotes(annotations []Annotation) []Annotation {
	notesByStart := make(map[int64][]int)
	for index, annotation := range annotations {
		if annotation.Type == AnnotationNote && annotation.Note != "" {
			notesByStart[annotation.StartPosition] = append(notesByStart[annotation.StartPosition], index)
		}
	}
	consumed := make(map[int]bool)
	result := make([]Annotation, 0, len(annotations))
	for _, annotation := range annotations {
		if annotation.Type == AnnotationHighlight || annotation.Type == AnnotationUnderline {
			for _, noteIndex := range notesByStart[annotation.EndPosition] {
				if annotation.Note == "" {
					annotation.Note = annotations[noteIndex].Note
				} else {
					annotation.Note += "\n" + annotations[noteIndex].Note
				}
				consumed[noteIndex] = true
			}
		}
		result = append(result, annotation)
	}
	filtered := result[:0]
	for index, annotation := range result {
		if !consumed[index] {
			filtered = append(filtered, annotation)
		}
	}
	return filtered
}

func findSidecarFiles(directory, bookExtension string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read sidecar directory: %w", err)
	}
	var paths []string
	wanted := ".azw3r"
	if bookExtension == ".kfx" {
		wanted = ".yjr"
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) == wanted {
			paths = append(paths, filepath.Join(directory, entry.Name()))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("sidecar contains no %s file", wanted)
	}
	sort.Strings(paths)
	return paths, nil
}
