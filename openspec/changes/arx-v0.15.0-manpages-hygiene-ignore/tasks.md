# Tasks: Arx v0.15.0 — Man Pages, Hygiene, Ignore, Fuzz Tests

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~350-450 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | PR 1: arxignore core + detectors → PR 2: man pages + hygiene + fuzz |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

```text
Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Medium
```

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | .arxignore domain + detector integration | PR 1 | Largest scope (10 detectors); standalone |
| 2 | Man pages + hygiene files + fuzz tests | PR 2 | Independent of PR 1; all additive |

## Phase 1: Foundation — ArxIgnore Domain

- [x] 1.1 Create `internal/domain/ignore.go` with `ArxIgnore` struct, `LoadArxIgnore(root string) (*ArxIgnore, error)`, and `IsIgnored(path string) bool` using `path.Match`
- [x] 1.2 Create `internal/domain/ignore_test.go` with table-driven tests: missing file → empty, parse patterns/skip comments/blanks, glob matching (`*`, `?`, `[...]`), non-matching paths

## Phase 2: Detector Integration — ArxIgnore Wiring

- [ ] 2.1 Modify `internal/infrastructure/detector/kotlin/detector.go` — `FindKotlinFiles` accepts `*domain.ArxIgnore`, calls `IsIgnored(relPath)` in walk callback
- [ ] 2.2 Modify `internal/infrastructure/detector/php/detector.go` — same pattern as 2.1
- [ ] 2.3 Modify `internal/infrastructure/detector/swift/detector.go` — same pattern as 2.1
- [ ] 2.4 Modify `internal/infrastructure/detector/go/detector.go` — same pattern as 2.1
- [ ] 2.5 Modify `internal/infrastructure/detector/typescript/detector.go` — same pattern as 2.1
- [ ] 2.6 Modify `internal/infrastructure/detector/python/detector.go` — same pattern as 2.1
- [ ] 2.7 Modify `internal/infrastructure/detector/java/detector.go` — same pattern as 2.1
- [ ] 2.8 Modify `internal/infrastructure/detector/rust/detector.go` — same pattern as 2.1
- [ ] 2.9 Modify `internal/infrastructure/detector/csharp/detector.go` — same pattern as 2.1
- [ ] 2.10 Modify `internal/infrastructure/detector/ruby/detector.go` — same pattern as 2.1
- [ ] 2.11 Update `internal/ports/detector.go` — pass `*domain.ArxIgnore` through `ExtractImports` call chain

## Phase 3: Man Pages

- [x] 3.1 Create `cmd/arx/man.go` with `arx man` Cobra subcommand, `--output` flag (default `./man/`), calls `rootCmd.GenManTree()` with section 1 header
- [x] 3.2 Register `manCmd` in `cmd/arx/root.go` (follow `completion.go` pattern)

## Phase 4: Project Hygiene

- [x] 4.1 Create `.editorconfig` at project root with Go (tabs) and markdown/yaml/json (spaces, 2-indent) settings
- [x] 4.2 Create `.github/dependabot.yml` with `gomod` and `github-actions` ecosystems, weekly schedule

## Phase 5: Fuzz Tests

- [ ] 5.1 Create `internal/infrastructure/detector/kotlin/parser_fuzz_test.go` with `FuzzKotlinParse`, seed corpus of 3-5 valid Kotlin import lines
- [ ] 5.2 Create `internal/infrastructure/detector/php/parser_fuzz_test.go` with `FuzzPhpParse`, seed corpus of 3-5 valid PHP import lines
- [ ] 5.3 Create `internal/infrastructure/detector/swift/parser_fuzz_test.go` with `FuzzSwiftParse`, seed corpus of 3-5 valid Swift import lines

## Phase 6: Documentation & Verification

- [ ] 6.1 Update `README.md` with v0.15.0 features (man pages, .arxignore, fuzz coverage)
- [ ] 6.2 Update roadmap with v0.15.0 completion
- [ ] 6.3 Run `go test ./...` — all tests pass
- [ ] 6.4 Run `go test -fuzz=FuzzKotlinParse -fuzztime=5s`, `FuzzPhpParse`, `FuzzSwiftParse` — no crashes
