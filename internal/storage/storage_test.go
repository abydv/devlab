package storage

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesDatabaseAndParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "devlab.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestOpenIsUsable(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "devlab.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE probe (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO probe (id) VALUES (?)`, "a"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var id string
	if err := db.QueryRow(`SELECT id FROM probe WHERE id = ?`, "a").Scan(&id); err != nil {
		t.Fatalf("select: %v", err)
	}
	if id != "a" {
		t.Errorf("id = %q, want %q", id, "a")
	}
}
