package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// openTestDB opens an in-memory SQLite database and runs all migrations.
// It finds the project root by walking up to go.mod, so it works regardless
// of the working directory and is safe for t.Parallel().
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Find project root by walking up to go.mod.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}

	migrationsDir := filepath.Join(dir, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			t.Fatalf("migration %s: %v", f, err)
		}
	}
	return db
}

// --- helpers for inserting test data ---

func mustInsertUser(t *testing.T, repo *sqlite.UserRepository, name, email string, role domain.Role) domain.User {
	t.Helper()
	u, err := repo.Save(context.Background(), domain.User{
		Name:         name,
		Email:        email,
		Role:         role,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func mustInsertGroup(t *testing.T, repo *sqlite.GroupRepository, name, color string) domain.Group {
	t.Helper()
	g, err := repo.Save(context.Background(), domain.Group{Name: name, Color: color})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func mustInsertOccurrence(t *testing.T, repo *sqlite.OccurrenceRepository, o domain.Occurrence) domain.Occurrence {
	t.Helper()
	saved, err := repo.Save(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

func mustInsertParticipation(t *testing.T, repo *sqlite.ParticipationRepository, userID, occurrenceID int64) domain.Participation {
	t.Helper()
	p, err := repo.Save(context.Background(), domain.Participation{
		UserID:       userID,
		OccurrenceID: occurrenceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func mustInsertComment(t *testing.T, repo *sqlite.CommentRepository, occurrenceID, userID int64, body string) domain.Comment {
	t.Helper()
	c, err := repo.Save(context.Background(), domain.Comment{
		OccurrenceID: occurrenceID,
		UserID:       userID,
		Body:         body,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mustInsertOOO(t *testing.T, repo *sqlite.OutOfOfficeRepository, userID int64, from, to time.Time) domain.OutOfOffice {
	t.Helper()
	o, err := repo.Save(context.Background(), domain.OutOfOffice{
		UserID: userID,
		From:   from,
		To:     to,
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 8, 0, 0, 0, time.UTC)
}
