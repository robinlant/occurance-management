package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository/sqlite"
	"github.com/robinlant/occurance-management/internal/service"
)

func createUser(t *testing.T, svc *service.UserService, name, email string) domain.User {
	t.Helper()
	// Use CreateUser which hashes the password.
	u, err := svc.CreateUser(context.Background(), name, email, "password1234", domain.RoleParticipant)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// --- OOO Overlap Tests ---

func TestAddOutOfOffice_NoOverlap(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)
	user := createUser(t, svc, "alice", "alice@test.com")

	// Create first OOO: Jan 1 - Jan 10
	_, err := svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("first OOO should succeed: %v", err)
	}

	// Create second OOO with no overlap: Feb 1 - Feb 10
	_, err = svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("non-overlapping OOO should succeed: %v", err)
	}
}

func TestAddOutOfOffice_PartialOverlapAtEnd(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)
	user := createUser(t, svc, "bob", "bob@test.com")

	// Existing OOO: Jan 1 - Jan 10
	_, err := svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}

	// New OOO overlaps at end: Jan 5 - Jan 15
	_, err = svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	)
	if err != service.ErrOOOOverlap {
		t.Fatalf("expected ErrOOOOverlap, got: %v", err)
	}
}

func TestAddOutOfOffice_PartialOverlapAtStart(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)
	user := createUser(t, svc, "carol", "carol@test.com")

	// Existing OOO: Jan 10 - Jan 20
	_, err := svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}

	// New OOO overlaps at start: Jan 5 - Jan 15
	_, err = svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	)
	if err != service.ErrOOOOverlap {
		t.Fatalf("expected ErrOOOOverlap, got: %v", err)
	}
}

func TestAddOutOfOffice_FullyContained(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)
	user := createUser(t, svc, "dave", "dave@test.com")

	// Existing OOO: Jan 1 - Jan 20
	_, err := svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}

	// New OOO fully contained: Jan 5 - Jan 15
	_, err = svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	)
	if err != service.ErrOOOOverlap {
		t.Fatalf("expected ErrOOOOverlap, got: %v", err)
	}
}

func TestAddOutOfOffice_FullyContaining(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)
	user := createUser(t, svc, "eve", "eve@test.com")

	// Existing OOO: Jan 5 - Jan 15
	_, err := svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}

	// New OOO fully containing existing: Jan 1 - Jan 20
	_, err = svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	)
	if err != service.ErrOOOOverlap {
		t.Fatalf("expected ErrOOOOverlap, got: %v", err)
	}
}

func TestAddOutOfOffice_AdjacentAllowed(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)
	user := createUser(t, svc, "frank", "frank@test.com")

	// Existing OOO: Jan 1 - Jan 10
	_, err := svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Adjacent OOO: Jan 10 - Jan 20 (start == end of existing) -- should be ALLOWED
	_, err = svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("adjacent OOO (start == end) should be allowed, got: %v", err)
	}
}

func TestAddOutOfOffice_ExactSamePeriod(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)
	user := createUser(t, svc, "hank", "hank@test.com")

	// Existing OOO: Jan 1 - Jan 10
	_, err := svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Exact same period: Jan 1 - Jan 10
	_, err = svc.AddOutOfOffice(ctx, user.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != service.ErrOOOOverlap {
		t.Fatalf("expected ErrOOOOverlap for exact same period, got: %v", err)
	}
}

func TestAddOutOfOffice_DifferentUsersNoConflict(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)

	svc := service.NewUserService(userRepo, oooRepo, partRepo)

	user1 := createUser(t, svc, "iris", "iris@test.com")
	user2 := createUser(t, svc, "jack", "jack@test.com")

	// User1 OOO: Jan 1 - Jan 10
	_, err := svc.AddOutOfOffice(ctx, user1.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}

	// User2 same period should be fine -- different user
	_, err = svc.AddOutOfOffice(ctx, user2.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("different users should not conflict, got: %v", err)
	}
}
