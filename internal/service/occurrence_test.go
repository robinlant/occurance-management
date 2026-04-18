package service_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
	"github.com/robinlant/dutyround/internal/service"
)

// setupTestDB opens an in-memory SQLite database and runs all migrations.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("chdir to project root: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

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

func TestGetAllForUser_ReturnsRepositoryResults(t *testing.T) {
	db := setupTestDB(t)

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	svc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)

	user, err := userRepo.Save(context.Background(), domain.User{
		Name: "eve", Email: "eve@example.com", Role: domain.RoleParticipant, PasswordHash: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}

	occ1, err := occRepo.Save(context.Background(), domain.Occurrence{
		Title: "Morning", Description: "d", Date: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	occ2, err := occRepo.Save(context.Background(), domain.Occurrence{
		Title: "Evening", Description: "d", Date: time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Participate in both.
	if _, err := partRepo.Save(context.Background(), domain.Participation{UserID: user.ID, OccurrenceID: occ1.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := partRepo.Save(context.Background(), domain.Participation{UserID: user.ID, OccurrenceID: occ2.ID}); err != nil {
		t.Fatal(err)
	}

	results, err := svc.GetAllForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
	if results[0].ID != occ1.ID || results[1].ID != occ2.ID {
		t.Fatalf("unexpected IDs: got [%d, %d], want [%d, %d]", results[0].ID, results[1].ID, occ1.ID, occ2.ID)
	}
}

func TestGetAllForUser_Integration(t *testing.T) {
	db := setupTestDB(t)

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	svc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)

	ctx := context.Background()

	// Create two users.
	alice, err := userRepo.Save(ctx, domain.User{
		Name: "alice", Email: "alice@int.com", Role: domain.RoleParticipant, PasswordHash: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := userRepo.Save(ctx, domain.User{
		Name: "bob", Email: "bob@int.com", Role: domain.RoleParticipant, PasswordHash: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create three occurrences.
	occ1, err := occRepo.Save(ctx, domain.Occurrence{
		Title: "A", Description: "d", Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	occ2, err := occRepo.Save(ctx, domain.Occurrence{
		Title: "B", Description: "d", Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	occ3, err := occRepo.Save(ctx, domain.Occurrence{
		Title: "C", Description: "d", Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Alice -> occ1, occ3.  Bob -> occ2, occ3.
	for _, p := range []domain.Participation{
		{UserID: alice.ID, OccurrenceID: occ1.ID},
		{UserID: alice.ID, OccurrenceID: occ3.ID},
		{UserID: bob.ID, OccurrenceID: occ2.ID},
		{UserID: bob.ID, OccurrenceID: occ3.ID},
	} {
		if _, err := partRepo.Save(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	// Verify Alice.
	aliceOccs, err := svc.GetAllForUser(ctx, alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceOccs) != 2 {
		t.Fatalf("alice: expected 2, got %d", len(aliceOccs))
	}
	if aliceOccs[0].Title != "A" || aliceOccs[1].Title != "C" {
		t.Fatalf("alice: unexpected titles: %q, %q", aliceOccs[0].Title, aliceOccs[1].Title)
	}

	// Verify Bob.
	bobOccs, err := svc.GetAllForUser(ctx, bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bobOccs) != 2 {
		t.Fatalf("bob: expected 2, got %d", len(bobOccs))
	}
	if bobOccs[0].Title != "B" || bobOccs[1].Title != "C" {
		t.Fatalf("bob: unexpected titles: %q, %q", bobOccs[0].Title, bobOccs[1].Title)
	}

	// Verify a user with no participations returns empty.
	noUser, err := userRepo.Save(ctx, domain.User{
		Name: "nobody", Email: "nobody@int.com", Role: domain.RoleParticipant, PasswordHash: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	emptyOccs, err := svc.GetAllForUser(ctx, noUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(emptyOccs) != 0 {
		t.Fatalf("nobody: expected 0, got %d", len(emptyOccs))
	}
}

// --- Bug #9: Withdraw for non-participant ---

func TestWithdraw_NotSignedUp(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	occSvc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)
	userSvc := service.NewUserService(userRepo, oooRepo, partRepo)

	user := createUser(t, userSvc, "ghost", "ghost@test.com")
	occ, err := occSvc.CreateOccurrence(ctx, domain.Occurrence{
		Title: "Test", Description: "d", Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = occSvc.Withdraw(ctx, occ.ID, user.ID)
	if !errors.Is(err, service.ErrNotSignedUp) {
		t.Fatalf("expected ErrNotSignedUp, got: %v", err)
	}
}

func TestWithdraw_SignedUp(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	occSvc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)
	userSvc := service.NewUserService(userRepo, oooRepo, partRepo)

	user := createUser(t, userSvc, "signed", "signed@test.com")
	occ, err := occSvc.CreateOccurrence(ctx, domain.Occurrence{
		Title: "Test", Description: "d", Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := occSvc.SignUp(ctx, occ.ID, user.ID); err != nil {
		t.Fatal(err)
	}

	err = occSvc.Withdraw(ctx, occ.ID, user.ID)
	if err != nil {
		t.Fatalf("withdraw after signup should succeed: %v", err)
	}

	parts, err := partRepo.FindByOccurrence(ctx, occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 0 {
		t.Fatalf("expected 0 participants after withdraw, got %d", len(parts))
	}
}

// --- Bug #11: CreateOccurrence with max_participants=0 ---

func TestCreateOccurrence_MaxParticipantsZero(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	svc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)

	_, err := svc.CreateOccurrence(ctx, domain.Occurrence{
		Title: "Zero Max", Description: "d", Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 0, MaxParticipants: 0,
	})
	if !errors.Is(err, service.ErrInvalidMaxParticipants) {
		t.Fatalf("expected ErrInvalidMaxParticipants for max=0, got: %v", err)
	}
}

func TestCreateOccurrence_MaxParticipantsNegative(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	svc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)

	_, err := svc.CreateOccurrence(ctx, domain.Occurrence{
		Title: "Neg Max", Description: "d", Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: -1,
	})
	if !errors.Is(err, service.ErrInvalidMaxParticipants) {
		t.Fatalf("expected ErrInvalidMaxParticipants for max=-1, got: %v", err)
	}
}

func TestCreateOccurrence_MaxParticipantsOne(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	svc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)

	occ, err := svc.CreateOccurrence(ctx, domain.Occurrence{
		Title: "One Max", Description: "d", Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		MinParticipants: 1, MaxParticipants: 1,
	})
	if err != nil {
		t.Fatalf("max=1 should succeed: %v", err)
	}
	if occ.ID == 0 {
		t.Fatal("occurrence should be created")
	}
}
