# Neovim Setup

## Install arx

```bash
go install github.com/pauvalls/arx/cmd/arx@latest
```

## Configure with lspconfig

Add to your Neovim config (e.g., `~/.config/nvim/init.lua`):

```lua
-- arx LSP configuration
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

-- Custom LSP config for arx (not in lspconfig by default)
if not configs.arx then
  configs.arx = {
    default_config = {
      cmd = { 'arx', 'lsp' },
      filetypes = { 'go', 'typescript', 'typescriptreact', 'python', 'java',
                    'kotlin', 'rust', 'cs', 'ruby', 'php', 'swift', 'yaml' },
      root_dir = lspconfig.util.root_pattern('arx.yaml', '.git'),
      settings = {},
    },
  }
end

-- Start arx LSP
lspconfig.arx.setup{
  on_attach = function(client, bufnr)
    -- Keybindings
    local opts = { buffer = bufnr, remap = false }

    -- Show architecture info on hover
    vim.keymap.set('n', 'K', vim.lsp.buf.hover, opts)

    -- Code actions (fix suggestions)
    vim.keymap.set({ 'n', 'x' }, '<leader>ca', vim.lsp.buf.code_action, opts)

    -- Go to definition (import resolution)
    vim.keymap.set('n', 'gd', vim.lsp.buf.definition, opts)
  end,
  capabilities = {
    textDocument = {
      hover = {
        contentFormat = { 'markdown', 'plaintext' },
      },
    },
  },
}
```

## Using with mason-lspconfig (advanced)

```lua
-- If you manage LSP servers with mason:
require('mason-lspconfig').setup({
  ensure_installed = { 'arx' },
})

-- For arx (since it's not in mason registry):
-- Just use the lspconfig config above — mason is optional
```

## Diagnostics

Arx diagnostics appear automatically in the Neovim diagnostics list:

```vim
:lua vim.diagnostic.open_float()
```

## Optional: Telescope integration

```lua
-- Show all architecture violations
vim.keymap.set('n', '<leader>av', function()
  local diagnostics = vim.diagnostic.get(0)
  if #diagnostics > 0 then
    require('telescope.builtin').diagnostics()
  end
end)
```
