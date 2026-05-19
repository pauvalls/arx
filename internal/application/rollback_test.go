package application

import (
	"os"
	"strings"
	"testing"
)

// testBackupStorage implements BackupStorage for testing.
type testBackupStorage struct {
	rootDir string
}

func (s *testBackupStorage) Root() string { return s.rootDir }

// workInTempDir changes to a temp directory and returns a cleanup function.
func workInTempDir(t *testing.T) (string, func()) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return tmpDir, func() { os.Chdir(cwd) }
}

func TestRollbackService_BackupSingleFile(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	testFile := "test.go"
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	backupPath, err := svc.BackupFile(testFile, "V-001")
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file was not created at %s", backupPath)
	}
	if !strings.Contains(backupPath, "V-001") {
		t.Errorf("backup path %q should contain violation ID", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package test\n" {
		t.Errorf("backup content = %q, want %q", string(data), "package test\n")
	}
}

func TestRollbackService_RestoreSingleFile(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	testFile := "test.go"
	if err := os.WriteFile(testFile, []byte("modified content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := svc.BackupFile(testFile, "V-001")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(testFile, []byte("new content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := svc.RollbackFile(testFile); err != nil {
		t.Fatalf("RollbackFile failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "modified content\n" {
		t.Errorf("restored content = %q, want %q", string(data), "modified content\n")
	}
}

func TestRollbackService_RollbackAll(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	files := []string{"a.go", "b.go", "c.go"}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("original "+f+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.BackupFile(f, "V-001"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("modified "+f+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := svc.RollbackAll(); err != nil {
		t.Fatalf("RollbackAll failed: %v", err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "original "+f+"\n" {
			t.Errorf("%s content = %q, want %q", f, string(data), "original "+f+"\n")
		}
	}
}

func TestRollbackService_ListBackups(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	testFile := "test.go"
	if err := os.WriteFile(testFile, []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BackupFile(testFile, "V-001"); err != nil {
		t.Fatal(err)
	}

	entries, err := svc.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 backup entry, got %d", len(entries))
	}
	if entries[0].ViolationID != "V-001" {
		t.Errorf("expected ViolationID V-001, got %q", entries[0].ViolationID)
	}
	if entries[0].Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestRollbackService_NoBackups(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	entries, err := svc.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for no backups, got %d", len(entries))
	}
}

func TestRollbackService_RestoreNonexistentFile(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	err := svc.RollbackFile("nonexistent.go")
	if err == nil {
		t.Fatal("expected error when restoring nonexistent file, got nil")
	}
}

func TestRollbackService_BackupWithSubdirs(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	if err := os.MkdirAll("subdir", 0755); err != nil {
		t.Fatal(err)
	}
	testFile := "subdir/test.go"
	if err := os.WriteFile(testFile, []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	backupPath, err := svc.BackupFile(testFile, "V-001")
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}

	if !strings.Contains(backupPath, "V-001") {
		t.Errorf("backup path %q should contain V-001", backupPath)
	}
	if !strings.HasSuffix(backupPath, "subdir/test.go.bak") {
		t.Errorf("backup path %q should end with 'subdir/test.go.bak'", backupPath)
	}
}

func TestRollbackService_RollbackAllMultiViolation(t *testing.T) {
	_, cleanup := workInTempDir(t)
	defer cleanup()

	backupDir := ".arx-backup"
	storage := &testBackupStorage{rootDir: backupDir}
	svc := NewRollbackService(storage)

	for _, f := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(f, []byte("original\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.BackupFile(f, "V-001"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("modified\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, f := range []string{"c.go", "d.go"} {
		if err := os.WriteFile(f, []byte("original\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.BackupFile(f, "V-002"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("modified\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := svc.RollbackAll(); err != nil {
		t.Fatalf("RollbackAll failed: %v", err)
	}

	for _, f := range []string{"a.go", "b.go", "c.go", "d.go"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "original\n" {
			t.Errorf("%s content = %q, want %q", f, string(data), "original\n")
		}
	}
}

func TestNewRollbackService(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &testBackupStorage{rootDir: tmpDir}
	svc := NewRollbackService(storage)
	if svc == nil {
		t.Fatal("NewRollbackService returned nil")
	}
}
