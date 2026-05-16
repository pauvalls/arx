# Design: Maturity Features (v0.20.0)

## Technical Approach

Four orthogonal features that improve DX and polish:

1. **JSON Schema**: A static `arx-schema.json` file at project root that documents the full Config struct. `arx init` adds a `$schema` field to generated `arx.yaml` so editors can offer autocomplete and validation.
2. **NO_COLOR Support**: Check `os.Getenv("NO_COLOR")` in the terminal output path. When set (to any value except `"0"`), strip all lipgloss styling and emit plain text.
3. **Smart Init**: After writing `arx.yaml`, check if the project is a git repo and append `.arx-cache/` and `.arx-baseline.json` to `.gitignore` if not already present.
4. **Verbose Check**: In `arx check --verbose`, after `RunDetectors` completes, print a per-detector status line showing name, ✓/✗, and reason (applicable/not applicable or error).

All changes are additive and backward-compatible.

## Architecture Decisions

| Decision | Chosen | Rejected | Rationale |
|----------|--------|----------|-----------|
| Schema location | Static file at project root (`arx-schema.json`) | Embedded in binary, generated from struct tags | Static file is editable by hand, visible to users, and works with editor tooling that reads `$schema` URLs or local paths. |
| Schema maintenance | Manually written JSON Schema | Code-generated from struct tags (e.g. go-jsonschema) | The config is stable and small (~15 fields). Manual avoids build-step complexity and keeps the schema as a living document. |
| NO_COLOR detection | Package-level init in `terminal.go` reads env once | Pass flag through every function signature | The env var is process-wide; reading once at init avoids plumbing a bool through the entire reporter chain. |
| NO_COLOR sentinel | Any value except `"0"` disables color | Only presence (non-empty) | Follows the NO_COLOR spec: set to any value means "no color", but `"0"` is the conventional opt-out. |
| .gitignore logic | Only when `.git` directory exists | Use `git rev-parse --git-dir` via exec | Checking for `.git` dir is simpler, no subprocess, and matches what most tools do. |
| Verbose detector output | Print to `os.Stderr` after `RunDetectors` returns | Modify `RunDetectors` to return per-detector results | Keeps the domain layer clean. The cmd layer owns CLI output formatting. |

## Data Flow

### 1. JSON Schema

```
arx init
    │
    ├──► Scan + Generate config (existing)
    ├──► Write arx.yaml (existing)
    │
    └──► NEW: Add "$schema": "./arx-schema.json" to config before marshal
              (only if output path is default "arx.yaml" or user didn't set --output)
```

### 2. NO_COLOR

```
Process start
    │
    └──► terminal.go init() reads os.Getenv("NO_COLOR")
              │
              ├──► NO_COLOR="" or "0" → colors enabled (existing behavior)
              └──► NO_COLOR=<anything else> → colors disabled
                        │
                        └──► All lipgloss.Style.Render() calls return plain text
```

### 3. Smart Init

```
arx init (after writing arx.yaml)
    │
    ├──► Check if .git directory exists in project root
    │         │
    │         └──► No → skip (not a git repo)
    │
    └──► Yes → read .gitignore (create if missing)
                  │
                  ├──► Check for ".arx-cache/" entry
                  │         └──► Missing → append
                  │
                  └──► Check for ".arx-baseline.json" entry
                            └──► Missing → append
```

### 4. Verbose Check

```
arx check --verbose
    │
    ├──► Load config (existing verbose output)
    ├──► RunDetectors (existing)
    │         │
    │         └──► NEW: After return, print per-detector status:
    │                   ✓ go     (applicable, 42 deps extracted)
    │                   ✗ python (not applicable)
    │                   ✗ typescript (error: tsconfig not found)
    │
    └──► Evaluate + Report (existing)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `arx-schema.json` | Create | JSON Schema v7 covering all Config fields (version, layers, rules, language_overrides, exclude, severity_config, max_violations, severity_mapping). |
| `cmd/arx/init.go` | Modify | After writing config, call `ensureGitignoreEntries(projectRoot)`. Add `$schema` field to config before marshal (default path only). |
| `cmd/arx/check.go` | Modify | After `RunDetectors` returns, if `--verbose`, print per-detector status lines. Requires a small wrapper around detector execution to capture per-detector results. |
| `internal/infrastructure/output/terminal.go` | Modify | Add package-level `var noColor bool` set in `init()`. Guard all lipgloss style usage behind it. When `noColor` is true, use `lipgloss.NewStyle()` (no styling) or plain `fmt` calls. |
| `internal/application/check.go` | Modify | Add `RunDetectorsWithStatus` that returns `([]Dependency, []DetectorStatus, error)` where `DetectorStatus` holds name, applicable bool, depCount, and error. |
| `internal/ports/detector.go` | Modify | No changes needed — existing `Name()` method is sufficient. |

## Interfaces / Contracts

### JSON Schema (`arx-schema.json`)

Covers the full `Config` struct:

```jsonc
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Arx Configuration",
  "type": "object",
  "required": ["version", "layers", "rules"],
  "properties": {
    "version": { "type": "string" },
    "layers": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "paths"],
        "properties": {
          "name": { "type": "string" },
          "paths": { "type": "array", "items": { "type": "string" } },
          "description": { "type": "string" },
          "tags": { "type": "array", "items": { "type": "string" } }
        }
      }
    },
    "rules": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id"],
        "properties": {
          "id": { "type": "string" },
          "from": { "type": "string" },
          "to": { "type": "array", "items": { "type": "string" } },
          "type": { "enum": ["Cannot", "Must", "Can", "MustNotCircular"] },
          "severity": { "enum": ["error", "warning", "info"] },
          "explanation": { "type": "string" },
          "pattern": { "type": "string" },
          "check": { "type": "string" },
          "template": { "type": "string" },
          "params": { "type": "object" },
          "overrides": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "path": { "type": "string" },
                "severity": { "enum": ["error", "warning", "info"] },
                "enabled": { "type": "boolean" }
              }
            }
          },
          "exclude": { "type": "array", "items": { "type": "string" } }
        }
      }
    },
    "language_overrides": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "properties": {
          "extensions": { "type": "array", "items": { "type": "string" } },
          "comment": { "type": "string" },
          "import": { "type": "string" }
        }
      }
    },
    "exclude": { "type": "array", "items": { "type": "string" } },
    "severity_config": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "properties": {
          "fail_build": { "type": "boolean" },
          "show_in_ui": { "type": "boolean" }
        }
      }
    },
    "max_violations": { "type": "integer", "minimum": 0 },
    "severity_mapping": {
      "type": "object",
      "additionalProperties": { "type": "string" }
    }
  }
}
```

### `$schema` injection in `arx init`

When the output path is the default (`arx.yaml`) or a relative path ending in `arx.yaml`, set `config.Version`-adjacent field:

```go
// In runInit, after service.Init() or service.InitWithPreset() returns config:
if isDefaultConfigPath(outputPath) {
    config.Schema = "./arx-schema.json"  // new field on Config
}
```

This requires adding a `Schema string` field to `domain.Config` with yaml/json tag `"$schema"`.

### NO_COLOR in terminal.go

```go
package output

var noColor bool

func init() {
    v := os.Getenv("NO_COLOR")
    noColor = v != "" && v != "0"
}

// In Report(): if noColor, use plain fmt.Println instead of lipgloss.Render()
// Helper: style(s lipgloss.Style, text string) string {
//     if noColor { return text }
//     return s.Render(text)
// }
```

### Detector Status for verbose check

```go
// internal/application/check.go
type DetectorStatus struct {
    Name        string
    Applicable  bool
    DepCount    int
    Err         error
}

func RunDetectorsWithStatus(ctx context.Context, projectRoot string, layers []domain.Layer, detectors []ports.Detector) ([]domain.Dependency, []DetectorStatus, error) {
    // Same errgroup logic as RunDetectors, but collects per-detector status
}
```

### Smart init gitignore helper

```go
// cmd/arx/init.go
func ensureGitignoreEntries(projectRoot string) error {
    gitDir := filepath.Join(projectRoot, ".git")
    if _, err := os.Stat(gitDir); os.IsNotExist(err) {
        return nil // not a git repo
    }

    gitignorePath := filepath.Join(projectRoot, ".gitignore")
    entries := []string{".arx-cache/", ".arx-baseline.json"}
    // Read existing, check for each entry, append missing
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `arx-schema.json` is valid JSON Schema | Parse with `github.com/xeipuuv/gojsonschema` or similar in a test |
| Unit | `$schema` field appears in generated `arx.yaml` | Test `runInit` with default output path, verify field present |
| Unit | `$schema` field absent with custom `--output` path | Test `runInit -o custom.yaml`, verify field absent |
| Unit | NO_COLOR env var disables styling | Set env in test, verify `noColor` is true, verify output has no ANSI codes |
| Unit | NO_COLOR=0 keeps colors enabled | Set env to "0", verify `noColor` is false |
| Unit | NO_COLOR unset keeps colors enabled | Unset env, verify `noColor` is false |
| Unit | `.gitignore` entries appended when missing | Create temp git repo, run init, verify entries present |
| Unit | `.gitignore` entries not duplicated | Run init twice, verify no duplicate entries |
| Unit | `.gitignore` skipped outside git repo | Create temp dir without `.git`, run init, verify no `.gitignore` created |
| Integration | `arx check --verbose` shows detector status | Run against a Go project, verify stdout/stderr contains detector lines |
| Integration | `arx check --verbose` shows ✗ for non-applicable detectors | Run against a project with only Go, verify python/typescript show ✗ |

## Migration / Rollout

No migration required. All changes are additive:
- Existing `arx.yaml` files without `$schema` work identically.
- Existing terminal output is unchanged unless `NO_COLOR` is set.
- `.gitignore` modification only happens on `arx init`, never on `arx check`.
- `--verbose` output is new; default `arx check` behavior is unchanged.

## Open Questions

- [ ] Should `$schema` use a remote URL (e.g. `https://arx.dev/schema/v1/arx-schema.json`) instead of a local relative path? **Recommendation: local `./arx-schema.json`** for v0.20.0 — no hosting infrastructure needed. Can add remote URL later.
- [ ] Should smart-init also add `.arx-cache/` patterns for the old cache format? **No** — only the current cache directory and baseline file are relevant.
- [ ] Should verbose detector output include timing information? **Not in v0.20.0** — keep it simple. Can add in a future iteration if users request it.
