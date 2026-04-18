package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// ---------- Save and FindByID ----------

func TestCommentSave_AssignsID(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	c := mustInsertComment(t, commentRepo, occ.ID, user.ID, "Hello world")
	if c.ID == 0 {
		t.Fatal("expected nonzero ID after insert")
	}
}

func TestCommentFindByID_Existing(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	c := mustInsertComment(t, commentRepo, occ.ID, user.ID, "Test body")

	found, err := commentRepo.FindByID(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Body != "Test body" {
		t.Errorf("expected body 'Test body', got %q", found.Body)
	}
	if found.OccurrenceID != occ.ID {
		t.Errorf("expected occurrence_id=%d, got %d", occ.ID, found.OccurrenceID)
	}
	if found.UserID != user.ID {
		t.Errorf("expected user_id=%d, got %d", user.ID, found.UserID)
	}
}

func TestCommentFindByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	commentRepo := sqlite.NewCommentRepository(db)

	_, err := commentRepo.FindByID(context.Background(), 99999)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

// ---------- FindByOccurrence ----------

func TestCommentFindByOccurrence_Multiple(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	alice := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, userRepo, "bob", "bob@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertComment(t, commentRepo, occ.ID, alice.ID, "First comment")
	mustInsertComment(t, commentRepo, occ.ID, bob.ID, "Second comment")

	comments, err := commentRepo.FindByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
}

func TestCommentFindByOccurrence_IncludesUserName(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	mustInsertComment(t, commentRepo, occ.ID, user.ID, "Hello")

	comments, err := commentRepo.FindByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	// This tests the JOIN with users table.
	if comments[0].UserName != "alice" {
		t.Errorf("expected UserName 'alice' (from JOIN), got %q", comments[0].UserName)
	}
}

func TestCommentFindByOccurrence_OrderedByCreatedAt(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertComment(t, commentRepo, occ.ID, user.ID, "First")
	mustInsertComment(t, commentRepo, occ.ID, user.ID, "Second")
	mustInsertComment(t, commentRepo, occ.ID, user.ID, "Third")

	comments, err := commentRepo.FindByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(comments))
	}
	// Ordered ASC by created_at.
	if comments[0].Body != "First" || comments[1].Body != "Second" || comments[2].Body != "Third" {
		t.Errorf("unexpected order: %q, %q, %q", comments[0].Body, comments[1].Body, comments[2].Body)
	}
}

func TestCommentFindByOccurrence_Empty(t *testing.T) {
	db := openTestDB(t)
	commentRepo := sqlite.NewCommentRepository(db)

	comments, err := commentRepo.FindByOccurrence(context.Background(), 99999)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments, got %d", len(comments))
	}
}

func TestCommentFindByOccurrence_Isolation(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ1 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift A", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	occ2 := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift B", Description: "d", Date: date(2026, 4, 2),
		MinParticipants: 1, MaxParticipants: 5,
	})

	mustInsertComment(t, commentRepo, occ1.ID, user.ID, "Comment on A")
	mustInsertComment(t, commentRepo, occ2.ID, user.ID, "Comment on B")

	commentsA, err := commentRepo.FindByOccurrence(context.Background(), occ1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(commentsA) != 1 {
		t.Fatalf("expected 1 comment on occ1, got %d", len(commentsA))
	}
	if commentsA[0].Body != "Comment on A" {
		t.Errorf("expected 'Comment on A', got %q", commentsA[0].Body)
	}
}

// ---------- Delete ----------

func TestCommentDelete_Existing(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})
	c := mustInsertComment(t, commentRepo, occ.ID, user.ID, "To delete")

	if err := commentRepo.Delete(context.Background(), c.ID); err != nil {
		t.Fatal(err)
	}

	_, err := commentRepo.FindByID(context.Background(), c.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got: %v", err)
	}
}

func TestCommentDelete_PreservesOthers(t *testing.T) {
	db := openTestDB(t)
	userRepo := sqlite.NewUserRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	user := mustInsertUser(t, userRepo, "alice", "alice@example.com", domain.RoleParticipant)
	occ := mustInsertOccurrence(t, occRepo, domain.Occurrence{
		Title: "Shift", Description: "d", Date: date(2026, 4, 1),
		MinParticipants: 1, MaxParticipants: 5,
	})

	c1 := mustInsertComment(t, commentRepo, occ.ID, user.ID, "Keep")
	c2 := mustInsertComment(t, commentRepo, occ.ID, user.ID, "Remove")

	if err := commentRepo.Delete(context.Background(), c2.ID); err != nil {
		t.Fatal(err)
	}

	comments, err := commentRepo.FindByOccurrence(context.Background(), occ.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 || comments[0].ID != c1.ID {
		t.Fatalf("expected only c1 to remain, got %+v", comments)
	}
}
