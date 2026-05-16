package db

import (
	"path/filepath"
	"testing"
)

func TestDefaultDBPathUsesHomeStateDirectory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	ClearDBPathOverride()

	dbPath, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("expected default db path, got error: %v", err)
	}

	expectedPath := filepath.Join(homeDir, ".local", "state", "pomo", DBFile)
	if dbPath != expectedPath {
		t.Fatalf("expected db path %q, got %q", expectedPath, dbPath)
	}
}

func TestResolveDBPathPrefersOverride(t *testing.T) {
	t.Cleanup(ClearDBPathOverride)
	overridePath := filepath.Join(t.TempDir(), "override.db")

	SetDBPathOverride(overridePath)

	dbPath, err := ResolveDBPath()
	if err != nil {
		t.Fatalf("expected override db path, got error: %v", err)
	}

	if dbPath != overridePath {
		t.Fatalf("expected override path %q, got %q", overridePath, dbPath)
	}
}
