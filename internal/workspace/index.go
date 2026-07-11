package workspace

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const schema = `
CREATE TABLE IF NOT EXISTS workspaces (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
	status     TEXT NOT NULL,
	template   TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`

// initSchema creates the workspaces index table if it does not already exist.
func initSchema(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("workspace: init schema: %w", err)
	}
	return nil
}

// indexNameTaken reports whether a Workspace with the given name
// (case-insensitive) is already present in the index.
func (m *Manager) indexNameTaken(name string) (bool, error) {
	var exists int
	err := m.db.QueryRow(`SELECT 1 FROM workspaces WHERE name = ? COLLATE NOCASE`, name).Scan(&exists)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	default:
		return false, fmt.Errorf("workspace: check name %s: %w", name, err)
	}
}

// indexInsert adds ws to the index.
func (m *Manager) indexInsert(ws *Workspace) error {
	_, err := m.db.Exec(
		`INSERT INTO workspaces (id, name, status, template, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		ws.ID, ws.Name, string(ws.Status), ws.Template,
		ws.CreatedAt.Format(time.RFC3339Nano), ws.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("workspace: index insert %s: %w", ws.ID, err)
	}
	return nil
}

// indexUpdateStatus updates ws's status and updated_at in the index.
func (m *Manager) indexUpdateStatus(ws *Workspace) error {
	_, err := m.db.Exec(
		`UPDATE workspaces SET status = ?, updated_at = ? WHERE id = ?`,
		string(ws.Status), ws.UpdatedAt.Format(time.RFC3339Nano), ws.ID,
	)
	if err != nil {
		return fmt.Errorf("workspace: index update status %s: %w", ws.ID, err)
	}
	return nil
}

// indexDelete removes id from the index.
func (m *Manager) indexDelete(id string) error {
	if _, err := m.db.Exec(`DELETE FROM workspaces WHERE id = ?`, id); err != nil {
		return fmt.Errorf("workspace: index delete %s: %w", id, err)
	}
	return nil
}

// indexList returns every indexed Workspace ID, ordered by creation time.
func (m *Manager) indexList() ([]string, error) {
	rows, err := m.db.Query(`SELECT id FROM workspaces ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("workspace: index list: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("workspace: index scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("workspace: index rows: %w", err)
	}
	return ids, nil
}
