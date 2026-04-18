package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// ---------- FindByID ----------

func TestGroupFindByID_Existing(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	g := mustInsertGroup(t, repo, "Engineering", "blue")

	found, err := repo.FindByID(context.Background(), g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Name != "Engineering" || found.Color != "blue" {
		t.Errorf("unexpected group: %+v", found)
	}
}

func TestGroupFindByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	_, err := repo.FindByID(context.Background(), 99999)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

// ---------- FindAll ----------

func TestGroupFindAll_Empty(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	groups, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(groups))
	}
}

func TestGroupFindAll_Multiple(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	mustInsertGroup(t, repo, "Engineering", "blue")
	mustInsertGroup(t, repo, "Marketing", "red")
	mustInsertGroup(t, repo, "Design", "purple")

	groups, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
}

// ---------- Save (update path) ----------

func TestGroupSave_UpdateNameAndColor(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	g := mustInsertGroup(t, repo, "Engineering", "blue")

	g.Name = "Platform"
	g.Color = "teal"
	_, err := repo.Save(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}

	found, err := repo.FindByID(context.Background(), g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Name != "Platform" {
		t.Errorf("expected name Platform, got %q", found.Name)
	}
	if found.Color != "teal" {
		t.Errorf("expected color teal, got %q", found.Color)
	}
}

// ---------- Delete ----------

func TestGroupDelete_Existing(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	g := mustInsertGroup(t, repo, "ToDelete", "red")

	if err := repo.Delete(context.Background(), g.ID); err != nil {
		t.Fatal(err)
	}

	_, err := repo.FindByID(context.Background(), g.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got: %v", err)
	}
}

func TestGroupDelete_NonExistent_NoError(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	err := repo.Delete(context.Background(), 99999)
	if err != nil {
		t.Fatalf("expected no error for deleting nonexistent group, got: %v", err)
	}
}

// ---------- Save preserves ID across insert and update ----------

func TestGroupSave_InsertAssignsID(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewGroupRepository(db)

	g, err := repo.Save(context.Background(), domain.Group{Name: "Test", Color: "green"})
	if err != nil {
		t.Fatal(err)
	}
	if g.ID == 0 {
		t.Fatal("expected nonzero ID after insert")
	}
}
