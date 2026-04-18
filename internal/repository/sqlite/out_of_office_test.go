package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// ---------- FindByID ----------

func TestOOOFindByID_Existing(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	ooo := mustInsertOOO(t, oooRepo, user.ID, date(2026, 1, 1), date(2026, 1, 10))

	found, err := oooRepo.FindByID(context.Background(), ooo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.UserID != user.ID {
		t.Errorf("expected user_id=%d, got %d", user.ID, found.UserID)
	}
}

func TestOOOFindByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	_, err := oooRepo.FindByID(context.Background(), 99999)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

// ---------- FindByUser ----------

func TestOOOFindByUser_Multiple(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)

	mustInsertOOO(t, oooRepo, user.ID, date(2026, 1, 1), date(2026, 1, 10))
	mustInsertOOO(t, oooRepo, user.ID, date(2026, 3, 1), date(2026, 3, 10))

	ooos, err := oooRepo.FindByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ooos) != 2 {
		t.Fatalf("expected 2 OOOs, got %d", len(ooos))
	}
}

func TestOOOFindByUser_Empty(t *testing.T) {
	db := openTestDB(t)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	ooos, err := oooRepo.FindByUser(context.Background(), 99999)
	if err != nil {
		t.Fatal(err)
	}
	if len(ooos) != 0 {
		t.Fatalf("expected 0, got %d", len(ooos))
	}
}

func TestOOOFindByUser_Isolation(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)

	mustInsertOOO(t, oooRepo, alice.ID, date(2026, 1, 1), date(2026, 1, 10))
	mustInsertOOO(t, oooRepo, bob.ID, date(2026, 2, 1), date(2026, 2, 10))

	aliceOOOs, err := oooRepo.FindByUser(context.Background(), alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceOOOs) != 1 {
		t.Fatalf("alice: expected 1, got %d", len(aliceOOOs))
	}
	if aliceOOOs[0].UserID != alice.ID {
		t.Errorf("expected alice's user_id, got %d", aliceOOOs[0].UserID)
	}
}

// ---------- Save (update path) ----------

func TestOOOSave_Update(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	ooo := mustInsertOOO(t, oooRepo, user.ID, date(2026, 1, 1), date(2026, 1, 10))

	// Extend the OOO period.
	ooo.To = date(2026, 1, 20)
	_, err := oooRepo.Save(context.Background(), ooo)
	if err != nil {
		t.Fatal(err)
	}

	found, err := oooRepo.FindByID(context.Background(), ooo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.To.Day() != 20 {
		t.Errorf("expected To day=20, got %d", found.To.Day())
	}
}

// ---------- Delete ----------

func TestOOODelete_Existing(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	ooo := mustInsertOOO(t, oooRepo, user.ID, date(2026, 1, 1), date(2026, 1, 10))

	if err := oooRepo.Delete(context.Background(), ooo.ID); err != nil {
		t.Fatal(err)
	}

	_, err := oooRepo.FindByID(context.Background(), ooo.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got: %v", err)
	}
}

func TestOOODelete_NonExistent(t *testing.T) {
	db := openTestDB(t)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	err := oooRepo.Delete(context.Background(), 99999)
	if err != nil {
		t.Fatalf("expected no error for nonexistent, got: %v", err)
	}
}
