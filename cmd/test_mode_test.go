package cmd

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Bahaaio/pomo/db"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestRootCommandRegistersTestFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("test")
	if flag == nil {
		t.Fatal("expected --test flag to be registered")
	}
}

func TestSetupTestModeCopiesExistingDatabaseAndCleansUp(t *testing.T) {
	resetTestModeState()
	t.Cleanup(resetTestModeState)

	t.Setenv("HOME", t.TempDir())

	defaultDBPath, err := db.DefaultDBPath()
	if err != nil {
		t.Fatalf("expected default db path, got error: %v", err)
	}

	seedDatabase(t, 1)

	testMode = true
	if err := setupTestMode(); err != nil {
		t.Fatalf("expected test mode setup to succeed, got error: %v", err)
	}

	overridePath, err := db.ResolveDBPath()
	if err != nil {
		t.Fatalf("expected override db path, got error: %v", err)
	}

	if overridePath == defaultDBPath {
		t.Fatalf("expected override path to differ from default path %q", defaultDBPath)
	}

	if _, err := os.Stat(overridePath); err != nil {
		t.Fatalf("expected temp db to exist, got error: %v", err)
	}

	tempDatabase, err := db.Connect()
	if err != nil {
		t.Fatalf("expected temp db connection, got error: %v", err)
	}

	tempRepo := db.NewSessionRepo(tempDatabase)
	assertSessionCount(t, tempRepo, 1)
	createWorkSession(t, tempRepo)

	if err := tempRepo.Close(); err != nil {
		t.Fatalf("expected temp repo to close cleanly, got error: %v", err)
	}

	originalRepo, err := openSQLitePath(defaultDBPath)
	if err != nil {
		t.Fatalf("expected original db connection, got error: %v", err)
	}
	assertSessionCount(t, originalRepo, 1)

	if err := originalRepo.Close(); err != nil {
		t.Fatalf("expected original repo to close cleanly, got error: %v", err)
	}

	tempDir := testModeTempDir
	cleanupTestMode()

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Fatalf("expected temp dir %q to be removed, got err=%v", tempDir, err)
	}

	resolvedPath, err := db.ResolveDBPath()
	if err != nil {
		t.Fatalf("expected default path after cleanup, got error: %v", err)
	}

	if resolvedPath != defaultDBPath {
		t.Fatalf("expected cleanup to restore default path %q, got %q", defaultDBPath, resolvedPath)
	}
}

func TestSetupTestModeCreatesEmptyTempDatabaseWhenNoRealDBExists(t *testing.T) {
	resetTestModeState()
	t.Cleanup(resetTestModeState)

	t.Setenv("HOME", t.TempDir())
	testMode = true

	if err := setupTestMode(); err != nil {
		t.Fatalf("expected test mode setup to succeed without a real db, got error: %v", err)
	}

	overridePath, err := db.ResolveDBPath()
	if err != nil {
		t.Fatalf("expected override db path, got error: %v", err)
	}

	if _, err := os.Stat(overridePath); err != nil {
		t.Fatalf("expected temp db file to exist, got error: %v", err)
	}

	tempDatabase, err := db.Connect()
	if err != nil {
		t.Fatalf("expected temp db connection, got error: %v", err)
	}

	tempRepo := db.NewSessionRepo(tempDatabase)
	assertSessionCount(t, tempRepo, 0)

	if err := tempRepo.Close(); err != nil {
		t.Fatalf("expected temp repo to close cleanly, got error: %v", err)
	}
}

func resetTestModeState() {
	testMode = false
	testModeSetupDone = false
	testModeTempDir = ""
	testModeDBPath = ""
	testModeCleanup = func() {}
	testModeCleanupOnce = sync.Once{}
	db.ClearDBPathOverride()
}

func seedDatabase(t *testing.T, sessions int) {
	t.Helper()

	database, err := db.Connect()
	if err != nil {
		t.Fatalf("expected default db connection, got error: %v", err)
	}

	repo := db.NewSessionRepo(database)
	for range make([]struct{}, sessions) {
		createWorkSession(t, repo)
	}

	if err := repo.Close(); err != nil {
		t.Fatalf("expected seeded repo to close cleanly, got error: %v", err)
	}
}

func createWorkSession(t *testing.T, repo *db.SessionRepo) {
	t.Helper()

	startedAt := time.Now().Add(-25 * time.Minute).Round(time.Second)
	endedAt := startedAt.Add(25 * time.Minute)

	if err := repo.CreateSession(startedAt, endedAt, endedAt.Sub(startedAt), db.WorkSession); err != nil {
		t.Fatalf("expected session insert to succeed, got error: %v", err)
	}
}

func assertSessionCount(t *testing.T, repo *db.SessionRepo, want int) {
	t.Helper()

	stats, err := repo.GetAllTimeStats()
	if err != nil {
		t.Fatalf("expected stats query to succeed, got error: %v", err)
	}

	if stats.TotalSessions != want {
		t.Fatalf("expected %d sessions, got %d", want, stats.TotalSessions)
	}
}

func openSQLitePath(dbPath string) (*db.SessionRepo, error) {
	database, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	return db.NewSessionRepo(database), nil
}
