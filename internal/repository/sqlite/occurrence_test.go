package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// helper: insert a user and return it with an assigned ID.
func insertUser(t *testing.T, repo *sqlite.UserRepository, name, email string) domain.User {
	t.Helper()
	return mustInsertUser(t, repo, name, email, domain.RoleParticipant)
}

// helper: insert an occurrence and return it with an assigned ID.
func insertOccurrence(t *testing.T, repo *sqlite.OccurrenceRepository, title string, d time.Time) domain.Occurrence {
	t.Helper()
	return mustInsertOccurrence(t, repo, domain.Occurrence{
		Title:           title,
		Description:     "desc",
		Date:            d,
		MinParticipants: 1,
		MaxParticipants: 5,
	})
}

// helper: insert a participation.
func insertParticipation(t *testing.T, repo *sqlite.ParticipationRepository, userID, occurrenceID int64) {
	t.Helper()
	mustInsertParticipation(t, repo, userID, occurrenceID)
}

// ---------- Tests ----------

func TestFindAllByUser_EmptyForNoParticipations(t *testing.T) {
	db := openTestDB(t)
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
	db := openTestDB(t)
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
	db := openTestDB(t)
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
	db := openTestDB(t)
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
	db := openTestDB(t)
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
	db := openTestDB(t)
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
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "shared-recurrence-id"
	for i := range 3 {
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

// ---------- RecurrenceID preservation during updates ----------

func TestSave_UpdatePreservesRecurrenceID(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "recurrence-preserve-test"
	saved, err := occRepo.Save(context.Background(), domain.Occurrence{
		Title:           "Original Title",
		Description:     "original desc",
		Date:            time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		MinParticipants: 1,
		MaxParticipants: 5,
		RecurrenceID:    recID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Update the occurrence: change title and description, but keep RecurrenceID.
	saved.Title = "Updated Title"
	saved.Description = "updated desc"
	// RecurrenceID is still recID on the struct — verify Save preserves it.
	_, err = occRepo.Save(context.Background(), saved)
	if err != nil {
		t.Fatal(err)
	}

	found, err := occRepo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Title != "Updated Title" {
		t.Errorf("Title: want %q, got %q", "Updated Title", found.Title)
	}
	if found.Description != "updated desc" {
		t.Errorf("Description: want %q, got %q", "updated desc", found.Description)
	}
	if found.RecurrenceID != recID {
		t.Errorf("RecurrenceID: want %q, got %q — update must preserve recurrence_id", recID, found.RecurrenceID)
	}
}

func TestSave_UpdateWithEmptyRecurrenceIDOverwrites(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "will-be-wiped"
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

	// Verify the recurrence_id was stored.
	before, err := occRepo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before.RecurrenceID != recID {
		t.Fatalf("setup: RecurrenceID want %q, got %q", recID, before.RecurrenceID)
	}

	// Simulate the bug: update with RecurrenceID="" (what happens when the
	// handler builds a struct from form data without preserving RecurrenceID).
	saved.RecurrenceID = ""
	saved.Title = "Edited Shift"
	_, err = occRepo.Save(context.Background(), saved)
	if err != nil {
		t.Fatal(err)
	}

	after, err := occRepo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	// The raw repo overwrites recurrence_id with whatever the struct contains.
	// This documents WHY the handler must fetch the existing occurrence and
	// copy RecurrenceID before calling Save.
	if after.RecurrenceID != "" {
		t.Errorf("RecurrenceID: want empty (raw repo overwrites), got %q", after.RecurrenceID)
	}
	if after.Title != "Edited Shift" {
		t.Errorf("Title: want %q, got %q", "Edited Shift", after.Title)
	}
}

func TestSave_UpdateNonRecurringStaysNonRecurring(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	saved, err := occRepo.Save(context.Background(), domain.Occurrence{
		Title:           "Single Event",
		Description:     "desc",
		Date:            time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC),
		MinParticipants: 2,
		MaxParticipants: 10,
		RecurrenceID:    "",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Update the occurrence — change title, leave RecurrenceID empty.
	saved.Title = "Single Event (edited)"
	saved.MaxParticipants = 8
	_, err = occRepo.Save(context.Background(), saved)
	if err != nil {
		t.Fatal(err)
	}

	found, err := occRepo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.RecurrenceID != "" {
		t.Errorf("RecurrenceID: want empty (non-recurring), got %q", found.RecurrenceID)
	}
	if found.Title != "Single Event (edited)" {
		t.Errorf("Title: want %q, got %q", "Single Event (edited)", found.Title)
	}
	if found.MaxParticipants != 8 {
		t.Errorf("MaxParticipants: want 8, got %d", found.MaxParticipants)
	}
}

func TestSave_UpdateSiblingDoesNotAffectOthers(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "shared-siblings"
	var siblings [3]domain.Occurrence
	for i := range 3 {
		o, err := occRepo.Save(context.Background(), domain.Occurrence{
			Title:           "Sibling Shift",
			Description:     "desc",
			Date:            time.Date(2026, 4, 1+i*7, 8, 0, 0, 0, time.UTC),
			MinParticipants: 1,
			MaxParticipants: 5,
			RecurrenceID:    recID,
		})
		if err != nil {
			t.Fatal(err)
		}
		siblings[i] = o
	}

	// Update only the second sibling: change title, keep RecurrenceID.
	siblings[1].Title = "Updated Sibling"
	siblings[1].Description = "updated desc"
	_, err := occRepo.Save(context.Background(), siblings[1])
	if err != nil {
		t.Fatal(err)
	}

	// Verify the updated sibling has the new title and still has its recurrence_id.
	updated, err := occRepo.FindByID(context.Background(), siblings[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated Sibling" {
		t.Errorf("updated sibling Title: want %q, got %q", "Updated Sibling", updated.Title)
	}
	if updated.RecurrenceID != recID {
		t.Errorf("updated sibling RecurrenceID: want %q, got %q", recID, updated.RecurrenceID)
	}

	// Verify the other two siblings are completely untouched.
	for _, idx := range []int{0, 2} {
		other, err := occRepo.FindByID(context.Background(), siblings[idx].ID)
		if err != nil {
			t.Fatalf("sibling[%d]: %v", idx, err)
		}
		if other.Title != "Sibling Shift" {
			t.Errorf("sibling[%d] Title: want %q, got %q", idx, "Sibling Shift", other.Title)
		}
		if other.Description != "desc" {
			t.Errorf("sibling[%d] Description: want %q, got %q", idx, "desc", other.Description)
		}
		if other.RecurrenceID != recID {
			t.Errorf("sibling[%d] RecurrenceID: want %q, got %q", idx, recID, other.RecurrenceID)
		}
	}
}

func TestFindAllByUser_IncludesPastAndFuture(t *testing.T) {
	db := openTestDB(t)
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
