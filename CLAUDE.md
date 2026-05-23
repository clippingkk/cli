# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`ck-cli` (ClippingKK CLI) is a command-line tool that parses Amazon Kindle's `My Clippings.txt` file into structured JSON format and optionally syncs highlights to the ClippingKK web service.

It is written in TypeScript on the [Bun](https://bun.com) runtime and ships as a single self-contained binary per platform, produced by `bun build --compile`.

### Key Features

- Multi-language parsing (Chinese, English)
- Stdin/stdout + file I/O
- Chunked GraphQL upload to ClippingKK web service
- Cross-platform (linux/darwin/windows × amd64/arm64 where supported)
- macOS binaries are signed and notarized via Anchore Quill

## Key Commands

```bash
bun install                            # install deps
bun run dev parse --input clip.txt     # run from source
bun test                               # run all tests
bun test --coverage                    # tests + coverage
bun run lint                           # oxlint
bun run format                         # oxfmt (write)
bun run format:check                   # oxfmt --check
bun run typecheck                      # tsc --noEmit
bun run build                          # compile a binary for the local platform
bun run build:all                      # cross-compile all 5 targets
bun run build:all -- --archive         # cross-compile + tar.gz/zip + checksums.txt
```

## Architecture

```
src/
├── main.tsx              # cac entrypoint; signal handling; top-level error trap
├── version.ts            # VERSION/COMMIT injected at build time via --define
├── commands/
│   ├── login.tsx         # writes token to ~/.ck-cli.toml
│   └── parse.tsx         # reads stdin/file, parses, routes to stdout/file/http
├── config/config.ts      # TOML load/save (~/.ck-cli.toml)
├── http/client.ts        # GraphQL POST + chunking (20) + concurrency (10)
├── models/clipping.ts    # ClippingItem + RFC3339 serialization
├── parser/parser.ts      # detectLanguage, splitIntoGroups, parseGroup, date parsing
├── ui/                   # Ink components rendered to stderr
└── utils/semaphore.ts    # withConcurrency helper (no external dep)
tests/
├── parser.test.ts        # unit tests + edge cases
├── fixtures.test.ts      # byte-identical parity vs the original Go binary
├── config.test.ts
├── client.test.ts        # mocked fetch
└── fixtures/             # *.txt inputs + *.result.json oracles
scripts/build.ts          # cross-compile orchestrator
```

### The Parity Contract

`tests/fixtures/clippings_*.result.json` are byte-identical snapshots produced by the original Go binary. `tests/fixtures.test.ts` re-parses each `*.txt` and asserts that the serialized JSON matches the oracle to the byte. **Any parser change must preserve this contract.** Regenerating an oracle requires explicit justification.

### Build Pipeline

`bun build --compile` produces self-contained binaries that embed the Bun runtime (~60–110 MB depending on target). The build script injects `CK_VERSION` and `CK_COMMIT` via `--define` and dead-code-eliminates Ink's optional `react-devtools-core` import via `--define process.env.DEV='"false"'`.

Targets:

| Bun target | Output |
|---|---|
| `bun-linux-x64` | `ck-cli-linux-amd64` |
| `bun-linux-arm64` | `ck-cli-linux-arm64` |
| `bun-darwin-x64` | `ck-cli-darwin-amd64` (signed + notarized via Quill in release CI) |
| `bun-darwin-arm64` | `ck-cli-darwin-arm64` (signed + notarized via Quill in release CI) |
| `bun-windows-x64` | `ck-cli-windows-amd64.exe` |

### Release Flow

1. Conventional commits land on `master`.
2. `release-please` opens/updates a release PR that bumps `package.json#version` and `CHANGELOG.md`.
3. Merging the release PR creates a Git tag and a GitHub Release.
4. The release workflow builds all 5 targets on `ubuntu-latest`, signs macOS binaries with Quill (reusing `QUILL_*` secrets), archives, generates `checksums.txt`, and uploads to the GitHub Release.
5. A Docker image is built from the musl variant and pushed to GHCR.

## Development Guidelines

### Code Style

- TypeScript strict mode is on (`noUncheckedIndexedAccess`, etc.); honor it.
- Lint with `oxlint` and format with `oxfmt` — both run in CI.
- Imports use explicit `.ts`/`.tsx` extensions (`allowImportingTsExtensions`).

### Testing

- Use `bun:test` (Jest-compatible API).
- Add parser test cases to `tests/parser.test.ts` and, where reasonable, extend fixture coverage rather than mocking edge cases.
- Mock `globalThis.fetch` for HTTP client tests.

### Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/). Common scopes:

- `feat(parser): ...`, `fix(http): ...`, `refactor(config): ...`, `perf(http): ...`
- `test`, `docs`, `build`, `ci`, `chore` (these are hidden from the user-facing changelog)

### Adding a New Command

1. Create `src/commands/<name>.tsx`.
2. Export an async `runFoo(opts): Promise<number>` that returns an exit code.
3. Register it in `src/main.tsx` with cac.
4. Add tests in `tests/`.

### Modifying the Parser

The parser must remain byte-compatible with the existing fixture oracles. If you change parsing semantics intentionally:

1. Add a failing fixture test first.
2. Update the parser.
3. Regenerate the oracle deliberately and document the change in the commit.
