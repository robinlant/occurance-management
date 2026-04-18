package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// ---------- FindByID ----------

func TestUserFindByID_Existing(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	saved := mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleParticipant)

	found, err := repo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Name != "alice" || found.Email != "alice@example.com" || found.Role != domain.RoleParticipant {
		t.Errorf("unexpected user: %+v", found)
	}
}

func TestUserFindByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	_, err := repo.FindByID(context.Background(), 99999)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

// ---------- FindByName ----------

func TestUserFindByName_Existing(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	mustInsertUser(t, repo, "bob", "bob@example.com", domain.RoleOrganizer)

	found, err := repo.FindByName(context.Background(), "bob")
	if err != nil {
		t.Fatal(err)
	}
	if found.Email != "bob@example.com" {
		t.Errorf("expected email bob@example.com, got %q", found.Email)
	}
}

func TestUserFindByName_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	_, err := repo.FindByName(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

// ---------- FindByEmail ----------

func TestUserFindByEmail_Existing(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	mustInsertUser(t, repo, "carol", "carol@example.com", domain.RoleParticipant)

	found, err := repo.FindByEmail(context.Background(), "carol@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if found.Name != "carol" {
		t.Errorf("expected name carol, got %q", found.Name)
	}
}

func TestUserFindByEmail_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	_, err := repo.FindByEmail(context.Background(), "nobody@example.com")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

// ---------- FindAll ----------

func TestUserFindAll_Empty(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	users, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

func TestUserFindAll_MultipleUsers(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleAdmin)
	mustInsertUser(t, repo, "bob", "bob@example.com", domain.RoleOrganizer)
	mustInsertUser(t, repo, "carol", "carol@example.com", domain.RoleParticipant)

	users, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
}

// ---------- SearchByNameOrEmail ----------

func TestUserSearchByNameOrEmail_MatchesName(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleParticipant)
	mustInsertUser(t, repo, "bob", "bob@example.com", domain.RoleParticipant)

	results, err := repo.SearchByNameOrEmail(context.Background(), "ali", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "alice" {
		t.Errorf("expected alice, got %q", results[0].Name)
	}
}

func TestUserSearchByNameOrEmail_MatchesEmail(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	mustInsertUser(t, repo, "alice", "alice@special.org", domain.RoleParticipant)
	mustInsertUser(t, repo, "bob", "bob@example.com", domain.RoleParticipant)

	results, err := repo.SearchByNameOrEmail(context.Background(), "special", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "alice" {
		t.Errorf("expected alice, got %q", results[0].Name)
	}
}

func TestUserSearchByNameOrEmail_NoMatch(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleParticipant)

	results, err := repo.SearchByNameOrEmail(context.Background(), "zzz", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestUserSearchByNameOrEmail_RespectsLimit(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	for i := range 5 {
		mustInsertUser(t, repo, "user"+string(rune('a'+i)), "user"+string(rune('a'+i))+"@example.com", domain.RoleParticipant)
	}

	results, err := repo.SearchByNameOrEmail(context.Background(), "user", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (limit), got %d", len(results))
	}
}

func TestUserSearchByNameOrEmail_EmptyQuery(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleParticipant)
	mustInsertUser(t, repo, "bob", "bob@example.com", domain.RoleParticipant)

	// Empty query should match all (LIKE '%%' matches everything).
	results, err := repo.SearchByNameOrEmail(context.Background(), "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for empty query, got %d", len(results))
	}
}

// ---------- Save (update path) ----------

func TestUserSave_UpdateName(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	u := mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleParticipant)

	u.Name = "alice-updated"
	_, err := repo.Save(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}

	found, err := repo.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Name != "alice-updated" {
		t.Errorf("expected name alice-updated, got %q", found.Name)
	}
	// Other fields unchanged.
	if found.Email != "alice@example.com" {
		t.Errorf("email changed unexpectedly: %q", found.Email)
	}
}

func TestUserSave_UpdateRole(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	u := mustInsertUser(t, repo, "bob", "bob@example.com", domain.RoleParticipant)

	u.Role = domain.RoleOrganizer
	_, err := repo.Save(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}

	found, err := repo.FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Role != domain.RoleOrganizer {
		t.Errorf("expected role organizer, got %q", found.Role)
	}
}

// ---------- Delete ----------

func TestUserDelete_Existing(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	u := mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleParticipant)

	if err := repo.Delete(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}

	_, err := repo.FindByID(context.Background(), u.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got: %v", err)
	}
}

func TestUserDelete_NonExistent_NoError(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	// SQLite DELETE of nonexistent row does not error.
	err := repo.Delete(context.Background(), 99999)
	if err != nil {
		t.Fatalf("expected no error for deleting nonexistent user, got: %v", err)
	}
}

func TestUserDelete_PreservesOtherUsers(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewUserRepository(db)

	alice := mustInsertUser(t, repo, "alice", "alice@example.com", domain.RoleParticipant)
	bob := mustInsertUser(t, repo, "bob", "bob@example.com", domain.RoleParticipant)

	if err := repo.Delete(context.Background(), alice.ID); err != nil {
		t.Fatal(err)
	}

	found, err := repo.FindByID(context.Background(), bob.ID)
	if err != nil {
		t.Fatalf("bob should still exist: %v", err)
	}
	if found.Name != "bob" {
		t.Errorf("expected bob, got %q", found.Name)
	}
}
