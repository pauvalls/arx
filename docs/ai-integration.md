# AI Coding Assistant Integration

Arx includes a skill for AI coding assistants (opencode, Claude Code, Cursor) that enables them to automatically analyze and configure arx for any project.

## How It Works

When you ask your AI assistant to set up arx in a project, the arx-setup skill activates and guides it through:

1. **Scanning** the project structure (languages, directory patterns)
2. **Detecting** architectural conventions (Clean, Hexagonal, DDD)
3. **Generating** an `arx.yaml` configuration with appropriate layers and rules
4. **Validating** the config and running an initial architecture check

## Quick Install

```bash
# Interactive: select from detected tools
arx skill install

# Install to all detected tools
arx skill install --all

# Install to specific tools
arx skill install opencode
arx skill install opencode claude
```

The command detects which AI coding assistants you have installed and installs the arx-setup skill to each one.

## What Gets Installed

Each tool receives a `SKILL.md` file in its skills directory:

| Tool | Installation Path |
|------|------------------|
| opencode | `~/.config/opencode/skills/arx-setup/SKILL.md` |
| Claude Code | `~/.claude/skills/arx-setup/SKILL.md` |

The skill file contains instructions that tell the AI assistant how to:
- Analyze codebase structure for architectural layers
- Generate accurate `arx.yaml` configurations
- Detect cross-language dependencies (proto, OpenAPI)
- Use the full arx feature set (suggest, explain, baseline, fmt)

## Manual Installation

If you prefer to install manually, copy the skill from the arx repository:

```bash
# For opencode
cp -r contrib/opencode/arx-setup ~/.config/opencode/skills/arx-setup

# For Claude Code
cp -r contrib/opencode/arx-setup ~/.claude/skills/arx-setup
```

## Usage Examples

After installation, you can ask your AI assistant:

> "Set up arx in this project"
> "Configure architecture audit for this NestJS project"
> "Generate arx.yaml for this Go Clean Architecture project"
> "Add cross-language dependency detection for our proto files"

The AI will scan your project, detect the architecture pattern, and generate a complete `arx.yaml` with appropriate layers, rules, and exclusions.

## Supported AI Assistants

| Assistant | Support | Notes |
|-----------|---------|-------|
| opencode | ✅ Full | Agent Skills system |
| Claude Code | ✅ Full | Skills at `~/.claude/skills/` |
| Cursor | ✅ Basic | Rules at `~/.cursor/rules/` |
| GitHub Copilot | ⏳ Planned | `.github/copilot-instructions.md` |
| Aider | ⏳ Planned | Configuration file support |

## Removing the Skill

```bash
# Remove from opencode
rm -rf ~/.config/opencode/skills/arx-setup

# Remove from Claude Code
rm -rf ~/.claude/skills/arx-setup
```
