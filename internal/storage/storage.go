// Package storage opens the SQLite database DevLab's other internal
// packages use to persist queryable indexes. It has no knowledge of
// any domain type (Workspace, Template, ...) — schema ownership stays
// with the package that owns the data.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open opens (creating if necessary) a SQLite database at path,
// creating its parent directory as needed.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("storage: create directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: connect %s: %w", path, err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: enable foreign keys: %w", err)
	}

	return db, nil
}
