# Design: Arx v0.14.0 — PHP & Swift Detectors

## Technical Approach

Add two language detectors (PHP, Swift) following the exact Ruby/Rust detector pattern: two-file split (`detector.go` + `parser.go`), registry registration, `ports.Detector` interface implementation. Both are additive — no domain or port interfaces change. Each detector activates only when its project manifest file exists in the project root.

## Architecture Decisions

| Decision | Options | Tradeoff | Decision |
|----------|---------|----------|----------|
| File structure | Single file vs detector+parser split | Split matches Rust/Ruby pattern, keeps regexes isolated | Two files: `detector.go` + `parser.go` |
| PHP detection | `composer.json` vs `*.php` file scan | `composer.json` is unambiguous, fast single stat call | `composer.json` in project root |
| PHP source dirs | `src/` only vs root + `src/` | PSR-4 allows code at root AND `src/`; both are common | `["", "src/"]` |
| PHP external skip | `vendor/` directory skip | Composer installs deps to `vendor/`; standard convention | Skip `vendor/` directory |
| Swift detection | `Package.swift` vs `*.swift` file scan | `Package.swift` is SPM manifest, unambiguous | `Package.swift` in project root |
| Swift source dirs | `Sources/` (SPM) vs project root | SPM mandates `Sources/` for package targets | `["Sources/"]` |
| Swift external skip | System module whitelist vs directory skip | Swift system modules are imported by name, not path | Whitelist: Foundation, UIKit, SwiftUI, AppKit, CoreData, Combine, Dispatch, os, CoreGraphics, QuartzCore |

## Data Flow

```
GetDetectors() ──→ PHPDetector (new entry)
                   SwiftDetector (new entry)

PHPDetector:
  Detect() ──→ os.Stat("composer.json") ──→ bool
  ExtractImports()
    ├── FindPHPFiles() ──→ filepath.Walk + skip vendor/tests/*Test.php
    │
    └── parseFile() ──→ regex extractImportsFromLine()
                          ├── use Namespace\Class;           → local/external
                          ├── use Namespace\Class as Alias;  → local/external
                          ├── use function Namespace\fn;     → local/external
                          ├── use const Namespace\CONST;     → local/external
                          └── require_once __DIR__ . '/path' → local

SwiftDetector:
  Detect() ──→ os.Stat("Package.swift") ──→ bool
  ExtractImports()
    ├── FindSwiftFiles() ──→ filepath.Walk + skip Tests/*Tests.swift
    │
    └── parseFile() ──→ regex extractImportsFromLine()
                          ├── import Module                  → local/external
                          ├── import struct Module.Type      → local/external
                          └── @_exported import Module       → local/external
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/infrastructure/detector/php/detector.go` | Create | `PHPDetector` struct, `Name()`, `Detect()`, `ExtractImports()`, `FindPHPFiles()`, `resolveImport()`, `parseFile()`, `shouldSkip()`, `shouldSkipPath()` |
| `internal/infrastructure/detector/php/parser.go` | Create | Regex patterns for `use`, `use function`, `use const`, `require_once`, `extractImportsFromLine()`, `isExternalDependency()` |
| `internal/infrastructure/detector/php/detector_test.go` | Create | Unit tests for detector and parser |
| `internal/infrastructure/detector/php/parser_test.go` | Create | Table-driven tests for all regex patterns |
| `internal/infrastructure/detector/swift/detector.go` | Create | `SwiftDetector` struct, same methods as PHP |
| `internal/infrastructure/detector/swift/parser.go` | Create | Regex patterns for `import`, `import struct`, `@_exported import`, `extractImportsFromLine()`, `isExternalDependency()` |
| `internal/infrastructure/detector/swift/detector_test.go` | Create | Unit tests for detector and parser |
| `internal/infrastructure/detector/swift/parser_test.go` | Create | Table-driven tests for all regex patterns |
| `internal/infrastructure/detector/registry.go` | Modify | Import `phpdetector`, `swiftdetector`; append both to `GetDetectors()` |

## Interfaces / Contracts

**PHPDetector** implements `ports.Detector`:

```go
type PHPDetector struct {
    modulePrefix string
    sourceDirs   []string  // {"", "src/"}
}

func New() *PHPDetector

func (d *PHPDetector) Name() string                              // "php"
func (d *PHPDetector) Detect(ctx, projectRoot) (bool, error)     // os.Stat(composer.json)
func (d *PHPDetector) ExtractImports(ctx, projectRoot, layers) ([]domain.Dependency, error)
func (d *PHPDetector) FindPHPFiles(projectRoot) ([]string, error)
func (d *PHPDetector) parseFile(filePath, projectRoot string, layerMap) ([]domain.Dependency, error)
func (d *PHPDetector) resolveImport(importPath, filePath, projectRoot string, layerMap) string
```

**PHP parser regex patterns** (`parser.go`):

```go
// use Namespace\Class;
useStandardPattern = regexp.MustCompile(`^\s*use\s+([A-Za-z_][A-Za-z0-9_\\]+)\s*;\s*$`)

// use Namespace\Class as Alias;
useAliasPattern = regexp.MustCompile(`^\s*use\s+([A-Za-z_][A-Za-z0-9_\\]+)\s+as\s+[A-Za-z_][A-Za-z0-9_]*\s*;\s*$`)

// use function Namespace\fn;
useFunctionPattern = regexp.MustCompile(`^\s*use\s+function\s+([A-Za-z_][A-Za-z0-9_\\]+)\s*;\s*$`)

// use const Namespace\CONST;
useConstPattern = regexp.MustCompile(`^\s*use\s+const\s+([A-Za-z_][A-Za-z0-9_\\]+)\s*;\s*$`)

// require_once __DIR__ . '/path';
requireOncePattern = regexp.MustCompile(`^\s*require_once\s+__DIR__\s*\.\s*['"]([^'"]+)['"]\s*;\s*$`)
```

**PHP `isExternalDependency`**: skip anything under `vendor/` path prefix; `require_once` with relative paths is always local.

**SwiftDetector** implements `ports.Detector`:

```go
type SwiftDetector struct {
    modulePrefix string
    sourceDirs   []string  // {"Sources/"}
}

func New() *SwiftDetector

func (d *SwiftDetector) Name() string                              // "swift"
func (d *SwiftDetector) Detect(ctx, projectRoot) (bool, error)     // os.Stat(Package.swift)
func (d *SwiftDetector) ExtractImports(ctx, projectRoot, layers) ([]domain.Dependency, error)
func (d *SwiftDetector) FindSwiftFiles(projectRoot) ([]string, error)
func (d *SwiftDetector) parseFile(filePath, projectRoot string, layerMap) ([]domain.Dependency, error)
func (d *SwiftDetector) resolveImport(importPath, filePath, projectRoot string, layerMap) string
```

**Swift parser regex patterns** (`parser.go`):

```go
// import Foundation
importStandardPattern = regexp.MustCompile(`^\s*import\s+([A-Za-z_][A-Za-z0-9_]*)\s*$`)

// import struct Module.Type
importMemberPattern = regexp.MustCompile(`^\s*import\s+(struct|class|enum|protocol|typealias|let|var|func)\s+([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*$`)

// @_exported import Module
exportedImportPattern = regexp.MustCompile(`^\s*@_exported\s+import\s+([A-Za-z_][A-Za-z0-9_]*)\s*$`)
```

**Swift `isExternalDependency`**: skip system module whitelist:
`Foundation`, `UIKit`, `SwiftUI`, `AppKit`, `CoreData`, `Combine`, `Dispatch`, `os`, `CoreGraphics`, `QuartzCore`.

**Registry update** (`registry.go`):

```go
import (
    phpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/php"
    swiftdetector "github.com/pauvalls/arx/internal/infrastructure/detector/swift"
    // ... existing imports
)

func GetDetectors() []ports.Detector {
    return []ports.Detector{
        // ... existing entries
        phpdetector.New(),
        swiftdetector.New(),
    }
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit — PHP detector | `Detect()` returns true for `composer.json`, false otherwise | Table-driven with temp dirs |
| Unit — PHP parser | `extractImportsFromLine()` for all 5 patterns + comments + edge cases | Table-driven matching Ruby/Rust pattern |
| Unit — PHP resolve | `resolveImport()` maps `use` paths to layers via `shared.MatchImportToLayer` | Table-driven with mock layerMap |
| Unit — PHP file discovery | `FindPHPFiles()` skips `vendor/`, `*Test.php`, `tests/` | Temp fixture with mixed files |
| Unit — Swift detector | `Detect()` returns true for `Package.swift`, false otherwise | Table-driven with temp dirs |
| Unit — Swift parser | `extractImportsFromLine()` for all 3 patterns + comments + edge cases | Table-driven |
| Unit — Swift resolve | `resolveImport()` maps `import` paths to layers; skips system modules | Table-driven with mock layerMap |
| Unit — Swift file discovery | `FindSwiftFiles()` skips `Tests/`, `*Tests.swift` | Temp fixture with mixed files |
| Integration | Full `ExtractImports()` on fixture PHP + Swift projects | Fixture dirs with realistic source files |

## Migration / Rollout

No migration required. Both detectors are new registry entries — they won't activate unless their manifest file (`composer.json` / `Package.swift`) exists in the project root. Fully backward-compatible.

## Open Questions

- [ ] PHP `require`/`include` (without `_once`): should these also be captured? The proposal specifies `require_once` only. `require` and `include` are semantically identical for import detection purposes — adding them is a one-line regex change if needed during implementation.
- [ ] Swift `@_exported` is an underscore attribute (internal/unstable). Should it be captured? Yes — it indicates re-exported dependencies that affect layer boundaries, even if the attribute itself may change in future Swift versions.
