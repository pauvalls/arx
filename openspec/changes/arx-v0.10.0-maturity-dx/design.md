# Design: Arx v0.10.0 — Maturity & Developer Experience

## Technical Approach

Eight additive capabilities to mature the CLI and improve developer experience: (1) diagram CLI command reusing existing application service with new Mermaid output format, (2) shell completion via Cobra's built-in generation, (3) config validation command, (4) doctor command for diagnostics, (5) C# detector following the exact Rust detector pattern, (6) JUnit XML output format, (7) GitHub Actions annotations output, (8) build infrastructure with Makefile, changelog, and deprecation fixes. All features are independent and can be implemented in parallel.

## Architecture Decisions

### Decision: diagram CLI as thin wrapper over DiagramService

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Create new diagram-specific service | Duplicates existing logic in `application/diagram.go` | Rejected |
| Reuse `DiagramService` with new CLI command | Single source of truth, consistent behavior | **Chosen** |
| Merge diagram into check command | Violates single responsibility; diagram is distinct use case | Rejected |

The `DiagramService` already exists and handles dependency extraction. The CLI command only adds format selection and output routing.

### Decision: Mermaid as third output format alongside ASCII and DOT

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Mermaid flowchart (`flowchart TD`) | Human-readable, GitHub-native rendering, no external tools | **Chosen** |
| Mermaid class diagram | More verbose, less suited for dependency graphs | Rejected |
| PlantUML | Requires external renderer, less GitHub-friendly | Rejected |

Mermaid flowcharts render natively in GitHub Markdown and most documentation tools. Simpler than DOT for basic dependency visualization.

### Decision: Shell completion using Cobra's built-in generation

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Custom completion scripts | Full control but duplicates Cobra's battle-tested logic | Rejected |
| Cobra's `GenBashCompletion`, `GenZshCompletion`, etc. | Maintained upstream, supports all major shells | **Chosen** |

Cobra provides `GenBashCompletion()`, `GenZshCompletion()`, `GenFishCompletion()`, `GenPowerShellCompletion()`. One subcommand per shell, each writes to stdout.

### Decision: config-validate as separate command (not flag on check)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `arx check --validate-only` | Hidden functionality, unclear purpose | Rejected |
| `arx config validate` subcommand | Explicit intent, discoverable via `arx --help` | **Chosen** |

Config validation is a distinct operation from architecture checking. Users should be able to validate config without running a full audit.

### Decision: doctor command as diagnostic aggregator

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Separate checks for each concern | Multiple commands, fragmented output | Rejected |
| Single `doctor` command with multiple checks | One-stop diagnostics, like `brew doctor` | **Chosen** |

Developers want a single command to verify their setup: version, project root, config, detectors. Aggregated report with clear pass/fail per check.

### Decision: CSharpDetector mirrors RustDetector exactly

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Custom C# detection logic | Diverges from established pattern | Rejected |
| Reuse Go import graph analysis | Wrong semantics — C# uses `using` statements, not package paths | Rejected |
| Exact mirror of Rust detector structure | Same `Detect()` / `ExtractImports()` / `parseFile()` / `resolveImport()` / `isExternalDependency()` flow | **Chosen** |

C# `using` statements map directly to Rust's `use` statements. Same regex-based parsing, same external dependency filtering pattern.

### Decision: JUnit output as XML marshaling (not string building)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Manual XML string concatenation | Error-prone, no escaping, hard to maintain | Rejected |
| `encoding/xml` marshaling | Type-safe, proper escaping, Go idiomatic | **Chosen** |

Go's `encoding/xml` handles escaping, nesting, and formatting. Define structs with xml tags, marshal to bytes.

### Decision: GitHub Annotations as line-based output (not structured)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| JSON structure for annotations | Requires parsing by GitHub, not standard format | Rejected |
| Workflow command format (`::error file=X,line=N::Z`) | GitHub Actions standard, parsed automatically | **Chosen** |

GitHub Actions parses workflow commands from stdout/stderr. One line per violation in format `::error file=path,line=num,title=title::message`.

### Decision: Deprecation fixes as part of build-infrastructure

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Separate change for deprecations | Fragmented work, same codebase touch | Rejected |
| Bundle with Makefile and CHANGELOG | Single "maturity" theme, logical grouping | **Chosen** |

`strings.Title()` deprecated in Go 1.18, `filepath.HasPrefix()` doesn't exist (common mistake for `strings.HasPrefix()`). Fix both while establishing build conventions.

## Data Flow

### diagram command

```
cmd/arx/diagram.go
  → diagramCmd with flags: --format (ascii|dot|mermaid), --output
  → DiagramService.Generate(projectRoot, layers, config)
  → switch format:
      ascii → output.GenerateASCII(result)
      dot → output.GenerateDOT(result)
      mermaid → output.GenerateMermaid(result) [NEW]
  → write to file (--output) or stdout
```

### completion command

```
cmd/arx/completion.go
  → completionCmd (hidden)
    → bash subcommand → cobra.GenBashCompletion(os.Stdout)
    → zsh subcommand → cobra.GenZshCompletion(os.Stdout)
    → fish subcommand → cobra.GenFishCompletion(os.Stdout)
    → powershell subcommand → cobra.GenPowerShellCompletion(os.Stdout)
```

### config-validate command

```
cmd/arx/config.go
  → configValidateCmd with --path flag
  → configReader.Read(path)
  → configReader.Validate(config)
  → terminal output: "✓ Config valid" or "✗ Error: ..."
```

### doctor command

```
cmd/arx/doctor.go
  → doctorCmd
  → DoctorService.Check(projectRoot) [NEW service]
    → check 1: project root exists
    → check 2: config file exists
    → check 3: detectors find files
    → check 4: version info
    → check 5: git status (if repo)
  → terminal output with ✓/✗ per check
```

### C# detector

```
internal/infrastructure/detector/csharp/detector.go
  → CSharpDetector.Detect(): look for .csproj or .sln files
  → CSharpDetector.ExtractImports():
      walk src/ or default project structure
      parseFile():
        extractImportsFromLine():
          using Namespace;                    (standard)
          using static Namespace.Class;       (static import)
          using Alias = Namespace.Type;       (alias)
      resolveImport():
        skip external: System.*, Microsoft.*
        match against layers
```

### JUnit output

```
internal/infrastructure/output/junit.go [NEW]
  → JUnitReporter.Report(violations, format)
  → build XML structure:
      <testsuites>
        <testsuite name="arx" tests=N failures=M>
          <testcase name="violation-i" file="X" line="N">
            <failure message="..." type="architecture"/>
          </testcase>
        </testsuite>
      </testsuites>
  → xml.MarshalIndent() → stdout
```

### GitHub Annotations output

```
internal/infrastructure/output/gh_annotations.go [NEW]
  → GitHubAnnotationsReporter.Report(violations, format)
  → for each violation:
      fmt.Printf("::error file=%s,line=%d,title=%s::%s\n",
        file, line, title, message)
```

### Build infrastructure

```
Makefile [NEW]
  → build: go build -o arx ./cmd/arx
  → test: go test ./...
  → lint: golangci-lint run
  → clean: rm -f arx

CHANGELOG.md [NEW]
  → releases from v0.1.0 to v0.10.0
  → format: date, version, features/fixes

Deprecation fixes:
  → strings.Title() → cases.Title(language.Und)
  → filepath.HasPrefix() → strings.HasPrefix() [fix non-existent function]
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/arx/diagram.go` | Create | Cobra command for diagram generation with --format and --output flags |
| `internal/infrastructure/output/mermaid.go` | Create | Mermaid flowchart generator from DiagramResult |
| `cmd/arx/completion.go` | Create | Shell completion command with bash/zsh/fish/powershell subcommands |
| `cmd/arx/config.go` | Create | Config validation command with --path flag |
| `cmd/arx/doctor.go` | Create | Doctor command for project diagnostics |
| `internal/application/doctor.go` | Create | DoctorService with checks for root, config, detectors, version, git |
| `internal/infrastructure/detector/csharp/detector.go` | Create | CSharpDetector — Detect() via .csproj/.sln, ExtractImports() via using statements |
| `internal/infrastructure/detector/csharp/parser.go` | Create | Regex patterns for C# using, using static, using alias |
| `internal/infrastructure/detector/csharp/detector_test.go` | Create | Tests for C# detector |
| `internal/infrastructure/detector/csharp/parser_test.go` | Create | Tests for C# parser regex |
| `internal/infrastructure/detector/registry.go` | Modify | Import and register `csharp.New()` |
| `internal/infrastructure/output/junit.go` | Create | JUnit XML reporter using encoding/xml |
| `internal/infrastructure/output/gh_annotations.go` | Create | GitHub Actions workflow command output |
| `internal/ports/output.go` | Create | OutputFormat constants and Reporter interface (missing file) |
| `Makefile` | Create | Build, test, lint, clean targets |
| `CHANGELOG.md` | Create | Release history from v0.1.0 to v0.10.0 |
| `cmd/arx/*.go` | Modify | Fix strings.Title() → cases.Title(), filepath.HasPrefix() → strings.HasPrefix() |

## Interfaces / Contracts

```go
// cmd/arx/diagram.go
var diagramCmd = &cobra.Command{
    Use:   "diagram [path]",
    Short: "Generate architecture diagram",
    Flags:
        --format string   ascii|dot|mermaid (default "ascii")
        --output string   Output file path (default stdout)
}

// internal/infrastructure/output/mermaid.go
func GenerateMermaid(result *application.DiagramResult) string
    // Returns Mermaid flowchart syntax:
    // flowchart TD
    //   A[Layer1] --> B[Layer2]
    //   style A fill:#lightblue
    // Violations marked with red edges

// cmd/arx/completion.go
var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate shell completion scripts",
}
// Subcommands call respective cobra.Gen*Completion(os.Stdout)

// cmd/arx/config.go
var configValidateCmd = &cobra.Command{
    Use:   "validate",
    Short: "Validate configuration file",
    Flags:
        --path string   Config file path (default "arx.yaml")
}

// internal/application/doctor.go
type DoctorService struct {
    detectors []ports.Detector
    version   string
}

func (s *DoctorService) Check(projectRoot string) DoctorResult

type DoctorResult struct {
    ProjectRoot   CheckResult
    ConfigExists  CheckResult
    DetectorsWork CheckResult
    Version       string
    GitStatus     string
}

type CheckResult struct {
    OK      bool
    Message string
}

// internal/infrastructure/detector/csharp/detector.go
type CSharpDetector struct {
    modulePrefix string
    sourceDirs   []string
}

func New() *CSharpDetector
func (d *CSharpDetector) Name() string           // "csharp"
func (d *CSharpDetector) Detect(ctx, root) bool  // .csproj or .sln exists
func (d *CSharpDetector) ExtractImports(ctx, root, layers) ([]Dependency, error)

// C# regex patterns (parser.go):
//   using\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s*;           — standard using
//   using\s+static\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s*;  — static import
//   using\s+([a-zA-Z_]+)\s*=\s*([a-zA-Z_][a-zA-Z0-9_.]+)\s*;  — alias
// External skip prefixes: System.*, Microsoft.*

// internal/infrastructure/output/junit.go
type JUnitReporter struct{}

func (r *JUnitReporter) Report(violations []domain.Violation, format ports.OutputFormat) error
    // XML structure:
    // <testsuites tests=N failures=M>
    //   <testsuite name="arx">
    //     <testcase name="violation-0" file="X" line="N">
    //       <failure message="..." type="architecture"/>
    //     </testcase>
    //   </testsuite>
    // </testsuites>

// internal/infrastructure/output/gh_annotations.go
type GitHubAnnotationsReporter struct{}

func (r *GitHubAnnotationsReporter) Report(violations []domain.Violation, format ports.OutputFormat) error
    // Output format per violation:
    // ::error file={file},line={line},title={title}::{message}

// internal/ports/output.go
type OutputFormat string
const (
    OutputFormatTerminal      OutputFormat = "terminal"
    OutputFormatJSON          OutputFormat = "json"
    OutputFormatSARIF         OutputFormat = "sarif"
    OutputFormatMarkdown      OutputFormat = "markdown"
    OutputFormatJUnit         OutputFormat = "junit"
    OutputFormatGHAnnotations OutputFormat = "gh-annotations"
)

type Reporter interface {
    Report(violations []domain.Violation, format OutputFormat) error
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | diagram command flags | Table-driven: --format ascii/dot/mermaid, --output to file |
| Unit | Mermaid output generation | Verify flowchart syntax, layer colors, violation edges |
| Unit | completion subcommands | Each shell generates valid completion script (smoke test) |
| Unit | config-validate | Valid config → success message; invalid → error with details |
| Unit | DoctorService checks | Mock each check: project root, config, detectors, git |
| Unit | CSharpDetector.Detect | Fake .csproj/.sln → true; empty dir → false |
| Unit | CSharpParser regex | Table-driven: `using System;`, `using static Math;`, `using Alias = Type;`, skip System.*, Microsoft.* |
| Unit | CSharpDetector.resolveImport | Resolve `MyApp.Domain.Model` to layer; skip `System.Collections.Generic` |
| Unit | JUnit XML marshaling | Verify XML structure, proper escaping, test counts |
| Unit | GitHub Annotations format | Verify workflow command syntax per violation |
| Integration | C# detector on real .csproj | ArxFakeProject with .csproj + src/*.cs → verify dependencies |
| Integration | diagram end-to-end | Real project → diagram command → verify output format |
| Integration | doctor on real project | Run doctor, verify all checks pass/fail appropriately |

C# detector tests mirror Rust detector test structure exactly. Output format tests are table-driven unit tests.

## Migration / Rollout

No migration required. All features are additive:
- `diagram` command is new — no existing functionality affected
- `completion` command is new — opt-in via `arx completion [shell]`
- `config validate` is new — separate command
- `doctor` command is new — diagnostic only
- C# detector is new registry entry — won't activate unless .csproj/.sln exists
- JUnit and GitHub Annotations are new output formats — selected via --format flag
- Makefile and CHANGELOG are new files — no code impact
- Deprecation fixes are internal — no API changes

## Open Questions

- [x] Should diagram command support layer filtering? **No** — keep v0.10.0 scoped. Additive feature for future version.
- [x] Should C# detector parse .csproj for project structure? **No** — use default `src/` like other detectors. .csproj parsing is complex (SDK-style vs old-style).
- [x] Should doctor command exit non-zero on warnings? **No** — doctor is informational. Exit 0 unless critical error (e.g., cannot read directory).
- [x] Should JUnit output include test cases for violations only or all files? **Violations only** — JUnit is for CI failure reporting, not full inventory.
- [x] Should GitHub Annotations use ::warning for non-error severities? **Yes** — map domain.Severity to workflow commands: error→::error, warning→::warning, info→::notice.
