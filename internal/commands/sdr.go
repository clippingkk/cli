package commands

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/clippingkk/cli/internal/models"
	"github.com/clippingkk/cli/internal/sdr"
	"github.com/urfave/cli/v2"
)

// SDRCommand extracts highlights from Kindle .sdr sidecars and their books.
var SDRCommand = &cli.Command{
	Name:  "sdr",
	Usage: "Extract highlighted text from Kindle .sdr sidecars",
	Description: `Read Kindle .sdr sidecars and recover highlighted text from their
unencrypted AZW3/KF8 books. The path may be a mounted Kindle or documents tree,
a single .sdr directory, or a single AZW3/KF8 book.

Examples:
  ck-cli sdr --path /Volumes/Kindle/documents
  ck-cli sdr --path "Book.sdr" --json
  ck-cli sdr --path "Book.azw3" --json`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "path",
			Aliases:  []string{"p"},
			Usage:    "Kindle documents tree, .sdr directory, or AZW3/KF8 book",
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Output the existing ClippingItem JSON format",
		},
	},
	Action: sdrAction,
}

func sdrAction(c *cli.Context) error {
	report, err := sdr.ExtractPath(c.String("path"))
	for _, warning := range report.Warnings {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", warning)
	}
	if err != nil {
		return err
	}

	count := 0
	for _, book := range report.Books {
		count += len(book.Highlights)
	}
	if c.Bool("json") {
		items := make([]models.ClippingItem, 0, count)
		for _, book := range report.Books {
			for _, highlight := range book.Highlights {
				items = append(items, models.ClippingItem{
					Title: highlight.Title, Content: highlight.Text,
					PageAt: highlight.PageAt, CreatedAt: highlight.CreatedAt,
				})
			}
		}
		if err := outputJSON(os.Stdout, items); err != nil {
			return err
		}
	} else {
		if err := renderSDRText(os.Stdout, report); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "📚 Extracted %d text annotations from %d books\n", count, report.Decoded)
	return nil
}

func renderSDRText(writer io.Writer, report sdr.Report) error {
	for _, book := range report.Books {
		if len(book.Highlights) == 0 {
			continue
		}
		if _, err := fmt.Fprintln(writer, book.Title); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		if _, err := fmt.Fprintln(writer, strings.Repeat("=", utf8.RuneCountInString(book.Title))); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		for _, highlight := range book.Highlights {
			if _, err := fmt.Fprintf(writer, "[%s] %s\n", highlight.Type, highlight.Text); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			if highlight.Note != "" {
				if _, err := fmt.Fprintf(writer, "  Note: %s\n", highlight.Note); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			}
			if _, err := fmt.Fprintf(writer, "  Location: %s | Created: %s\n\n",
				highlight.PageAt, highlight.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}
	}
	return nil
}
