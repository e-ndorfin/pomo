package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/Bahaaio/pomo/db"
)

var (
	testModeSetupDone   bool
	testModeTempDir     string
	testModeDBPath      string
	testModeCleanupMu   sync.Mutex
	testModeCleanup     = func() {}
	testModeCleanupOnce sync.Once
)

func setupTestMode() error {
	if !testMode || testModeSetupDone {
		return nil
	}

	defaultDBPath, err := db.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("resolve database path: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "pomo-test-db-*")
	if err != nil {
		return fmt.Errorf("create temp db directory: %w", err)
	}

	tempDBPath := filepath.Join(tempDir, db.DBFile)
	if err := seedTempDB(defaultDBPath, tempDBPath); err != nil {
		_ = os.RemoveAll(tempDir)
		return err
	}

	db.SetDBPathOverride(tempDBPath)
	log.Printf("using temporary test database: %s", tempDBPath)

	testModeTempDir = tempDir
	testModeDBPath = tempDBPath
	testModeSetupDone = true
	testModeCleanupOnce = sync.Once{}

	testModeCleanupMu.Lock()
	testModeCleanup = func() {
		db.ClearDBPathOverride()
		testModeSetupDone = false
		testModeDBPath = ""

		if testModeTempDir != "" {
			if err := os.RemoveAll(testModeTempDir); err != nil {
				log.Printf("failed to remove temporary test database: %v", err)
			}
		}

		testModeTempDir = ""
	}
	testModeCleanupMu.Unlock()

	return nil
}

func cleanupTestMode() {
	testModeCleanupMu.Lock()
	cleanup := testModeCleanup
	testModeCleanupMu.Unlock()

	testModeCleanupOnce.Do(cleanup)
}

func seedTempDB(sourcePath, destPath string) error {
	if err := os.WriteFile(destPath, nil, 0o644); err != nil {
		return fmt.Errorf("create temporary database file: %w", err)
	}

	if err := copyIfExists(sourcePath, destPath); err != nil {
		return err
	}

	if err := copyIfExists(sourcePath+"-wal", destPath+"-wal"); err != nil {
		return err
	}

	if err := copyIfExists(sourcePath+"-shm", destPath+"-shm"); err != nil {
		return err
	}

	return nil
}

func copyIfExists(sourcePath, destPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat database file %s: %w", sourcePath, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("database file %s is not a regular file", sourcePath)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open database file %s: %w", sourcePath, err)
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create database file %s: %w", destPath, err)
	}

	if _, err := io.Copy(dest, source); err != nil {
		_ = dest.Close()
		return fmt.Errorf("copy database file %s: %w", sourcePath, err)
	}

	if err := dest.Close(); err != nil {
		return fmt.Errorf("close database file %s: %w", destPath, err)
	}

	return nil
}
