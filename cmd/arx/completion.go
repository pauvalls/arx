package main

import (
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command group
var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for arx.

Supported shells: bash, zsh, fish, powershell

To install completions:

  Bash:
    $ arx completion bash > /etc/bash_completion.d/arx
    $ source /etc/bash_completion.d/arx

  Zsh:
    $ arx completion zsh > "${fpath[1]}/_arx"
    $ source ~/.zshrc

  Fish:
    $ arx completion fish > ~/.config/fish/completions/arx.fish
    $ source ~/.config/fish/completions/arx.fish

  PowerShell:
    $ arx completion powershell | Out-String | Invoke-Expression

After installation, restart your shell or source the completion file.`,
	Hidden: true, // Hidden from main help, but available via "completion --help"
}

// completionBashCmd generates bash completion script
var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Long: `Generate bash completion script for arx.

To install bash completions:

  # System-wide (requires root):
    $ arx completion bash | sudo tee /etc/bash_completion.d/arx
    $ source /etc/bash_completion.d/arx

  # User-only:
    $ arx completion bash >> ~/.bash_completion
    $ source ~/.bash_completion

  # macOS with Homebrew:
    $ arx completion bash > $(brew --prefix)/etc/bash_completion.d/arx

After installation, restart your shell or source the completion file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletion(cmd.OutOrStdout())
	},
}

// completionZshCmd generates zsh completion script
var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Long: `Generate zsh completion script for arx.

To install zsh completions:

  # If you have a compinit setup:
    $ arx completion zsh > "${fpath[1]}/_arx"

  # User-only (add to ~/.zshrc first):
    $ mkdir -p ~/.zsh/completions
    $ arx completion zsh > ~/.zsh/completions/_arx
    $ echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
    $ source ~/.zshrc

  # macOS with Homebrew:
    $ arx completion zsh > $(brew --prefix)/share/zsh/site-functions/_arx

After installation, restart your shell or run: compinit
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(cmd.OutOrStdout())
	},
}

// completionFishCmd generates fish completion script
var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Long: `Generate fish completion script for arx.

To install fish completions:

  # System-wide or user config:
    $ arx completion fish > ~/.config/fish/completions/arx.fish
    $ source ~/.config/fish/completions/arx.fish

  # Or use fish's built-in completion save:
    $ arx completion fish | fish_update_completions

After installation, restart your shell or source the completion file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
	},
}

// completionPowerShellCmd generates PowerShell completion script
var completionPowerShellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate PowerShell completion script",
	Long: `Generate PowerShell completion script for arx.

To install PowerShell completions:

  # Current session:
    $ arx completion powershell | Out-String | Invoke-Expression

  # Persistent (add to your $PROFILE):
    $ arx completion powershell >> $PROFILE
    $ . $PROFILE

  # Find your $PROFILE path:
    $ echo $PROFILE

After installation, restart your shell or source your profile.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
	},
}

func init() {
	// Register subcommands
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	completionCmd.AddCommand(completionPowerShellCmd)

	// Register completion command group
	rootCmd.AddCommand(completionCmd)
}
