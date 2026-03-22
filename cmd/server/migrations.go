package main

import (
	"database/sql"
	"log/slog"
	"os"
	"sort"
	"strings"
)

func runMigrations(db *sql.DB) error {
	// Create migration tracking table
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	// Read migration directory
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		// Check if already applied
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = ?`, f).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		migration, err := os.ReadFile("migrations/" + f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(migration)); err != nil {
			return err
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, f); err != nil {
			return err
		}
		slog.Info("migration applied", "file", f)
	}
	return nil
}
