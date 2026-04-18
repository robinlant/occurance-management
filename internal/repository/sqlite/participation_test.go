package sqlite_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// ---------- FindByID ----------

func TestParticipationFindByID_Existing(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	p := mustInsertParticipation(t, partRepo, user.ID, occ.ID)

	found, err := partRepo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.UserID != user.ID || found.OccurrenceID != occ.ID {
		t.Errorf("unexpected participation: %+v", found)
	}
}

func TestParticipationFindByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	partRepo := sqlite.NewParticipationRepository(db)

	_, err := partRepo.FindByID(context.Background(), 99999)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

// ---------- FindByOccurrence ----------

func TestParticipationFindByOccurrence_Multiple(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, alice.ID, occ.ID)
	mustInsertParticipation(t, partRepo, bob.ID, occ.ID)

	parts, err := partRepo.FindByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 participations, got %d", len(parts))
	}
}

func TestParticipationFindByOccurrence_Empty(t *testing.T) {
	db := openTestDB(t)
	partRepo := sqlite.NewParticipationRepository(db)

	parts, err := partRepo.FindByOccurrence(context.Background(), 99999)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 0 {
		t.Fatalf("expected 0 participations, got %d", len(parts))
	}
}

// ---------- FindByUser ----------

func TestParticipationFindByUser(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ1 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "A", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	occ2 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "B", Description: "d", Date: date(2026, 4, 2),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, user.ID, occ1.ID)
	mustInsertParticipation(t, partRepo, user.ID, occ2.ID)

	parts, err := partRepo.FindByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 participations, got %d", len(parts))
	}
}

// ---------- FindUsersByOccurrence ----------

func TestParticipationFindUsersByOccurrence(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleOrganizer)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, alice.ID, occ.ID)
	mustInsertParticipation(t, partRepo, bob.ID, occ.ID)

	users, err := partRepo.FindUsersByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	names := map[string]bool{}
	for _, u := range users {
		names[u.Name] = true
	}
	if !names["alice"] || !names["bob"] {
		t.Errorf("expected alice and bob, got %v", names)
	}
}

func TestParticipationFindUsersByOccurrence_Empty(t *testing.T) {
	db := openTestDB(t)
	partRepo := sqlite.NewParticipationRepository(db)

	users, err := partRepo.FindUsersByOccurrence(context.Background(), 99999)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

// ---------- CountByUser ----------

func TestParticipationCountByUser(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	for i := range 3 {
		occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
			Title: "Shift", Description: "d", Date: date(2026, 4, 1+i),
			MinParticipants: 1, MaxParticipants: 5,
		})
		mustInsertParticipation(t, partRepo, user.ID, occ.ID)
	}

	count, err := partRepo.CountByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected 3, got %d", count)
	}
}

func TestParticipationCountByUser_Zero(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)

	count, err := partRepo.CountByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

// ---------- CountByOccurrence ----------

func TestParticipationCountByOccurrence(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	for i := range 4 {
		u := mustInsertUser(t, userRepo, "user"+string(rune('a'+i)), "user"+string(rune('a'+i))+"@test.com", domain.RoleParticipant)
		mustInsertParticipation(t, partRepo, u.ID, occ.ID)
	}

	count, err := partRepo.CountByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("expected 4, got %d", count)
	}
}

// ---------- CountByUserInRange ----------

func TestParticipationCountByUserInRange(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)

	// Create occurrences across different months.
	jan := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Jan", Description: "d", Date: date(2026, 1, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})
	feb := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Feb", Description: "d", Date: date(2026, 2, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mar := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Mar", Description: "d", Date: date(2026, 3, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, user.ID, jan.ID)
	mustInsertParticipation(t, partRepo, user.ID, feb.ID)
	mustInsertParticipation(t, partRepo, user.ID, mar.ID)

	// Count only Feb.
	count, err := partRepo.CountByUserInRange(context.Background(), user.ID,
		date(2026, 2, 1), date(2026, 2, 28))
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 participation in Feb, got %d", count)
	}

	// Count Jan-Feb.
	count, err = partRepo.CountByUserInRange(context.Background(), user.ID,
		date(2026, 1, 1), date(2026, 2, 28))
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 participations in Jan-Feb, got %d", count)
	}
}

// ---------- CountAllByOccurrence ----------

func TestParticipationCountAllByOccurrence(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)

	occ1 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "A", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	occ2 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "B", Description: "d", Date: date(2026, 4, 2),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, alice.ID, occ1.ID)
	mustInsertParticipation(t, partRepo, bob.ID, occ1.ID)
	mustInsertParticipation(t, partRepo, alice.ID, occ2.ID)

	counts, err := partRepo.CountAllByOccurrence(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if counts[occ1.ID] != 2 {
		t.Errorf("occ1: expected 2, got %d", counts[occ1.ID])
	}
	if counts[occ2.ID] != 1 {
		t.Errorf("occ2: expected 1, got %d", counts[occ2.ID])
	}
}

func TestParticipationCountAllByOccurrence_Empty(t *testing.T) {
	db := openTestDB(t)
	partRepo := sqlite.NewParticipationRepository(db)

	counts, err := partRepo.CountAllByOccurrence(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 0 {
		t.Fatalf("expected empty map, got %v", counts)
	}
}

// ---------- CountByUserGroupedByDate ----------

func TestParticipationCountByUserGroupedByDate(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)

	// Two occurrences on the same day.
	occ1 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Morning", Description: "d", Date: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	occ2 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Evening", Description: "d", Date: time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	// One occurrence on a different day.
	occ3 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Next Day", Description: "d", Date: date(2026, 4, 2),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, user.ID, occ1.ID)
	mustInsertParticipation(t, partRepo, user.ID, occ2.ID)
	mustInsertParticipation(t, partRepo, user.ID, occ3.ID)

	m, err := partRepo.CountByUserGroupedByDate(context.Background(), user.ID,
		date(2026, 4, 1), date(2026, 4, 30))
	if err != nil {
		t.Fatal(err)
	}
	if m["2026-04-01"] != 2 {
		t.Errorf("2026-04-01: expected 2, got %d", m["2026-04-01"])
	}
	if m["2026-04-02"] != 1 {
		t.Errorf("2026-04-02: expected 1, got %d", m["2026-04-02"])
	}
}

// ---------- ExistsForUserInDateRange ----------

func TestParticipationExistsForUserInDateRange_True(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertParticipation(t, partRepo, user.ID, occ.ID)

	exists, err := partRepo.ExistsForUserInDateRange(context.Background(), user.ID,
		date(2026, 4, 1), date(2026, 4, 30))
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected true")
	}
}

func TestParticipationExistsForUserInDateRange_False(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 6, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertParticipation(t, partRepo, user.ID, occ.ID)

	// Query April range — occurrence is in June.
	exists, err := partRepo.ExistsForUserInDateRange(context.Background(), user.ID,
		date(2026, 4, 1), date(2026, 4, 30))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected false")
	}
}

// ---------- CountAndInsert (atomic signup) ----------

func TestParticipationCountAndInsert_UnderMax(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	isOver, err := partRepo.CountAndInsert(context.Background(), occ.ID, user.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if isOver {
		t.Fatal("expected isOverMax=false when under max")
	}

	// Verify the participation was actually inserted.
	count, err := partRepo.CountByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 participation, got %d", count)
	}
}

func TestParticipationCountAndInsert_AtMax(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	// Fill to max=2.
	u1 := mustInsertUser(t, userRepo, "u1", "u1@test.com", domain.RoleParticipant)
	u2 := mustInsertUser(t, userRepo, "u2", "u2@test.com", domain.RoleParticipant)
	u3 := mustInsertUser(t, userRepo, "u3", "u3@test.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 2,
	})

	mustInsertParticipation(t, partRepo, u1.ID, occ.ID)
	mustInsertParticipation(t, partRepo, u2.ID, occ.ID)

	// Third signup should report isOverMax=true.
	isOver, err := partRepo.CountAndInsert(context.Background(), occ.ID, u3.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !isOver {
		t.Fatal("expected isOverMax=true when at max")
	}

	// But the participation is still inserted (over-limit handling is caller's job).
	count, err := partRepo.CountByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected 3 participations (inserted despite over-max), got %d", count)
	}
}

func TestParticipationCountAndInsert_FirstParticipant(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 1,
	})

	// First participant, count=0 which is < max=1, so not over.
	isOver, err := partRepo.CountAndInsert(context.Background(), occ.ID, user.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if isOver {
		t.Fatal("expected isOverMax=false for first participant with max=1")
	}
}

// ---------- DeleteByOccurrenceAndUser ----------

func TestParticipationDeleteByOccurrenceAndUser(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, alice.ID, occ.ID)
	mustInsertParticipation(t, partRepo, bob.ID, occ.ID)

	// Delete alice's participation.
	err := partRepo.DeleteByOccurrenceAndUser(context.Background(), occ.ID, alice.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Only bob should remain.
	parts, err := partRepo.FindByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 participation, got %d", len(parts))
	}
	if parts[0].UserID != bob.ID {
		t.Errorf("expected bob's participation, got user_id=%d", parts[0].UserID)
	}
}

func TestParticipationDeleteByOccurrenceAndUser_NonExistent(t *testing.T) {
	db := openTestDB(t)
	partRepo := sqlite.NewParticipationRepository(db)

	// Should not error.
	err := partRepo.DeleteByOccurrenceAndUser(context.Background(), 99999, 99999)
	if err != nil {
		t.Fatalf("expected no error for non-existent, got: %v", err)
	}
}

// ---------- Delete ----------

func TestParticipationDelete(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	p := mustInsertParticipation(t, partRepo, user.ID, occ.ID)

	if err := partRepo.Delete(context.Background(), p.ID); err != nil {
		t.Fatal(err)
	}

	_, err := partRepo.FindByID(context.Background(), p.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got: %v", err)
	}
}

// ---------- LeaderboardAll ----------

func TestParticipationLeaderboardAll_SortedByCountDesc(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)
	carol := mustInsertUser(t, userRepo, "carol", "carol@example.com", domain.RoleParticipant)

	// alice: 3 participations, bob: 1, carol: 2.
	for i := range 3 {
		occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
			Title: "Shift", Description: "d", Date: date(2026, 4, 1+i),
			MinParticipants: 1, MaxParticipants: 5,
		})
		mustInsertParticipation(t, partRepo, alice.ID, occ.ID)
		if i == 0 {
			mustInsertParticipation(t, partRepo, bob.ID, occ.ID)
		}
		if i < 2 {
			mustInsertParticipation(t, partRepo, carol.ID, occ.ID)
		}
	}

	rows, err := partRepo.LeaderboardAll(context.Background(), []domain.Role{domain.RoleParticipant}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// Should be sorted: alice(3), carol(2), bob(1).
	if rows[0].Name != "alice" || rows[0].Count != 3 {
		t.Errorf("row[0]: expected alice/3, got %s/%d", rows[0].Name, rows[0].Count)
	}
	if rows[1].Name != "carol" || rows[1].Count != 2 {
		t.Errorf("row[1]: expected carol/2, got %s/%d", rows[1].Name, rows[1].Count)
	}
	if rows[2].Name != "bob" || rows[2].Count != 1 {
		t.Errorf("row[2]: expected bob/1, got %s/%d", rows[2].Name, rows[2].Count)
	}
}

func TestParticipationLeaderboardAll_FiltersByRole(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	participant := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	organizer := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleOrganizer)

	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertParticipation(t, partRepo, participant.ID, occ.ID)
	mustInsertParticipation(t, partRepo, organizer.ID, occ.ID)

	// Only participants.
	rows, err := partRepo.LeaderboardAll(context.Background(), []domain.Role{domain.RoleParticipant}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (participants only), got %d", len(rows))
	}
	if rows[0].Name != "alice" {
		t.Errorf("expected alice, got %q", rows[0].Name)
	}
}

func TestParticipationLeaderboardAll_FiltersByGroup(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	g1 := mustInsertGroup(t, groupRepo, "Eng", "blue")
	g2 := mustInsertGroup(t, groupRepo, "Sales", "red")

	occ1 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Eng Shift", Description: "d", Date: date(2026, 4, 1),
		GroupID: g1.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	occ2 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Sales Shift", Description: "d", Date: date(2026, 4, 2),
		GroupID: g2.ID, MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, user.ID, occ1.ID)
	mustInsertParticipation(t, partRepo, user.ID, occ2.ID)

	// Leaderboard filtered to g1 only.
	rows, err := partRepo.LeaderboardAll(context.Background(), []domain.Role{domain.RoleParticipant}, g1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Count != 1 {
		t.Errorf("expected count=1 for g1, got %d", rows[0].Count)
	}
}

func TestParticipationLeaderboardAll_ZeroParticipationsIncluded(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	// User exists but has no participations — should still appear with count=0.
	mustInsertUser(t, userRepo, "ghost", "ghost@example.com", domain.RoleParticipant)

	rows, err := partRepo.LeaderboardAll(context.Background(), []domain.Role{domain.RoleParticipant}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (user with 0 participations), got %d", len(rows))
	}
	if rows[0].Count != 0 {
		t.Errorf("expected count=0, got %d", rows[0].Count)
	}
}

// ---------- LeaderboardInRange ----------

func TestParticipationLeaderboardInRange(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)

	jan := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Jan", Description: "d", Date: date(2026, 1, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mar := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Mar", Description: "d", Date: date(2026, 3, 15),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertParticipation(t, partRepo, user.ID, jan.ID)
	mustInsertParticipation(t, partRepo, user.ID, mar.ID)

	// Query Feb-Apr range — only Mar should count.
	rows, err := partRepo.LeaderboardInRange(context.Background(),
		date(2026, 2, 1), date(2026, 4, 30),
		[]domain.Role{domain.RoleParticipant}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Count != 1 {
		t.Errorf("expected count=1 in range, got %d", rows[0].Count)
	}
}

// ---------- ExportInRange ----------

func TestParticipationExportInRange(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)
	g := mustInsertGroup(t, groupRepo, "Eng", "blue")

	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift A", Description: "d", Date: date(2026, 4, 10),
		GroupID: g.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertParticipation(t, partRepo, alice.ID, occ.ID)
	mustInsertParticipation(t, partRepo, bob.ID, occ.ID)

	rows, err := partRepo.ExportInRange(context.Background(),
		date(2026, 4, 1), date(2026, 4, 30),
		[]domain.Role{domain.RoleParticipant}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 export rows, got %d", len(rows))
	}
	// Ordered by date, title, user name.
	if rows[0].UserName != "alice" || rows[1].UserName != "bob" {
		t.Errorf("unexpected order: %q, %q", rows[0].UserName, rows[1].UserName)
	}
	if rows[0].GroupName != "Eng" {
		t.Errorf("expected group name Eng, got %q", rows[0].GroupName)
	}
	if rows[0].OccurrenceTitle != "Shift A" {
		t.Errorf("expected title Shift A, got %q", rows[0].OccurrenceTitle)
	}
}

func TestParticipationExportInRange_NoGroup(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Ungrouped", Description: "d", Date: date(2026, 4, 10),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertParticipation(t, partRepo, user.ID, occ.ID)

	rows, err := partRepo.ExportInRange(context.Background(),
		date(2026, 4, 1), date(2026, 4, 30),
		[]domain.Role{domain.RoleParticipant}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 export row, got %d", len(rows))
	}
	// GroupName should be empty (COALESCE to '').
	if rows[0].GroupName != "" {
		t.Errorf("expected empty group name, got %q", rows[0].GroupName)
	}
}

func TestParticipationExportInRange_FilterByGroup(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	g1 := mustInsertGroup(t, groupRepo, "Eng", "blue")
	g2 := mustInsertGroup(t, groupRepo, "Sales", "red")

	occ1 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Eng Shift", Description: "d", Date: date(2026, 4, 10),
		GroupID: g1.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	occ2 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Sales Shift", Description: "d", Date: date(2026, 4, 10),
		GroupID: g2.ID, MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertParticipation(t, partRepo, user.ID, occ1.ID)
	mustInsertParticipation(t, partRepo, user.ID, occ2.ID)

	// Filter to g1 only.
	rows, err := partRepo.ExportInRange(context.Background(),
		date(2026, 4, 1), date(2026, 4, 30),
		[]domain.Role{domain.RoleParticipant}, g1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 export row (g1 only), got %d", len(rows))
	}
	if rows[0].OccurrenceTitle != "Eng Shift" {
		t.Errorf("expected Eng Shift, got %q", rows[0].OccurrenceTitle)
	}
}

// ---------- Save (update path) ----------

func TestParticipationSave_Update(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	p := mustInsertParticipation(t, partRepo, alice.ID, occ.ID)

	// Reassign participation to bob.
	p.UserID = bob.ID
	_, err := partRepo.Save(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}

	found, err := partRepo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.UserID != bob.ID {
		t.Errorf("expected user_id=%d (bob), got %d", bob.ID, found.UserID)
	}
}
