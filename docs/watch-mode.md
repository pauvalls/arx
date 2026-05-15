# Watch Mode

Get continuous architecture feedback while coding.

## Problem

Developers discover violations only when they run `arx check` manually or when CI fails.
By then, the violating code is written, committed, and context is lost.

## Solution

`arx check --watch` monitors file changes and re-runs checks automatically:

```bash
arx check --watch
# → 0 violations | watching for changes... (debounce: 500ms)

# Edit a file that introduces a violation...
# → +1 violation, -0 resolved in 0.8s
#   + domain/order.go:14 → infrastructure/postgres.go  NEW

# Fix the violation...
# → +0 violations, -1 resolved in 0.6s
#   - domain/order.go:14 → infrastructure/postgres.go  RESOLVED
```

## Usage

```bash
# Start watch mode
arx check --watch

# Custom debounce interval
arx check --watch --interval 2s

# With baseline support
arx baseline
arx check --watch

# Verbose — show each file change detected
arx check --watch --verbose
# → [WATCH] File changed: internal/domain/order.go

# JSON output per change
arx check --watch --format json > watch-output.json
```

## How It Works

1. **Initial check**: Runs `arx check` once on startup
2. **File watcher**: Uses `fsnotify` to monitor the project tree
3. **Debounce**: Waits 500ms (configurable) of inactivity before re-checking
4. **Diff**: Compares new violations against previous, shows only changes
5. **Graceful shutdown**: Ctrl+C or SIGINT stops cleanly

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--watch` | `false` | Enable watch mode |
| `--interval` | `500ms` | Debounce interval |

## Limitations

- Ignores `.git/`, `node_modules/`, `vendor/`, `target/`, `build/`
- Full re-check on each change (not incremental per-file)
- Performance depends on project size — use cache (`--no-cache` to disable)
