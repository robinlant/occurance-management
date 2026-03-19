package main

import (
	"database/sql"
	"os"
)

func runMigrations(db *sql.DB) error {
	files := []string{
		"migrations/001_init.sql",
		"migrations/002_email_notifications.sql",
	}
	for _, f := range files {
		migration, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(migration)); err != nil {
			return err
		}
	}
	return nil
}
