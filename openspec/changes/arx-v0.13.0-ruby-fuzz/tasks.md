# Tasks: Ruby Detector + Parser Fuzz Tests

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~500–650 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Ruby detector) → PR 2 (fuzz tests + README) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Ruby detector + tests + fixture + registry | PR 1 | Self-contained; can review independently |
| 2 | Parser fuzz tests (Java/C#/Rust/Ruby) + README | PR 2 | Depends on PR 1 only for Ruby fuzz test |

## Phase 1: Ruby Detector Skeleton

- [x] 1.1 Create `internal/infrastructure/detector/ruby/detector.go` with `RubyDetector` struct, `New()`, `Name()` returning `"ruby"`, `Detect()` via `os.Stat("Gemfile")`, `shouldSkip()` (skip `vendor`, `spec`, `test`, `.git`, `node_modules`, `.idea`, `.vscode`), `shouldSkipPath()`, `FindRubyFiles()` (walk root + `lib/`, skip `_test.rb`/`_spec.rb`, skip vendor/spec/test dirs), `ExtractImports()` following Rust/C# pattern with ctx cancellation and layerMap
- [x] 1.2 Create `internal/infrastructure/detector/ruby/parser.go` with 4 regex patterns (`requireRelativePattern`, `requireAllPattern`, `requireExpandPattern`, `requirePattern`), `extractImportsFromLine()` (skip comments/empty, try patterns in order, skip bare `require` as external), `isExternal()` (returns true for bare `require` without `relative`/`all`/`File.expand_path`)
- [x] 1.3 Modify `internal/infrastructure/detector/registry.go` — import `rubydetector`, append `rubydetector.New()` to `GetDetectors()`

## Phase 2: Ruby Detector Tests + Fixtures

- [x] 2.1 Create `internal/infrastructure/detector/ruby/detector_test.go` — table-driven tests for `Name()`, `Detect()` (Gemfile present/absent), `shouldSkipPath()`, `FindRubyFiles()` (skips spec/test/vendor, skips `_test.rb`/`_spec.rb`), `ExtractImports()` with layers, ctx cancellation
- [x] 2.2 Create `internal/infrastructure/detector/ruby/parser_test.go` — table-driven tests for `extractImportsFromLine()` covering all 4 require variants, comments, empty lines, inline comments, edge cases; `isExternal()` table tests
- [x] 2.3 Create `test/fixtures/ruby-project/Gemfile` (minimal), `test/fixtures/ruby-project/lib/domain/order.rb` (with `require_relative`), `test/fixtures/ruby-project/lib/infrastructure/order_repo.rb`, `test/fixtures/ruby-project/lib/application/order_service.rb` (with cross-layer `require_relative`), `test/fixtures/ruby-project/spec/order_spec.rb` (should be skipped)
- [x] 2.4 Create integration test in `detector_test.go` — `ExtractImports()` on `test/fixtures/ruby-project/` verifying cross-layer dependencies resolve correctly

## Phase 3: Parser Fuzz Tests

- [ ] 3.1 Create `internal/infrastructure/detector/java/parser_fuzz_test.go` — `FuzzJavaParse(f *testing.F)` seeding with valid `import`/`package` lines, following `config_fuzz_test.go` pattern
- [ ] 3.2 Create `internal/infrastructure/detector/csharp/parser_fuzz_test.go` — `FuzzCSharpParse(f *testing.F)` seeding with valid `using` directives
- [ ] 3.3 Create `internal/infrastructure/detector/rust/parser_fuzz_test.go` — `FuzzRustParse(f *testing.F)` seeding with valid `use` statements
- [ ] 3.4 Create `internal/infrastructure/detector/ruby/parser_fuzz_test.go` — `FuzzRubyParse(f *testing.F)` seeding with all 4 require variants

## Phase 4: Polish

- [ ] 4.1 Update `README.md` — add Ruby row to Supported Languages table (`Ruby | Regex | require patterns | v0.13.0`), update "Why Arx?" cross-language list to include Ruby
- [ ] 4.2 Run `go test ./...` and `go test -fuzz=FuzzRubyParse -fuzztime=5s ./internal/infrastructure/detector/ruby/` + all 3 existing parser fuzz tests to verify no panics
