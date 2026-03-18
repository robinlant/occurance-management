package main

import (
	"database/sql"
	"os"
)

func runMigrations(db *sql.DB) error {
	migration, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(migration))
	return err
}
