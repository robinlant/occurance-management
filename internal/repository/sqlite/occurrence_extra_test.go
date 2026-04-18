package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// ---------- FindByGroup ----------

func TestOccurrenceFindByGroup(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)

	g1 := mustInsertGroup(t, groupRepo, "Eng", "blue")
	g2 := mustInsertGroup(t, groupRepo, "Sales", "red")

	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Eng A", Description: "d", Date: date(2026, 4, 1),
		GroupID: g1.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Eng B", Description: "d", Date: date(2026, 4, 2),
		GroupID: g1.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Sales A", Description: "d", Date: date(2026, 4, 1),
		GroupID: g2.ID, MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindByGroup(context.Background(), g1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 for g1, got %d", len(results))
	}
	for _, o := range results {
		if o.GroupID != g1.ID {
			t.Errorf("expected group_id=%d, got %d", g1.ID, o.GroupID)
		}
	}
}

func TestOccurrenceFindByGroup_Empty(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	results, err := occRepo.FindByGroup(context.Background(), 99999)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0, got %d", len(results))
	}
}

func TestOccurrenceFindByGroup_OrderedByDate(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)

	g := mustInsertGroup(t, groupRepo, "Eng", "blue")

	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Late", Description: "d", Date: date(2026, 6, 1),
		GroupID: g.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Early", Description: "d", Date: date(2026, 1, 1),
		GroupID: g.ID, MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindByGroup(context.Background(), g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Title != "Early" || results[1].Title != "Late" {
		t.Errorf("expected Early then Late, got %q then %q", results[0].Title, results[1].Title)
	}
}

// ---------- FindByDate ----------

func TestOccurrenceFindByDate(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Morning", Description: "d", Date: time.Date(2026, 4, 10, 8, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Evening", Description: "d", Date: time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Other Day", Description: "d", Date: date(2026, 4, 11),
		MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindByDate(context.Background(), date(2026, 4, 10))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 on Apr 10, got %d", len(results))
	}
}

func TestOccurrenceFindByDate_None(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	results, err := occRepo.FindByDate(context.Background(), date(2026, 12, 25))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0, got %d", len(results))
	}
}

// ---------- FindOpenSpots ----------

func TestOccurrenceFindOpenSpots(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	u1 := mustInsertUser(t, userRepo, "u1", "u1@test.com", domain.RoleParticipant)
	u2 := mustInsertUser(t, userRepo, "u2", "u2@test.com", domain.RoleParticipant)

	// open: max=3, 1 participant.
	open := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Open", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 3,
	})
	mustInsertParticipation(t, partRepo, u1.ID, open.ID)

	// full: max=1, 1 participant.
	full := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Full", Description: "d", Date: date(2026, 4, 2),
		MinParticipants: 1, MaxParticipants: 1,
	})
	mustInsertParticipation(t, partRepo, u1.ID, full.ID)

	// empty: max=5, 0 participants.
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Empty", Description: "d", Date: date(2026, 4, 3),
		MinParticipants: 1, MaxParticipants: 5,
	})

	// overfull: max=1, 2 participants.
	over := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Overfull", Description: "d", Date: date(2026, 4, 4),
		MinParticipants: 1, MaxParticipants: 1,
	})
	mustInsertParticipation(t, partRepo, u1.ID, over.ID)
	mustInsertParticipation(t, partRepo, u2.ID, over.ID)

	results, err := occRepo.FindOpenSpots(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	titles := map[string]bool{}
	for _, o := range results {
		titles[o.Title] = true
	}
	if !titles["Open"] {
		t.Error("Open should have open spots")
	}
	if !titles["Empty"] {
		t.Error("Empty should have open spots")
	}
	if titles["Full"] {
		t.Error("Full should NOT have open spots")
	}
	if titles["Overfull"] {
		t.Error("Overfull should NOT have open spots")
	}
}

// ---------- FindUpcomingByUser ----------

func TestOccurrenceFindUpcomingByUser(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)

	past := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Past", Description: "d", Date: date(2020, 1, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	future := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Future", Description: "d", Date: date(2030, 12, 31),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, user.ID, past.ID)
	mustInsertParticipation(t, partRepo, user.ID, future.ID)

	results, err := occRepo.FindUpcomingByUser(context.Background(), user.ID, date(2026, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 upcoming, got %d", len(results))
	}
	if results[0].Title != "Future" {
		t.Errorf("expected Future, got %q", results[0].Title)
	}
}

// ---------- FindByTitleLike ----------

func TestOccurrenceFindByTitleLike(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Morning Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Evening Shift", Description: "d", Date: date(2026, 4, 2),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Meeting", Description: "d", Date: date(2026, 4, 3),
		MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindByTitleLike(context.Background(), "Shift", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 matches for 'Shift', got %d", len(results))
	}
}

func TestOccurrenceFindByTitleLike_RespectsLimit(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	for i := range 5 {
		mustInsertOccurrence(t, occRepo, domain.Occurrence{
			Title: "Shift", Description: "d", Date: date(2026, 4, 1+i),
			MinParticipants: 1, MaxParticipants: 5,
		})
	}

	results, err := occRepo.FindByTitleLike(context.Background(), "Shift", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 (limit), got %d", len(results))
	}
}

func TestOccurrenceFindByTitleLike_NoMatch(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindByTitleLike(context.Background(), "zzz", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0, got %d", len(results))
	}
}

// ---------- FindInRange ----------

func TestOccurrenceFindInRange_AllGroups(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "In Range", Description: "d", Date: date(2026, 4, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Out of Range", Description: "d", Date: date(2026, 6, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindInRange(context.Background(),
		date(2026, 4, 1), date(2026, 4, 30), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].Title != "In Range" {
		t.Errorf("expected In Range, got %q", results[0].Title)
	}
}

func TestOccurrenceFindInRange_FilterByGroup(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)

	g1 := mustInsertGroup(t, groupRepo, "Eng", "blue")
	g2 := mustInsertGroup(t, groupRepo, "Sales", "red")

	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Eng Shift", Description: "d", Date: date(2026, 4, 10),
		GroupID: g1.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Sales Shift", Description: "d", Date: date(2026, 4, 10),
		GroupID: g2.ID, MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindInRange(context.Background(),
		date(2026, 4, 1), date(2026, 4, 30), g1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 (Eng only), got %d", len(results))
	}
	if results[0].Title != "Eng Shift" {
		t.Errorf("expected Eng Shift, got %q", results[0].Title)
	}
}

func TestOccurrenceFindInRange_BothBoundariesIncluded(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	// FindInRange formats from/to as YYYY-MM-DD strings and compares against the
	// stored timestamp. To ensure the "to" boundary is inclusive, extend it by a day.
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Start", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "End", Description: "d", Date: date(2026, 4, 30),
		MinParticipants: 1, MaxParticipants: 5,
	})

	// Use May 1 as "to" to include all Apr 30 occurrences regardless of time component.
	results, err := occRepo.FindInRange(context.Background(),
		date(2026, 4, 1), date(2026, 5, 1), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
}

func TestOccurrenceFindInRange_ToNotInclusiveWithTimeComponent(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	// An occurrence at 08:00 on the "to" date is NOT included because
	// "2026-04-30 08:00:00" > "2026-04-30" lexicographically.
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "With Time", Description: "d", Date: date(2026, 4, 30), // 08:00
		MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindInRange(context.Background(),
		date(2026, 4, 1), date(2026, 4, 30), 0)
	if err != nil {
		t.Fatal(err)
	}
	// Documents current behavior: occurrence at 08:00 on "to" date is excluded.
	if len(results) != 0 {
		t.Fatalf("expected 0 (time component makes it exceed to-boundary), got %d", len(results))
	}
}

// ---------- FindByRecurrenceID ----------

func TestOccurrenceFindByRecurrenceID_EmptyString(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	results, err := occRepo.FindByRecurrenceID(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 for empty recurrence_id, got %d", len(results))
	}
}

func TestOccurrenceFindByRecurrenceID_Multiple(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "test-recurrence"
	for i := range 3 {
		mustInsertOccurrence(t, occRepo, domain.Occurrence{
			Title: "Recurring", Description: "d", Date: date(2026, 4, 1+i*7),
			RecurrenceID: recID, MinParticipants: 1, MaxParticipants: 5,
		})
	}
	// Different recurrence.
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Other", Description: "d", Date: date(2026, 5, 1),
		RecurrenceID: "other-rec", MinParticipants: 1, MaxParticipants: 5,
	})

	results, err := occRepo.FindByRecurrenceID(context.Background(), recID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}
}

// ---------- Delete ----------

func TestOccurrenceDelete(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "ToDelete", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	if err := occRepo.Delete(context.Background(), occ.ID); err != nil {
		t.Fatal(err)
	}

	all, err := occRepo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(all))
	}
}

func TestOccurrenceDelete_NonExistent(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	err := occRepo.Delete(context.Background(), 99999)
	if err != nil {
		t.Fatalf("expected no error for nonexistent, got: %v", err)
	}
}

func TestOccurrenceDelete_PreservesOthers(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	occ1 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Keep", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	occ2 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Remove", Description: "d", Date: date(2026, 4, 2),
		MinParticipants: 1, MaxParticipants: 5,
	})

	if err := occRepo.Delete(context.Background(), occ2.ID); err != nil {
		t.Fatal(err)
	}

	all, err := occRepo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != occ1.ID {
		t.Fatalf("expected only occ1 to remain, got %+v", all)
	}
}

// ---------- DeleteByRecurrenceID ----------

func TestOccurrenceDeleteByRecurrenceID(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "to-delete-series"
	for i := range 3 {
		mustInsertOccurrence(t, occRepo, domain.Occurrence{
			Title: "Recurring", Description: "d", Date: date(2026, 4, 1+i*7),
			RecurrenceID: recID, MinParticipants: 1, MaxParticipants: 5,
		})
	}
	// Different recurrence — should survive.
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Other", Description: "d", Date: date(2026, 5, 1),
		RecurrenceID: "keep-this", MinParticipants: 1, MaxParticipants: 5,
	})

	affected, err := occRepo.DeleteByRecurrenceID(context.Background(), recID)
	if err != nil {
		t.Fatal(err)
	}
	if affected != 3 {
		t.Fatalf("expected 3 affected, got %d", affected)
	}

	all, err := occRepo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Title != "Other" {
		t.Fatalf("expected only Other to remain, got %+v", all)
	}
}

func TestOccurrenceDeleteByRecurrenceID_EmptyString(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	affected, err := occRepo.DeleteByRecurrenceID(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if affected != 0 {
		t.Fatalf("expected 0 for empty recurrence_id, got %d", affected)
	}
}

// ---------- DeleteByRecurrenceIDFromDate ----------

func TestOccurrenceDeleteByRecurrenceIDFromDate(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	recID := "partial-delete"
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Week 1", Description: "d", Date: date(2026, 4, 1),
		RecurrenceID: recID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Week 2", Description: "d", Date: date(2026, 4, 8),
		RecurrenceID: recID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Week 3", Description: "d", Date: date(2026, 4, 15),
		RecurrenceID: recID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Week 4", Description: "d", Date: date(2026, 4, 22),
		RecurrenceID: recID, MinParticipants: 1, MaxParticipants: 5,
	})

	// Delete from Week 3 onwards.
	affected, err := occRepo.DeleteByRecurrenceIDFromDate(context.Background(), recID, date(2026, 4, 15))
	if err != nil {
		t.Fatal(err)
	}
	if affected != 2 {
		t.Fatalf("expected 2 affected (week 3 and 4), got %d", affected)
	}

	remaining, err := occRepo.FindByRecurrenceID(context.Background(), recID)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining, got %d", len(remaining))
	}
	if remaining[0].Title != "Week 1" || remaining[1].Title != "Week 2" {
		t.Errorf("unexpected remaining: %q, %q", remaining[0].Title, remaining[1].Title)
	}
}

func TestOccurrenceDeleteByRecurrenceIDFromDate_EmptyString(t *testing.T) {
	db := openTestDB(t)
	occRepo := sqlite.NewOccurrenceRepository(db)

	affected, err := occRepo.DeleteByRecurrenceIDFromDate(context.Background(), "", date(2026, 4, 1))
	if err != nil {
		t.Fatal(err)
	}
	if affected != 0 {
		t.Fatalf("expected 0 for empty recurrence_id, got %d", affected)
	}
}
