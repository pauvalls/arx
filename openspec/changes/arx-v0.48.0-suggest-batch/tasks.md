# v0.48 Suggest Batch + Conflict Detection — Tasks

## Phase 1: Core Types & Domain

- [x] T-01: Create `internal/domain/conflict.go` with Conflict, HunkRange types
- [x] T-02: Create `internal/application/conflict.go` with hunk-range conflict detection
- [x] T-03: Create `internal/application/rollback.go` with RollbackService
- [x] T-04: Modify `internal/application/suggest.go` for violation-ID based backup scheme

## Phase 2: CLI Features

- [x] T-05: Add `--all`, `--dry-run` flags to `cmd/arx/suggest.go`
- [x] T-06: Add staged review loop (y/N/s/e/q) with conflict warnings
- [x] T-07: Create `cmd/arx/rollback.go` with rollback command
- [x] T-08: Rollback instructions printed after suggest apply

## Phase 3: Explain Integration

- [x] T-09: Add `--suggest` flag to `cmd/arx/explain.go`
- [x] T-10: Inline fix suggestion in explain output

## Phase 4: Integration Tests & Legacy Support

- [x] T-11: Suggest batch integration test (dry-run, apply, rollback)
- [x] T-12: Conflict resolution integration test
- [x] T-13: Legacy backup compat for rollback command
