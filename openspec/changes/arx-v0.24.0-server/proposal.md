# Proposal: arx v0.24.0 — Embedded Web Dashboard

## Why

Developers want a live, visual view of their architecture health without re-running `arx check` manually. A lightweight web dashboard provides real-time violation tracking, coupling visualization, and debt scoring — turning arx from a CI-only tool into a continuous companion.

## What Changes

- New `arx server` command that starts an HTTP server
- REST API exposing status, violations, coupling, and debt data
- Single-page dashboard (HTML/CSS/JS) embedded in the binary
- File watching (fsnotify) to auto-refresh on code changes
- Periodic re-check every 30 seconds as fallback

## Impact

- **New files**: 6 (server command, HTTP handlers, dashboard, API, tests)
- **Modified files**: 0 (fully additive)
- **Dependencies**: None new (stdlib `net/http`, `html/template`, `embed`; existing `fsnotify`, `cobra`)
- **Breaking changes**: None

## Approach

Go stdlib only (`net/http`, `html/template`, `embed`). No external web framework. Dashboard is a single HTML file with vanilla CSS/JS, auto-refreshing via `fetch` polling.
