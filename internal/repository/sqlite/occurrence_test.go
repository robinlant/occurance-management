package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository/sqlite"
)

// setupTestDB opens an in-memory SQLite database and runs all migrations.
// It changes the working directory to the project root so the migrations/
// directory is accessible, then restores the original directory on cleanup.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Save the original working directory and switch to the project root
	// so that os.ReadDir("migrations") works.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// The test binary runs from the package directory; walk up to the project root.
	if err := os.Chdir("../../.."); err != nil {
		t.Fatalf("chdir to project root: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Run migrations (same logic as cmd/server/migrations.go).
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, f := range files {
		migration, err := os.ReadFile("migrations/" + f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(migration)); err != nil {
			t.Fatalf("migration %s: %v", f, err)
		}
	}

	return db
}

// helper: insert a user and return it with an assigned ID.
func insertUser(t *testing.T, repo *sqlite.UserRepository, name, email string) domain.User {
	t.Helper()
	u, err := repo.Save(context.Background(), domain.User{
		Name:         name,
		Email:        email,
		Role:         domain.RoleParticipant,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// helper: insert an occurrence and return it with an assigned ID.
func insertOccurrence(t *testing.T, repo *sqlite.OccurrenceRepository, title string, date time.Time) domain.Occurrence {
	t.Helper()
	o, err := repo.Save(context.Background(), domain.Occurrence{
		Title:           title,
		Description:     "desc",
		Date:            date,
		MinParticipants: 1,
		MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

// helper: insert a participation.
func insertParticipation(t *testing.T, repo *sqlite.ParticipationRepository, userID, occurrenceID int64) {
	t.Helper()
	_, err := repo.Save(context.Background(), domain.Participation{
		UserID:       userID,
		OccurrenceID: occurrenceID,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ---------- Tests ----------

func TestFindAllByUser_EmptyForNoParticipations(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	userRepo := sqlite.NewUserRepository(db)

	user := insertUser(t, userRepo, "alice", "alice@example.com")

	// User exists but has no participations.
	results, err := occRepo.FindAllByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 occurrences, got %d", len(results))
	}
}

func TestFindAllByUser_ReturnsOnlyUserOccurrences(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := insertUser(t, userRepo, "bob", "bob@example.com")

	occ1 := insertOccurrence(t, occRepo, "Shift A", time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC))
	occ2 := insertOccurrence(t, occRepo, "Shift B", time.Date(2026, 4, 2, 8, 0, 0, 0, time.UTC))
	_ = insertOccurrence(t, occRepo, "Shift C", time.Date(2026, 4, 3, 8, 0, 0, 0, time.UTC)) // not signed up

	insertParticipation(t, partRepo, user.ID, occ1.ID)
	insertParticipation(t, partRepo, user.ID, occ2.ID)

	results, err := occRepo.FindAllByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 occurrences, got %d", len(results))
	}

	ids := map[int64]bool{results[0].ID: true, results[1].ID: true}
	if !ids[occ1.ID] || !ids[occ2.ID] {
		t.Fatalf("unexpected occurrence IDs: got %v, want %d and %d", ids, occ1.ID, occ2.ID)
	}
}

func TestFindAllByUser_OrderedByDate(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := insertUser(t, userRepo, "carol", "carol@example.com")

	// Insert out of chronological order.
	occLate := insertOccurrence(t, occRepo, "Late", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))
	occEarly := insertOccurrence(t, occRepo, "Early", time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	occMid := insertOccurrence(t, occRepo, "Mid", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))

	insertParticipation(t, partRepo, user.ID, occLate.ID)
	insertParticipation(t, partRepo, user.ID, occEarly.ID)
	insertParticipation(t, partRepo, user.ID, occMid.ID)

	results, err := occRepo.FindAllByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 occurrences, got %d", len(results))
	}
	if results[0].Title != "Early" || results[1].Title != "Mid" || results[2].Title != "Late" {
		t.Fatalf("unexpected order: %s, %s, %s", results[0].Title, results[1].Title, results[2].Title)
	}
}

func TestFindAllByUser_MultipleUsersIsolation(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	alice := insertUser(t, userRepo, "alice", "alice@iso.com")
	bob := insertUser(t, userRepo, "bob", "bob@iso.com")

	occ1 := insertOccurrence(t, occRepo, "Occ 1", time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	occ2 := insertOccurrence(t, occRepo, "Occ 2", time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC))
	occ3 := insertOccurrence(t, occRepo, "Occ 3", time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC))

	// Alice participates in occ1 and occ3.
	insertParticipation(t, partRepo, alice.ID, occ1.ID)
	insertParticipation(t, partRepo, alice.ID, occ3.ID)
	// Bob participates in occ2.
	insertParticipation(t, partRepo, bob.ID, occ2.ID)

	aliceResults, err := occRepo.FindAllByUser(context.Background(), alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceResults) != 2 {
		t.Fatalf("alice: expected 2, got %d", len(aliceResults))
	}
	for _, r := range aliceResults {
		if r.ID == occ2.ID {
			t.Fatalf("alice should not see occ2 (bob's occurrence)")
		}
	}

	bobResults, err := occRepo.FindAllByUser(context.Background(), bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bobResults) != 1 {
		t.Fatalf("bob: expected 1, got %d", len(bobResults))
	}
	if bobResults[0].ID != occ2.ID {
		t.Fatalf("bob: expected occ2 (ID %d), got ID %d", occ2.ID, bobResults[0].ID)
	}
}

// ---------- RecurrenceID round-trip ----------

func TestSaveAndFindByID_RecurrenceIDPreserved(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "abc123def456"
	saved, err := occRepo.Save(context.Background(), domain.Occurrence{
		Title:           "Recurring Shift",
		Description:     "desc",
		Date:            time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		MinParticipants: 1,
		MaxParticipants: 5,
		RecurrenceID:    recID,
	})
	if err != nil {
		t.Fatal(err)
	}

	found, err := occRepo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.RecurrenceID != recID {
		t.Errorf("RecurrenceID: want %q, got %q", recID, found.RecurrenceID)
	}
}

func TestSaveAndFindByID_EmptyRecurrenceID(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	saved, err := occRepo.Save(context.Background(), domain.Occurrence{
		Title:           "Single Shift",
		Description:     "desc",
		Date:            time.Date(2026, 4, 2, 8, 0, 0, 0, time.UTC),
		MinParticipants: 1,
		MaxParticipants: 5,
		RecurrenceID:    "",
	})
	if err != nil {
		t.Fatal(err)
	}

	found, err := occRepo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.RecurrenceID != "" {
		t.Errorf("RecurrenceID: want empty, got %q", found.RecurrenceID)
	}
}

func TestFindAll_RecurrenceIDPreservedInList(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "shared-recurrence-id"
	for i := 0; i < 3; i++ {
		_, err := occRepo.Save(context.Background(), domain.Occurrence{
			Title:           "Recurring",
			Description:     "desc",
			Date:            time.Date(2026, 4, 1+i, 8, 0, 0, 0, time.UTC),
			MinParticipants: 1,
			MaxParticipants: 5,
			RecurrenceID:    recID,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	all, err := occRepo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 occurrences, got %d", len(all))
	}
	for i, o := range all {
		if o.RecurrenceID != recID {
			t.Errorf("occurrence[%d] RecurrenceID: want %q, got %q", i, recID, o.RecurrenceID)
		}
	}
}

func TestFindAllByUser_IncludesPastAndFuture(t *testing.T) {
	db := setupTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := insertUser(t, userRepo, "dan", "dan@example.com")

	past := insertOccurrence(t, occRepo, "Past", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	future := insertOccurrence(t, occRepo, "Future", time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC))

	insertParticipation(t, partRepo, user.ID, past.ID)
	insertParticipation(t, partRepo, user.ID, future.ID)

	results, err := occRepo.FindAllByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 occurrences (past + future), got %d", len(results))
	}
	// Ordered by date: past first.
	if results[0].Title != "Past" {
		t.Fatalf("expected first result to be 'Past', got %q", results[0].Title)
	}
	if results[1].Title != "Future" {
		t.Fatalf("expected second result to be 'Future', got %q", results[1].Title)
	}
}
