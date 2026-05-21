# Zed Setup

## Install arx

```bash
go install github.com/pauvalls/arx/cmd/arx@latest
```

## Configure settings.json

Add to your `~/.config/zed/settings.json`:

```json
{
  "lsp": {
    "arx": {
      "binary": {
        "path": "arx",
        "arguments": ["lsp"]
      },
      "settings": {}
    }
  },
  "languages": {
    "Go": {
      "language_servers": ["gopls", "arx"]
    },
    "TypeScript": {
      "language_servers": ["typescript-language-server", "arx"]
    },
    "Python": {
      "language_servers": ["pyright", "arx"]
    },
    "YAML": {
      "language_servers": ["arx"]
    }
  }
}
```

## Verifying

1. Open a project with `arx.yaml`
2. Open a Go file
3. Check the LSP status: `Ctrl+Shift+P` → "LSP Logs" → should show `arx` connected

## Diagnostics

- Inline diagnostics appear in the editor
- Hover over an import to see architectural context
- Diagnostics panel: `Ctrl+Shift+M`

## Troubleshooting

If arx doesn't start, check:

```bash
which arx            # Must be on PATH
arx lsp --help       # Must work
ls arx.yaml          # Must exist in project root
```
