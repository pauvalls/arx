package application

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupStorage provides access to the backup directory.
type BackupStorage interface {
	Root() string
}

// BackupEntry represents a single backup record.
type BackupEntry struct {
	Filename    string
	Timestamp   time.Time
	ViolationID string
}

// RollbackService handles file backups and rollbacks.
type RollbackService struct {
	storage BackupStorage
}

// NewRollbackService creates a new RollbackService.
func NewRollbackService(storage BackupStorage) *RollbackService {
	return &RollbackService{storage: storage}
}

// BackupFile creates a backup of the given file with the violation ID as subdirectory.
// Returns the path to the backup file.
// Scheme: .arx-backup/<violation-id>/<file-path>.bak
func (s *RollbackService) BackupFile(filePath, violationID string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot read file %q: %w", filePath, err)
	}

	backupDir := s.storage.Root()
	violationDir := filepath.Join(backupDir, violationID)
	// Use the file path as the base for backup naming (preserves directory structure)
	backupFile := filepath.Join(violationDir, filepath.Clean(filePath)+".bak")
	backupFileDir := filepath.Dir(backupFile)

	if err := os.MkdirAll(backupFileDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create backup directory: %w", err)
	}
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return "", fmt.Errorf("cannot write backup: %w", err)
	}
	return backupFile, nil
}

// RollbackFile restores a single file from its backup.
func (s *RollbackService) RollbackFile(filePath string) error {
	backupDir := s.storage.Root()
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup directory %q does not exist", backupDir)
	}

	cleanPath := filepath.Clean(filePath)
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("cannot read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			// Legacy format: flat .bak files
			legacyFile := filepath.Join(backupDir, filepath.Base(filePath)+".bak")
			if _, err := os.Stat(legacyFile); err == nil {
				return s.restoreFrom(legacyFile, filePath)
			}
			continue
		}
		violationDir := filepath.Join(backupDir, entry.Name())
		backupFile := filepath.Join(violationDir, cleanPath+".bak")
		if _, err := os.Stat(backupFile); err == nil {
			return s.restoreFrom(backupFile, filePath)
		}
	}

	return fmt.Errorf("no backup found for %q in %s", filePath, backupDir)
}

// RollbackAll restores all backed-up files.
func (s *RollbackService) RollbackAll() error {
	backupDir := s.storage.Root()
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("cannot read backup directory: %w", err)
	}

	var lastErr error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		violationDir := filepath.Join(backupDir, entry.Name())
		if err := s.restoreAllFromDir(violationDir); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Clean removes orphaned backup directories (empty or with no .bak files).
func (s *RollbackService) Clean() error {
	backupDir := s.storage.Root()
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("cannot read backup directory: %w", err)
	}

	var lastErr error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if timestampDirRe.MatchString(entry.Name()) {
			// Legacy timestamp directories — remove entirely
			if err := os.RemoveAll(filepath.Join(backupDir, entry.Name())); err != nil {
				lastErr = err
			}
			continue
		}
		// Violation-ID directories — check if empty of .bak files
		violationDir := filepath.Join(backupDir, entry.Name())
		hasBackup := false
		filepath.WalkDir(violationDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".bak") {
				hasBackup = true
				return filepath.SkipAll
			}
			return nil
		})
		if !hasBackup {
			if err := os.RemoveAll(violationDir); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// ListBackups lists all available backups.
func (s *RollbackService) ListBackups() ([]BackupEntry, error) {
	backupDir := s.storage.Root()
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil, nil
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read backup directory: %w", err)
	}

	var backups []BackupEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		violationDir := filepath.Join(backupDir, entry.Name())
		err := filepath.WalkDir(violationDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".bak") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			// Compute the original file path relative to violationDir
			relPath, _ := filepath.Rel(violationDir, path)
			origFile := strings.TrimSuffix(relPath, ".bak")
			backups = append(backups, BackupEntry{
				Filename:    origFile,
				Timestamp:   info.ModTime(),
				ViolationID: entry.Name(),
			})
			return nil
		})
		if err != nil {
			continue
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// restoreAllFromDir walks a directory and restores all .bak files to their original locations.
func (s *RollbackService) restoreAllFromDir(violationDir string) error {
	var lastErr error
	err := filepath.WalkDir(violationDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".bak") {
			return nil
		}
		// Reconstruct original file path
		relPath, _ := filepath.Rel(violationDir, path)
		origFile := strings.TrimSuffix(relPath, ".bak")
		if err := s.restoreFrom(path, origFile); err != nil {
			lastErr = err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return lastErr
}

// restoreFrom copies data from srcFile to dstFile.
func (s *RollbackService) restoreFrom(srcFile, dstFile string) error {
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("cannot read backup %q: %w", srcFile, err)
	}
	if err := os.WriteFile(dstFile, data, 0644); err != nil {
		return fmt.Errorf("cannot restore %q: %w", dstFile, err)
	}
	return nil
}
