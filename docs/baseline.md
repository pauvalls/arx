# Baseline

Use baselines to adopt arx in existing projects without fixing every violation at once.

## Problem

Running `arx check` on a legacy project often shows dozens or hundreds of violations.
Requiring teams to fix everything before enabling CI creates friction and delays adoption.

## Solution

A **baseline** captures the current violations and suppresses them on subsequent runs.
Only **new** violations (not in the baseline) are reported.

```bash
# 1. Create a baseline from current violations
arx baseline
# → Baseline created with 47 violations

# 2. Run check — only NEW violations are reported
arx check
# → 0 new violations ✅ (47 suppressed)

# 3. Someone introduces a new violation
arx check
# → 1 new violation ❌
```

## Commands

### `arx baseline`

Creates `.arx-baseline.json` in the project root:

```bash
arx baseline                     # Create baseline
arx baseline --reset             # Regenerate from current state
arx baseline --output custom.json  # Custom path
```

### Baseline-aware check

`arx check` automatically loads `.arx-baseline.json` if present:

```bash
arx check                        # Uses baseline (reports only new violations)
arx check --no-baseline          # Ignore baseline, report all violations
arx check --verbose              # Show suppressed count
```

## How It Works

Each violation is identified by a **fingerprint**: `rule_id:source_layer:target_layer:import`.

- The baseline stores fingerprints, not file paths or line numbers
- This means violations are recognized even if files move or lines shift
- When `arx.yaml` changes, the baseline is flagged as **stale** (but still usable with a warning)

## CI Integration

In CI mode, baseline is respected automatically:

```bash
arx check --ci
# → Exit 0 if no NEW violations
# → Exit 1 if NEW violations found
```

## Best Practices

1. **Create baseline on clean slate** — After running `arx check` once, create baseline
2. **Commit baseline** — `.arx-baseline.json` should be in version control
3. **Periodically refresh** — Use `arx baseline --reset` after fixing bulk violations
4. **Never disable baseline CI** — Use overrides for per-path exceptions instead
