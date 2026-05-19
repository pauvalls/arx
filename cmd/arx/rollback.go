package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pauvalls/arx/internal/application"
	"github.com/spf13/cobra"
)

// legacyBackupStore implements application.BackupStorage for .arx-backup.
type legacyBackupStore struct{}

func (legacyBackupStore) Root() string { return ".arx-backup" }

var rollbackCmd = &cobra.Command{
	Use:   "rollback [file]",
	Short: "Restore files from backup",
	Long: `Restore files that were backed up during 'arx suggest --apply'.

You can restore a single file, see available backups, restore everything,
or clean up orphaned backup directories.

Examples:
  arx rollback test.go               # Restore a single file
  arx rollback --list                # Show available backups
  arx rollback --all                 # Restore all backed-up files
  arx rollback --clean               # Remove orphaned backup directories`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRollback,
}

var (
	rollbackList  bool
	rollbackAll   bool
	rollbackClean bool

	// rollbackStdout allows test override for output.
	rollbackStdout io.Writer
)

func rollbackOutputWriter() io.Writer {
	if rollbackStdout != nil {
		return rollbackStdout
	}
	return os.Stdout
}

func init() {
	rollbackCmd.Flags().BoolVar(&rollbackList, "list", false, "Show available backups")
	rollbackCmd.Flags().BoolVar(&rollbackAll, "all", false, "Restore all backed-up files")
	rollbackCmd.Flags().BoolVar(&rollbackClean, "clean", false, "Remove orphaned backup directories")
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	out := rollbackOutputWriter()
	store := legacyBackupStore{}
	svc := application.NewRollbackService(store)

	// --clean flag: remove orphaned backup directories
	if rollbackClean {
		fmt.Fprintln(out, "Cleaning orphaned backups...")
		if err := svc.Clean(); err != nil {
			return fmt.Errorf("clean failed: %w", err)
		}
		fmt.Fprintln(out, "✓ Orphaned backups removed.")
		return nil
	}

	// --list flag: show available backups
	if rollbackList {
		backups, err := svc.ListBackups()
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		// Also detect legacy timestamp-based backups
		legacyBackups := detectLegacyBackups(".arx-backup")

		if len(backups) == 0 && len(legacyBackups) == 0 {
			fmt.Fprintln(out, "No backups found.")
			return nil
		}

		fmt.Fprintln(out, "Available backups:")
		fmt.Fprintln(out)

		if len(backups) > 0 {
			fmt.Fprintln(out, "  Violation-ID backups:")
			for _, b := range backups {
				fmt.Fprintf(out, "    %-20s  %s  (%s)\n", b.Filename, b.Timestamp.Format(time.RFC3339), b.ViolationID)
			}
		}

		if len(legacyBackups) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "  ⚠️  Legacy backups (timestamp format):")
			for _, lb := range legacyBackups {
				fmt.Fprintf(out, "    %-20s  %s\n", lb.Filename, lb.Timestamp)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "  Warning: Legacy backup format — may restore more files than expected.")
		}

		return nil
	}

	// --all flag: restore everything
	if rollbackAll {
		fmt.Fprintln(out, "Restoring all files from backup...")
		if err := svc.RollbackAll(); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
		fmt.Fprintln(out, "✓ All files restored.")
		return nil
	}

	// Single file restore
	if len(args) == 0 {
		return fmt.Errorf("specify a file to restore, or use --list or --all")
	}

	filePath := args[0]
	fmt.Fprintf(out, "Restoring %s...\n", filePath)
	if err := svc.RollbackFile(filePath); err != nil {
		// Try legacy backup as fallback
		if legacyErr := restoreFromLegacy(out, filePath); legacyErr == nil {
			return nil
		}
		return fmt.Errorf("cannot restore %q: %w", filePath, err)
	}
	fmt.Fprintln(out, "✓ File restored.")
	return nil
}

// timestampDirRe matches timestamp-based backup directory names.
var rollbackTimestampRe = regexp.MustCompile(`^\d{8}T\d{6}$`)

// legacyBackupEntry represents a file backed up in the old timestamp format.
type legacyBackupEntry struct {
	Filename  string
	Timestamp string
}

// detectLegacyBackups finds timestamp-based backup directories and lists files.
func detectLegacyBackups(backupDir string) []legacyBackupEntry {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil
	}

	var result []legacyBackupEntry
	for _, entry := range entries {
		if !entry.IsDir() || !rollbackTimestampRe.MatchString(entry.Name()) {
			continue
		}
		tsDir := filepath.Join(backupDir, entry.Name())
		files, err := os.ReadDir(tsDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".bak") {
				result = append(result, legacyBackupEntry{
					Filename:  strings.TrimSuffix(f.Name(), ".bak"),
					Timestamp: entry.Name(),
				})
			}
		}
	}
	return result
}

// restoreFromLegacy attempts to restore a file from legacy timestamp-based backups.
func restoreFromLegacy(out io.Writer, filePath string) error {
	baseName := filepath.Base(filePath)
	entries, err := os.ReadDir(".arx-backup")
	if err != nil {
		return err
	}

	var latestDir string
	for _, entry := range entries {
		if entry.IsDir() && rollbackTimestampRe.MatchString(entry.Name()) {
			if entry.Name() > latestDir {
				latestDir = entry.Name()
			}
		}
	}

	if latestDir == "" {
		return fmt.Errorf("no legacy backup found")
	}

	fmt.Fprintf(out, "  ⚠️  Legacy backup format — may restore more files than expected.\n")

	backupFile := filepath.Join(".arx-backup", latestDir, baseName+".bak")
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return err
	}

	fmt.Fprintln(out, "✓ File restored from legacy backup.")
	return nil
}


