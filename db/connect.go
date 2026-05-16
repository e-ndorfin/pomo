// Package db handles the database connection and initialization.
package db

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Bahaaio/pomo/config"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const DBFile = config.AppName + ".db"

var (
	dbPathOverride string
	dbPathMu       sync.RWMutex
)

// Connect connects to the SQLite database,
// creates the necessary directories,
// and performs migrations if needed.
func Connect() (*sqlx.DB, error) {
	dbPath, err := ResolveDBPath()
	if err != nil {
		log.Println("failed to get db path:", err)
		return nil, err
	}

	dbDir := filepath.Dir(dbPath)

	// create the db directory if it doesn't exist
	if err = os.MkdirAll(dbDir, 0o755); err != nil {
		log.Println("failed to create db directory:", err)
		return nil, err
	}

	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		log.Println("failed to connect to the db:", err)
		return nil, err
	}
	log.Println("connected to the db")

	if err = db.Ping(); err != nil {
		log.Println("failed to ping the db:", err)
		return nil, err
	}
	log.Println("pinged the db")

	// limit the number of open connections to 1
	db.SetMaxOpenConns(1)

	// migrate the database
	if err = createSchema(db); err != nil {
		log.Println("failed to migrate the db:", err)
		return nil, err
	}

	return db, nil
}

// ResolveDBPath returns the active database path, respecting any process-level override.
func ResolveDBPath() (string, error) {
	dbPathMu.RLock()
	override := dbPathOverride
	dbPathMu.RUnlock()

	if override != "" {
		return override, nil
	}

	return DefaultDBPath()
}

// DefaultDBPath returns the standard on-disk database path for the current user.
func DefaultDBPath() (string, error) {
	dbDir, err := getDBDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dbDir, DBFile), nil
}

// SetDBPathOverride forces Connect to use a specific database path for the current process.
func SetDBPathOverride(path string) {
	dbPathMu.Lock()
	defer dbPathMu.Unlock()

	dbPathOverride = path
}

// ClearDBPathOverride removes any process-level database path override.
func ClearDBPathOverride() {
	dbPathMu.Lock()
	defer dbPathMu.Unlock()

	dbPathOverride = ""
}

func createSchema(db *sqlx.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	log.Println("created the schema")

	if _, err := db.Exec("ALTER TABLE sessions ADD COLUMN ended_at TEXT;"); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}

	return nil
}

// returns the path to the db directory
func getDBDir() (string, error) {
	var dir string

	// on Linux and macOS, use ~/.local/state
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}

		dir = filepath.Join(dir, ".local", "state")
	} else {
		// on other OSes, use the standard user config directory
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}

	// join the dir with the app name
	return filepath.Join(dir, config.AppName), nil
}
