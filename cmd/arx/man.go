package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// manCmd represents the man command
var manCmd = &cobra.Command{
	Use:   "man",
	Short: "Generate man pages for arx",
	Long: `Generate man pages for all arx commands.

By default, man pages are written to docs/man/ in the current directory.
Use --output to specify a different directory.

Each command gets its own man page file (e.g., arx.1, arx-check.1).

  $ arx man
  $ arx man --output /usr/local/share/man/man1/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")

		// Ensure output directory exists
		if err := os.MkdirAll(output, 0o755); err != nil {
			return err
		}

		header := &doc.GenManHeader{
			Title:   "ARX",
			Section: "1",
		}
		return doc.GenManTree(rootCmd, header, output)
	},
}

func init() {
	manCmd.Flags().StringP("output", "o", "docs/man", "output directory for man pages")
	rootCmd.AddCommand(manCmd)
}
