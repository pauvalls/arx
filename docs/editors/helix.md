# Helix Setup

## Install arx

```bash
go install github.com/pauvalls/arx/cmd/arx@latest
```

## Configure languages.toml

Add to your `~/.config/helix/languages.toml`:

```toml
[language-server.arx]
command = "arx"
args = ["lsp"]

[[language]]
name = "go"
language-servers = ["arx", "gopls"]

[[language]]
name = "typescript"
language-servers = ["arx", "typescript-language-server"]

[[language]]
name = "python"
language-servers = ["arx", "pyright"]

# Other supported languages
[[language]]
name = "java"
language-servers = ["arx", "jdtls"]

[[language]]
name = "rust"
language-servers = ["arx", "rust-analyzer"]

[[language]]
name = "toml"
language-servers = ["arx"]
```

## Language Configuration

If you need per-language settings for arx, add a `[language-server.arx.config]` section:

```toml
[language-server.arx]
command = "arx"
args = ["lsp"]
```

## Verifying

Open a file in a project with `arx.yaml` and run:

```
:lsp-workspace-command
```

You should see arx diagnostics if there are architecture violations.

## Keybindings

Helix handles LSP integration automatically:
- Hover: `Ctrl+w` (or `Alt+Enter` in some configurations)
- Code actions: `Ctrl+a` in normal mode
- Diagnostics: Use `:vsplit-diagnostics` or `:hsplit-diagnostics`
