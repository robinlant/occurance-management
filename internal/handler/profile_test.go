package handler

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
	"github.com/robinlant/dutyround/internal/service"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Find project root by looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}

	migrationsDir := filepath.Join(dir, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			t.Fatalf("read migration %s: %v", e.Name(), err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			t.Fatalf("exec migration %s: %v", e.Name(), err)
		}
	}
	return db
}

type testEnv struct {
	db        *sql.DB
	handler   *ProfileHandler
	ctx       context.Context
	ginCtx    *gin.Context
	occRepo   *sqlite.OccurrenceRepository
	partRepo  *sqlite.ParticipationRepository
	userRepo  *sqlite.UserRepository
	groupRepo *sqlite.GroupRepository
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)

	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)

	occSvc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)
	groupSvc := service.NewGroupService(groupRepo)
	userSvc := service.NewUserService(userRepo, oooRepo, partRepo)

	handler := NewProfileHandler(userSvc, occSvc, groupSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/profile", nil)

	return &testEnv{
		db:        db,
		handler:   handler,
		ctx:       context.Background(),
		ginCtx:    c,
		occRepo:   occRepo,
		partRepo:  partRepo,
		userRepo:  userRepo,
		groupRepo: groupRepo,
	}
}

// createUser inserts a user and returns the saved domain.User.
func (e *testEnv) createUser(t *testing.T, name string) domain.User {
	t.Helper()
	u, err := e.userRepo.Save(e.ctx, domain.User{
		Name:         name,
		Email:        name + "@test.com",
		Role:         domain.RoleParticipant,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("create user %s: %v", name, err)
	}
	return u
}

// createGroup inserts a group and returns it.
func (e *testEnv) createGroup(t *testing.T, name, color string) domain.Group {
	t.Helper()
	g, err := e.groupRepo.Save(e.ctx, domain.Group{Name: name, Color: color})
	if err != nil {
		t.Fatalf("create group %s: %v", name, err)
	}
	return g
}

// createOccurrence inserts an occurrence and returns it.
func (e *testEnv) createOccurrence(t *testing.T, title string, date time.Time, groupID int64, min, max int) domain.Occurrence {
	t.Helper()
	o, err := e.occRepo.Save(e.ctx, domain.Occurrence{
		Title:           title,
		Date:            date,
		GroupID:         groupID,
		MinParticipants: min,
		MaxParticipants: max,
	})
	if err != nil {
		t.Fatalf("create occurrence %s: %v", title, err)
	}
	return o
}

// signUp links a user to an occurrence via participation.
func (e *testEnv) signUp(t *testing.T, userID, occurrenceID int64) {
	t.Helper()
	_, err := e.partRepo.Save(e.ctx, domain.Participation{
		UserID:       userID,
		OccurrenceID: occurrenceID,
	})
	if err != nil {
		t.Fatalf("sign up user %d for occurrence %d: %v", userID, occurrenceID, err)
	}
}

func TestBuildUserOccurrences_UpcomingBeforePast(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createUser(t, "alice")

	now := time.Now()
	future1 := env.createOccurrence(t, "future1", now.Add(24*time.Hour), 0, 1, 5)
	future2 := env.createOccurrence(t, "future2", now.Add(48*time.Hour), 0, 1, 5)
	past1 := env.createOccurrence(t, "past1", now.Add(-24*time.Hour), 0, 1, 5)
	past2 := env.createOccurrence(t, "past2", now.Add(-48*time.Hour), 0, 1, 5)

	env.signUp(t, user.ID, future1.ID)
	env.signUp(t, user.ID, future2.ID)
	env.signUp(t, user.ID, past1.ID)
	env.signUp(t, user.ID, past2.ID)

	items, _ := env.handler.buildUserOccurrences(env.ginCtx, user.ID)

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	// First two should be future, last two should be past
	for i := 0; i < 2; i++ {
		if items[i].Date.Before(now) {
			t.Errorf("item[%d] %q should be future but is past (date=%v)", i, items[i].Title, items[i].Date)
		}
	}
	for i := 2; i < 4; i++ {
		if !items[i].Date.Before(now) {
			t.Errorf("item[%d] %q should be past but is future (date=%v)", i, items[i].Title, items[i].Date)
		}
	}
}

func TestBuildUserOccurrences_UpcomingOrderedEarliestFirst(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createUser(t, "bob")

	now := time.Now()
	f1 := env.createOccurrence(t, "soon", now.Add(1*time.Hour), 0, 1, 5)
	f2 := env.createOccurrence(t, "later", now.Add(72*time.Hour), 0, 1, 5)
	f3 := env.createOccurrence(t, "mid", now.Add(24*time.Hour), 0, 1, 5)

	env.signUp(t, user.ID, f1.ID)
	env.signUp(t, user.ID, f2.ID)
	env.signUp(t, user.ID, f3.ID)

	items, _ := env.handler.buildUserOccurrences(env.ginCtx, user.ID)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	expected := []string{"soon", "mid", "later"}
	for i, want := range expected {
		if items[i].Title != want {
			t.Errorf("item[%d]: want title %q, got %q", i, want, items[i].Title)
		}
	}
}

func TestBuildUserOccurrences_PastOrderedNewestFirst(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createUser(t, "carol")

	now := time.Now()
	p1 := env.createOccurrence(t, "yesterday", now.Add(-24*time.Hour), 0, 1, 5)
	p2 := env.createOccurrence(t, "last_week", now.Add(-7*24*time.Hour), 0, 1, 5)
	p3 := env.createOccurrence(t, "two_days_ago", now.Add(-48*time.Hour), 0, 1, 5)

	env.signUp(t, user.ID, p1.ID)
	env.signUp(t, user.ID, p2.ID)
	env.signUp(t, user.ID, p3.ID)

	items, _ := env.handler.buildUserOccurrences(env.ginCtx, user.ID)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	expected := []string{"yesterday", "two_days_ago", "last_week"}
	for i, want := range expected {
		if items[i].Title != want {
			t.Errorf("item[%d]: want title %q, got %q", i, want, items[i].Title)
		}
	}
}

func TestBuildUserOccurrences_MixedSorting(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createUser(t, "dave")

	now := time.Now()
	futureEarliest := env.createOccurrence(t, "future_earliest", now.Add(2*time.Hour), 0, 1, 5)
	futureLatest := env.createOccurrence(t, "future_latest", now.Add(48*time.Hour), 0, 1, 5)
	pastNewest := env.createOccurrence(t, "past_newest", now.Add(-1*time.Hour), 0, 1, 5)
	pastMiddle := env.createOccurrence(t, "past_middle", now.Add(-48*time.Hour), 0, 1, 5)
	pastOldest := env.createOccurrence(t, "past_oldest", now.Add(-7*24*time.Hour), 0, 1, 5)

	env.signUp(t, user.ID, futureEarliest.ID)
	env.signUp(t, user.ID, futureLatest.ID)
	env.signUp(t, user.ID, pastNewest.ID)
	env.signUp(t, user.ID, pastMiddle.ID)
	env.signUp(t, user.ID, pastOldest.ID)

	items, _ := env.handler.buildUserOccurrences(env.ginCtx, user.ID)

	if len(items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(items))
	}

	expected := []string{
		"future_earliest",
		"future_latest",
		"past_newest",
		"past_middle",
		"past_oldest",
	}
	for i, want := range expected {
		if items[i].Title != want {
			t.Errorf("item[%d]: want title %q, got %q", i, want, items[i].Title)
		}
	}
}

func TestBuildUserOccurrences_StatusComputation(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createUser(t, "eve")
	extra1 := env.createUser(t, "extra1")
	extra2 := env.createUser(t, "extra2")
	extra3 := env.createUser(t, "extra3")
	extra4 := env.createUser(t, "extra4")

	now := time.Now()

	// "under": 0 participants (only eve signed up = 1), min=2 => count from map is per-occurrence total
	// Actually user "eve" signed up means count=1 for under test. We need count < min.
	// For "under": min=3, max=5, sign up only eve => count=1 < 3 => "under"
	underOcc := env.createOccurrence(t, "under_occ", now.Add(24*time.Hour), 0, 3, 5)
	env.signUp(t, user.ID, underOcc.ID)

	// "good": min=1, max=3, sign up eve + extra1 + extra2 = 3 => count=3, min<=count<=max => "good"
	goodOcc := env.createOccurrence(t, "good_occ", now.Add(48*time.Hour), 0, 1, 3)
	env.signUp(t, user.ID, goodOcc.ID)
	env.signUp(t, extra1.ID, goodOcc.ID)
	env.signUp(t, extra2.ID, goodOcc.ID)

	// "over": min=1, max=2, sign up eve + extra1 + extra2 + extra3 + extra4 = 5 => count=5 > max=2 => "over"
	overOcc := env.createOccurrence(t, "over_occ", now.Add(72*time.Hour), 0, 1, 2)
	env.signUp(t, user.ID, overOcc.ID)
	env.signUp(t, extra1.ID, overOcc.ID)
	env.signUp(t, extra2.ID, overOcc.ID)
	env.signUp(t, extra3.ID, overOcc.ID)
	env.signUp(t, extra4.ID, overOcc.ID)

	items, _ := env.handler.buildUserOccurrences(env.ginCtx, user.ID)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	statusByTitle := make(map[string]string)
	for _, item := range items {
		statusByTitle[item.Title] = item.Status
	}

	tests := []struct {
		title      string
		wantStatus string
	}{
		{"under_occ", "under"},
		{"good_occ", "good"},
		{"over_occ", "over"},
	}
	for _, tc := range tests {
		got := statusByTitle[tc.title]
		if got != tc.wantStatus {
			t.Errorf("occurrence %q: want status %q, got %q", tc.title, tc.wantStatus, got)
		}
	}
}

func TestBuildUserOccurrences_GroupMapConstruction(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createUser(t, "frank")

	g1 := env.createGroup(t, "Engineering", "blue")
	g2 := env.createGroup(t, "Marketing", "red")

	now := time.Now()
	occ1 := env.createOccurrence(t, "eng_duty", now.Add(24*time.Hour), g1.ID, 1, 5)
	occ2 := env.createOccurrence(t, "mkt_duty", now.Add(48*time.Hour), g2.ID, 1, 5)

	env.signUp(t, user.ID, occ1.ID)
	env.signUp(t, user.ID, occ2.ID)

	items, groupMap := env.handler.buildUserOccurrences(env.ginCtx, user.ID)

	// Verify group map has both groups
	if len(groupMap) != 2 {
		t.Fatalf("expected 2 groups in map, got %d", len(groupMap))
	}

	if groupMap[g1.ID].Name != "Engineering" {
		t.Errorf("group %d: want name %q, got %q", g1.ID, "Engineering", groupMap[g1.ID].Name)
	}
	if groupMap[g1.ID].Color != "blue" {
		t.Errorf("group %d: want color %q, got %q", g1.ID, "blue", groupMap[g1.ID].Color)
	}
	if groupMap[g2.ID].Name != "Marketing" {
		t.Errorf("group %d: want name %q, got %q", g2.ID, "Marketing", groupMap[g2.ID].Name)
	}
	if groupMap[g2.ID].Color != "red" {
		t.Errorf("group %d: want color %q, got %q", g2.ID, "red", groupMap[g2.ID].Color)
	}

	// Verify occurrences reference the right groups
	occGroupMap := make(map[string]int64)
	for _, item := range items {
		occGroupMap[item.Title] = item.GroupID
	}
	if occGroupMap["eng_duty"] != g1.ID {
		t.Errorf("eng_duty: want group_id %d, got %d", g1.ID, occGroupMap["eng_duty"])
	}
	if occGroupMap["mkt_duty"] != g2.ID {
		t.Errorf("mkt_duty: want group_id %d, got %d", g2.ID, occGroupMap["mkt_duty"])
	}
}

func TestBuildUserOccurrences_EmptyList(t *testing.T) {
	env := setupTestEnv(t)
	user := env.createUser(t, "ghost")

	items, groupMap := env.handler.buildUserOccurrences(env.ginCtx, user.ID)

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
	// groupMap may or may not be empty depending on whether groups exist,
	// but it should not be nil
	if groupMap == nil {
		t.Error("expected non-nil groupMap, got nil")
	}
}
