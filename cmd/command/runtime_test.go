package command

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenLogFileAtPathsFallsBackToNextCandidate(t *testing.T) {
	tempDir := t.TempDir()
	blocker := filepath.Join(tempDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}

	wantPath := filepath.Join(tempDir, "logs", "output.log")
	logFile, gotPath, err := openLogFileAtPaths([]string{
		filepath.Join(blocker, "output.log"),
		wantPath,
	})
	if err != nil {
		t.Fatalf("open log file: %v", err)
	}
	defer logFile.Close()

	if gotPath != wantPath {
		t.Fatalf("got path %q, want %q", gotPath, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("stat fallback log path: %v", err)
	}
}

func TestOpenLogFileAtPathsReturnsErrorWhenCandidatesFail(t *testing.T) {
	tempDir := t.TempDir()
	blocker := filepath.Join(tempDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}

	_, _, err := openLogFileAtPaths([]string{
		filepath.Join(blocker, "one.log"),
		filepath.Join(blocker, "two.log"),
	})
	if err == nil {
		t.Fatal("expected error when all log paths fail")
	}
}
