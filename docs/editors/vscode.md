# VS Code Setup

## Install the arx LSP Server

```bash
go install github.com/pauvalls/arx/cmd/arx@latest
```

## Configure settings.json

Add to your `.vscode/settings.json` (workspace level) or your global `settings.json`:

```json
{
  "arx.enableLSP": true,
  "arx.lsp.path": "arx",
  "arx.lsp.args": ["lsp"],
  "arx.checkOnSave": true,
  "arx.severity": {
    "error": "error",
    "warning": "warning",
    "info": "information"
  }
}
```

> **Note**: VS Code support uses arx's LSP server via stdin/stdout. The built-in LSP client was designed for arx's protocol. If you prefer using the generic LSP client, see the Neovim/Helix configurations below.

## Keybindings (optional)

Add to `keybindings.json`:

```json
[
  {
    "key": "ctrl+shift+a",
    "command": "arx.check"
  },
  {
    "key": "ctrl+shift+e",
    "command": "arx.explain",
    "args": { "violationId": "${selectedId}" }
  }
]
```

## Tasks (optional)

```json
// .vscode/tasks.json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "arx: check workspace",
      "type": "shell",
      "command": "arx workspace",
      "problemMatcher": []
    },
    {
      "label": "arx: audit",
      "type": "shell",
      "command": "arx audit",
      "problemMatcher": []
    }
  ]
}
```
