# API Reference

Arx provides several API surfaces: REST API (via `arx server`), SSE events (for real-time dashboard updates), Webhook (for GitHub PR integration), and LSP protocol (for editor integration).

## REST API

The REST server is started with `arx server`. All endpoints are prefixed with `/api`.

### GET /api/health

Health check endpoint.

**Response:** `200 OK`
```json
{"status": "ok"}
```

---

### GET /api/status

Server status, version, violation count, and metrics.

**Response:** `200 OK`
```json
{
  "version": "0.57.0",
  "uptime": "2h15m30s",
  "last_check": "2026-05-21T12:00:00Z",
  "violation_count": 3,
  "violations_by_severity": {
    "error": 2,
    "warning": 1
  },
  "debt_score": 42,
  "check_error": "",
  "config_reloaded": false
}
```

---

### GET /api/violations

Current violations list.

**Response:** `200 OK`
```json
[
  {
    "id": "D-01",
    "rule_id": "domain-no-infra",
    "severity": "error",
    "file": "internal/domain/order.go",
    "line": 42,
    "source_layer": "domain",
    "target_layer": "infrastructure",
    "import": "github.com/example/internal/infrastructure/postgres",
    "message": "Domain layer must not depend on infrastructure layer"
  }
]
```

---

### GET /api/coupling

Coupling matrix between layers with percentage breakdown.

**Response:** `200 OK`
```json
[
  {
    "from": "domain",
    "to": "infrastructure",
    "count": 5,
    "percentage": 12.5
  },
  {
    "from": "application",
    "to": "domain",
    "count": 23,
    "percentage": 57.5
  }
]
```

---

### GET /api/debt

Technical debt score.

**Response:** `200 OK`
```json
{
  "total": 42,
  "trend": "stable",
  "trend_delta": 0,
  "breakdown": {
    "error": 20,
    "warning": 15,
    "info": 7
  }
}
```

---

### GET /api/metrics

Performance metrics from the last check.

**Response:** `200 OK`
```json
{
  "check_duration_ms": 234,
  "files_scanned": 142,
  "total_deps": 1044,
  "detectors_run": 3
}
```

---

### GET /api/config

Returns a summary of the loaded configuration.

**Response:** `200 OK`
```json
{
  "loaded": true,
  "layers": ["domain", "application", "infrastructure"],
  "rules": [
    {"id": "domain-purity", "severity": "error", "type": "from-to", "from": "domain", "to": ["infrastructure"]},
    {"id": "infra-deps-limit", "severity": "warning", "type": "expression"}
  ],
  "functions": ["is_clean"]
}
```

---

### POST /api/reload

Force a config reload and full re-check.

**Response:** `200 OK`
```json
{
  "status": "reloaded",
  "message": "Config reloaded and check completed"
}
```

---

### GET /api/performance

Per-detector timing breakdown from the last profiled check.

**Response:** `200 OK`
```json
{
  "phases": [
    {"name": "Go", "duration_ms": 12.3},
    {"name": "TypeScript", "duration_ms": 45.1}
  ],
  "total_ms": 66.1
}
```

---

## SSE (Server-Sent Events)

**Endpoint:** `GET /api/events`

Connects to a real-time event stream. Compatible with `EventSource` in browsers.

### Headers

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

### Event Types

#### `check_complete`

Fired after every architecture check completes.

```
event: check_complete
data: {"violations": 3, "duration_ms": 234}
```

#### `config_reload`

Fired when `arx.yaml` is modified (detected by file watcher).

```
event: config_reload
data: {"path": "/repo/arx.yaml", "timestamp": "2026-05-21T12:00:00Z"}
```

#### `heartbeat`

Sent every 30 seconds to keep the connection alive.

```
event: heartbeat
data: {"timestamp": "2026-05-21T12:00:30Z"}
```

### Client Behavior

- Non-blocking broadcast: slow clients miss events (buffer of 8 events)
- Auto-reconnect: use standard `EventSource` reconnection
- Fallback: browsers without `EventSource` support fall back to polling

---

## GitHub Webhook

**Endpoint:** `POST /api/github-webhook`

Receives GitHub pull request events for automated architecture checks.

### Security

- HMAC-SHA256 signature verification using the configured webhook secret
- Header: `X-Hub-Signature-256`
- Only processes `pull_request` events
- Responds with `200 OK` for verified events, `401 Unauthorized` for invalid signatures

### Request

```json
{
  "action": "opened",
  "pull_request": {
    "number": 42,
    "head": {"ref": "feature/branch", "sha": "abc123"},
    "base": {"ref": "main", "sha": "def456"}
  },
  "repository": {
    "full_name": "owner/repo",
    "clone_url": "https://github.com/owner/repo.git"
  }
}
```

### Response

```json
{
  "status": "success",
  "check_run_id": 12345678,
  "conclusion": "success",
  "summary": "0 new violations, 0 resolved"
}
```

### Configured via `arx server`

The webhook endpoint is only active when `WithPRCheckService` is configured on the server. This is done through the `--github-secret` flag (when available) or programmatically.

---

## LSP Protocol

The LSP server communicates over **stdin/stdout** using **JSON-RPC 2.0** with **Content-Length** headers.

### Transport

```
Content-Length: <number>\r\n
\r\n
<json-rpc-message>
```

### Supported Methods

#### `initialize`

Client → Server handshake.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "processId": 12345,
    "capabilities": {},
    "rootUri": "file:///path/to/project",
    "workspaceFolders": [{"uri": "file:///path/to/project", "name": "my-project"}]
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "capabilities": {
      "textDocumentSync": 1,
      "codeActionProvider": true,
      "hoverProvider": true,
      "diagnosticProvider": {
        "interFileDependencies": true,
        "workspaceDiagnostics": true
      }
    },
    "serverInfo": {
      "name": "arx-lsp",
      "version": "0.57.0"
    }
  }
}
```

#### `textDocument/didOpen`

Notifies the server that a document was opened. Triggers diagnostics.

```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/didOpen",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/project/internal/domain/order.go",
      "languageId": "go",
      "version": 1,
      "text": "package domain\n\nimport \"fmt\"\n"
    }
  }
}
```

#### `textDocument/didChange`

Notifies the server that a document was modified. Triggers re-diagnostics.

```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/didChange",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/project/internal/domain/order.go",
      "version": 2
    },
    "contentChanges": [
      {"text": "package domain\n\nimport \"fmt\"\n\nfunc New() {}"}
    ]
  }
}
```

#### `textDocument/didClose`

Notifies the server that a document was closed.

```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/didClose",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/project/internal/domain/order.go"
    }
  }
}
```

#### `textDocument/codeAction`

Requests quick-fix code actions for a diagnostic.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "textDocument/codeAction",
  "params": {
    "textDocument": {"uri": "file:///path/to/project/internal/domain/order.go"},
    "range": {"start": {"line": 41, "character": 0}, "end": {"line": 41, "character": 80}},
    "context": {
      "diagnostics": [
        {
          "range": {},
          "severity": 1,
          "source": "arx",
          "message": "Domain layer must not depend on infrastructure",
          "code": "domain-no-infra"
        }
      ]
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": [
    {
      "title": "Fix: Move import to infrastructure layer",
      "kind": "quickfix",
      "diagnostics": [],
      "command": {
        "title": "arx.fix",
        "command": "arx.fix",
        "arguments": [
          {
            "uri": "file:///path/to/project/internal/domain/order.go",
            "rule_id": "domain-no-infra",
            "violation_id": "D-01"
          }
        ]
      }
    }
  ]
}
```

#### `textDocument/hover`

Provides architectural context when hovering over an import.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "textDocument/hover",
  "params": {
    "textDocument": {"uri": "file:///path/to/project/internal/domain/order.go"},
    "position": {"line": 42, "character": 5}
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "contents": {
      "kind": "markdown",
      "value": "**Layer: domain**\n\nRules applied:\n- `domain-no-infra`: Cannot [domain → infrastructure] (error)\n- `domain-purity`: Cannot [domain → application] (error)"
    }
  }
}
```

#### `textDocument/publishDiagnostics`

Server → Client push notification for diagnostics.

```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/publishDiagnostics",
  "params": {
    "uri": "file:///path/to/project/internal/domain/order.go",
    "diagnostics": [
      {
        "range": {
          "start": {"line": 42, "character": 0},
          "end": {"line": 42, "character": 80}
        },
        "severity": 1,
        "source": "arx",
        "message": "Domain layer must not depend on infrastructure layer. Import: github.com/example/internal/infrastructure/postgres",
        "code": "domain-no-infra"
      }
    ]
  }
}
```

**Severity mapping:**
| LSP Severity | arx Severity |
|-------------|--------------|
| 1 (Error) | `error` |
| 2 (Warning) | `warning` |
| 3 (Info) | `info` |
| 4 (Hint) | — |

#### `shutdown`

Graceful shutdown.

```json
{"jsonrpc": "2.0", "id": 4, "method": "shutdown"}
```

#### `exit`

Exit notification (must follow `shutdown`).

```json
{"jsonrpc": "2.0", "method": "exit"}
```

### Error Codes

| Code | Meaning |
|------|---------|
| -32002 | Request sent before `initialize` |
| -32600 | Request sent after `shutdown` |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |
