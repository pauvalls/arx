# Proposal: Arx v0.15.0 — Man Pages, Hygiene, Ignore, Fuzz Tests

## Problem

Arx lacks several project-hygiene and usability features:
1. **No man pages** — users cannot access offline documentation via `man arx`
2. **Missing hygiene files** — no `.editorconfig` or `.github/dependabot.yml`
3. **No .arxignore support** — users cannot exclude paths from detector scans
4. **Incomplete fuzz coverage** — Kotlin, PHP, and Swift parsers lack fuzz tests

## Approach

### 1. man-pages
- New `cmd/arx/man.go` with `arx man` subcommand
- Uses Cobra's `GenManTree()` to generate man pages to `--output` directory or stdout
- Follows existing completion command pattern

### 2. project-hygiene
- Add `.editorconfig` at project root with standard Go/editor settings
- Add `.github/dependabot.yml` for Go module and GitHub Actions updates

### 3. arxignore
- New `internal/domain/ignore.go` with `ArxIgnore` struct
- `LoadArxIgnore(root string)` reads `.arxignore` from project root
- `IsIgnored(path string)` uses glob matching against patterns
- Integrate into detector `FindXxxFiles` methods to filter ignored paths

### 4. fuzz-tests-all
- Add fuzz tests for Kotlin, PHP, Swift parsers following existing `parser_fuzz_test.go` pattern
- Seed corpus with valid import statements per language
- Fuzz with random byte slices via `extractImportsFromLine`

## Scope

4 independent work streams, all additive. No breaking changes.

## Success Criteria

- `arx man --output ./man` generates valid man pages
- `.editorconfig` and `.github/dependabot.yml` exist at project root
- `.arxignore` patterns filter detector file discovery
- All 3 new fuzz tests pass and compile
