package sqlite_test

import (
	"context"
	"testing"

	"github.com/robinlant/dutyround/internal/repository/sqlite"
)

// ---------- Get / Set ----------

func TestSettingsGetSet_RoundTrip(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)
	ctx := context.Background()

	if err := repo.Set(ctx, "app_name", "DutyRound"); err != nil {
		t.Fatal(err)
	}

	val, err := repo.Get(ctx, "app_name")
	if err != nil {
		t.Fatal(err)
	}
	if val != "DutyRound" {
		t.Errorf("expected 'DutyRound', got %q", val)
	}
}

func TestSettingsGet_NonExistentReturnsEmpty(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)

	val, err := repo.Get(context.Background(), "nonexistent_key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Errorf("expected empty string for nonexistent key, got %q", val)
	}
}

func TestSettingsSet_Upsert(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)
	ctx := context.Background()

	if err := repo.Set(ctx, "color", "blue"); err != nil {
		t.Fatal(err)
	}
	// Overwrite.
	if err := repo.Set(ctx, "color", "red"); err != nil {
		t.Fatal(err)
	}

	val, err := repo.Get(ctx, "color")
	if err != nil {
		t.Fatal(err)
	}
	if val != "red" {
		t.Errorf("expected 'red' after upsert, got %q", val)
	}
}

// ---------- GetAll ----------

func TestSettingsGetAll_IncludesMigrationDefaults(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)

	m, err := repo.GetAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Migrations seed default settings (smtp, email, etc).
	if len(m) == 0 {
		t.Fatal("expected migration-seeded defaults, got empty map")
	}
	// Verify a known default exists.
	if _, ok := m["email_enabled"]; !ok {
		t.Error("expected email_enabled key from migration defaults")
	}
}

func TestSettingsGetAll_IncludesCustomEntries(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)
	ctx := context.Background()

	// Count migration defaults first.
	defaults, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defaultCount := len(defaults)

	repo.Set(ctx, "custom_key1", "val1")
	repo.Set(ctx, "custom_key2", "val2")

	m, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != defaultCount+2 {
		t.Fatalf("expected %d entries (defaults + 2 custom), got %d", defaultCount+2, len(m))
	}
	if m["custom_key1"] != "val1" || m["custom_key2"] != "val2" {
		t.Errorf("custom keys missing or wrong: %v", m)
	}
}

// ---------- SetMultiple ----------

func TestSettingsSetMultiple(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)
	ctx := context.Background()

	err := repo.SetMultiple(ctx, map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	})
	if err != nil {
		t.Fatal(err)
	}

	m, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if m["a"] != "1" || m["b"] != "2" || m["c"] != "3" {
		t.Errorf("unexpected map: %v", m)
	}
}

func TestSettingsSetMultiple_Upsert(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)
	ctx := context.Background()

	// Set initial values.
	repo.Set(ctx, "x", "old")

	// SetMultiple should overwrite existing and add new.
	err := repo.SetMultiple(ctx, map[string]string{
		"x": "new",
		"y": "fresh",
	})
	if err != nil {
		t.Fatal(err)
	}

	m, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if m["x"] != "new" {
		t.Errorf("expected x=new after upsert, got %q", m["x"])
	}
	if m["y"] != "fresh" {
		t.Errorf("expected y=fresh, got %q", m["y"])
	}
}

func TestSettingsSetMultiple_EmptyMap(t *testing.T) {
	db := openTestDB(t)
	repo := sqlite.NewSettingsRepository(db)

	// Should not error on empty input.
	err := repo.SetMultiple(context.Background(), map[string]string{})
	if err != nil {
		t.Fatalf("expected no error for empty map, got: %v", err)
	}
}
