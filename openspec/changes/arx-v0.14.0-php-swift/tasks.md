# Tasks: Arx v0.14.0 — PHP & Swift Detectors

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~800–1000 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (PHP detector) → PR 2 (Swift detector) → PR 3 (polish + E2E) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | PHP detector + parser + tests + fixture | PR 1 | Self-contained; follows Ruby/Rust pattern |
| 2 | Swift detector + parser + tests + fixture | PR 2 | Self-contained; follows Ruby/Rust pattern |
| 3 | Registry update + README + E2E tests | PR 3 | Depends on PR 1 + PR 2 |

## Phase 1: PHP Detector

- [ ] 1.1 Create `internal/infrastructure/detector/php/detector.go` — `PHPDetector` struct with `sourceDirs = {"", "src/"}`, `New()`, `Name()` returning `"php"`, `Detect()` via `os.Stat("composer.json")`, `shouldSkip()` (skip `vendor`, `.git`, `node_modules`, `.idea`, `.vscode`, `tests`), `shouldSkipPath()`, `FindPHPFiles()` (walk root + `src/`, skip `vendor/` dir, skip `*Test.php` files), `ExtractImports()` following Ruby pattern with ctx cancellation and layerMap, `parseFile()` line-by-line with `extractImportsFromLine()`, `resolveImport()` (relative paths resolved from source file dir, namespace paths matched via `shared.MatchImportToLayer`), `resolveSourcePath()` (convert `Namespace\Class` to `Namespace/Class.php` and check src dirs)
- [ ] 1.2 Create `internal/infrastructure/detector/php/parser.go` — 5 regex patterns: `useStandardPattern` (`use Namespace\Class;`), `useAliasPattern` (`use Namespace\Class as Alias;`), `useFunctionPattern` (`use function Namespace\fn;`), `useConstPattern` (`use const Namespace\CONST;`), `requireOncePattern` (`require_once __DIR__ . '/path';`), `extractImportsFromLine()` (skip comments starting with `//` or `#`, strip inline `//` comments, try patterns in order), `isExternalDependency()` (skip anything under `vendor/` path prefix; `require_once` with relative paths is always local)
- [ ] 1.3 Create `internal/infrastructure/detector/php/detector_test.go` — table-driven tests for `Name()`, `Detect()` (composer.json present/absent), `shouldSkipPath()`, `FindPHPFiles()` (skips vendor/tests, skips `*Test.php`), `ExtractImports()` with layers, ctx cancellation
- [ ] 1.4 Create `internal/infrastructure/detector/php/parser_test.go` — table-driven tests for `extractImportsFromLine()` covering all 5 patterns (`use`, `use as`, `use function`, `use const`, `require_once`), comments (`//`, `#`), empty lines, inline comments, edge cases; `isExternalDependency()` table tests (vendor paths = external, relative require_once = local, namespace use = check vendor prefix)
- [ ] 1.5 Create `test/fixtures/php-project/composer.json` (minimal), `test/fixtures/php-project/src/Domain/Order.php` (with `use` statements), `test/fixtures/php-project/src/Infrastructure/OrderRepository.php`, `test/fixtures/php-project/src/Application/OrderService.php` (with cross-layer `use` + `require_once __DIR__ . '/../Domain/Order.php'`), `test/fixtures/php-project/vendor/autoload.php` (should be skipped), `test/fixtures/php-project/tests/OrderTest.php` (should be skipped), `test/fixtures/php-project/arx.yaml`
- [ ] 1.6 Create integration test — `ExtractImports()` on `test/fixtures/php-project/` verifying cross-layer dependencies resolve correctly, vendor files skipped, test files skipped

## Phase 2: Swift Detector

- [ ] 2.1 Create `internal/infrastructure/detector/swift/detector.go` — `SwiftDetector` struct with `sourceDirs = {"Sources/"}`, `New()`, `Name()` returning `"swift"`, `Detect()` via `os.Stat("Package.swift")`, `shouldSkip()` (skip `.git`, `node_modules`, `.idea`, `.vscode`, `DerivedData`, `.build`), `shouldSkipPath()`, `FindSwiftFiles()` (walk `Sources/`, skip `Tests/` dir, skip `*Tests.swift` files), `ExtractImports()` following Ruby pattern with ctx cancellation and layerMap, `parseFile()` line-by-line with `extractImportsFromLine()`, `resolveImport()` (module names matched via `shared.MatchImportToLayer`), `resolveSourcePath()` (check `Sources/<Module>/<File>.swift` patterns)
- [ ] 2.2 Create `internal/infrastructure/detector/swift/parser.go` — 3 regex patterns: `importStandardPattern` (`import Module`), `importMemberPattern` (`import struct|class|enum|protocol|typealias|let|var|func Module.Type`), `exportedImportPattern` (`@_exported import Module`), `extractImportsFromLine()` (skip `//` comments, strip inline comments, try patterns in order), `isExternalDependency()` (skip system module whitelist: `Foundation`, `UIKit`, `SwiftUI`, `AppKit`, `CoreData`, `Combine`, `Dispatch`, `os`, `CoreGraphics`, `QuartzCore`)
- [ ] 2.3 Create `internal/infrastructure/detector/swift/detector_test.go` — table-driven tests for `Name()`, `Detect()` (Package.swift present/absent), `shouldSkipPath()`, `FindSwiftFiles()` (skips Tests/, skips `*Tests.swift`), `ExtractImports()` with layers, ctx cancellation
- [ ] 2.4 Create `internal/infrastructure/detector/swift/parser_test.go` — table-driven tests for `extractImportsFromLine()` covering all 3 patterns (`import`, `import struct`, `@_exported import`), comments, empty lines, inline comments, edge cases; `isExternalDependency()` table tests (system modules = external, custom modules = local)
- [ ] 2.5 Create `test/fixtures/swift-project/Package.swift` (minimal SPM manifest), `test/fixtures/swift-project/Sources/Domain/Order.swift` (with `import` statements), `test/fixtures/swift-project/Sources/Infrastructure/OrderRepository.swift`, `test/fixtures/swift-project/Sources/Application/OrderService.swift` (with cross-layer `import` + `@_exported import`), `test/fixtures/swift-project/Tests/OrderTests.swift` (should be skipped), `test/fixtures/swift-project/arx.yaml`
- [ ] 2.6 Create integration test — `ExtractImports()` on `test/fixtures/swift-project/` verifying cross-layer dependencies resolve correctly, test files skipped, system modules filtered

## Phase 3: Registry + Polish

- [ ] 3.1 Modify `internal/infrastructure/detector/registry.go` — import `phpdetector` and `swiftdetector`, append `phpdetector.New()` and `swiftdetector.New()` to `GetDetectors()`
- [ ] 3.2 Update `README.md` — add PHP row to Supported Languages table (`PHP | Regex | use/require_once | v0.14.0`), add Swift row (`Swift | Regex | import/@_exported | v0.14.0`), update "Why Arx?" cross-language list
- [ ] 3.3 Update `docs/roadmap.md` — add v0.14.0 section with PHP + Swift detectors checked, move PHP/Swift from "Future" to completed

## Phase 4: E2E Tests

- [ ] 4.1 Add `TestE2E_PHPProject` to `test/integration/e2e_test.go` — builds arx binary, runs `arx check` on `test/fixtures/php-project/`, verifies violations detected (exit code 1 with violation output)
- [ ] 4.2 Add `TestE2E_SwiftProject` to `test/integration/e2e_test.go` — same pattern as PHP, runs on `test/fixtures/swift-project/`
- [ ] 4.3 Run `go test ./...` — all existing tests pass, new PHP/Swift unit + integration + E2E tests pass
