package handler

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/service"
)

type LeaderboardHandler struct {
	occurrences *service.OccurrenceService
	groups      *service.GroupService
}

func NewLeaderboardHandler(occ *service.OccurrenceService, grp *service.GroupService) *LeaderboardHandler {
	return &LeaderboardHandler{occurrences: occ, groups: grp}
}

func (h *LeaderboardHandler) Show(c *gin.Context) {
	lang := i18n.GetLang(c)
	now := time.Now()
	syFrom, syTo := studentYearDates(now)
	tyFrom, tyTo := thisYearDates(now)

	from, to, reversed := parseDateRange(c)
	if reversed {
		SetFlash(c, "error", i18n.T(lang, "flash.dateRangeReversed"))
		from, to = syFrom, syTo
	} else if from.IsZero() && to.IsZero() {
		from, to = syFrom, syTo
	}

	// Role filter: "participants" (default), "organizers", "all"
	roleFilter := c.Query("role_filter")
	if domain.ValidateRoleFilter(roleFilter) != nil {
		roleFilter = ""
	}
	if roleFilter == "" {
		roleFilter = "participants"
	}
	roles := leaderboardRoles(roleFilter)

	// Group filter
	groupIDStr := c.Query("group")
	groupID, _ := strconv.ParseInt(groupIDStr, 10, 64)

	entries, err := h.occurrences.GetLeaderboard(c.Request.Context(), from, to, roles, groupID)
	if err != nil {
		slog.Error("leaderboard: query failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	allGroups, _ := h.groups.List(c.Request.Context())

	var maxCount, totalCount int
	for _, e := range entries {
		totalCount += e.Count
		if e.Count > maxCount {
			maxCount = e.Count
		}
	}
	var average float64
	if len(entries) > 0 {
		average = float64(totalCount) / float64(len(entries))
	}

	fromStr := formatDateInput(from)
	toStr := formatDateInput(to)
	syFromStr := formatDateInput(syFrom)
	syToStr := formatDateInput(syTo)
	tyFromStr := formatDateInput(tyFrom)
	tyToStr := formatDateInput(tyTo)

	data := gin.H{
		"Entries":           entries,
		"MaxCount":          maxCount,
		"Average":           average,
		"From":              fromStr,
		"To":                toStr,
		"StudentYearFrom":   syFromStr,
		"StudentYearTo":     syToStr,
		"ThisYearFrom":      tyFromStr,
		"ThisYearTo":        tyToStr,
		"StudentYearActive": fromStr == syFromStr && toStr == syToStr,
		"ThisYearActive":    fromStr == tyFromStr && toStr == tyToStr,
		"RoleFilter":        roleFilter,
		"Groups":            allGroups,
		"ActiveGroup":       groupID,
		"ActivePage":        "leaderboard",
		"PageTitle":         i18n.T(lang, "title.leaderboard"),
	}

	if c.GetHeader("HX-Request") == "true" {
		Partial(c, "leaderboard_table.html", data)
		return
	}
	Page(c, "leaderboard.html", pageData(c, data), "leaderboard_table.html")
}

func leaderboardRoles(filter string) []domain.Role {
	switch filter {
	case "organizers":
		return []domain.Role{domain.RoleOrganizer, domain.RoleAdmin}
	case "all":
		return []domain.Role{domain.RoleParticipant, domain.RoleOrganizer, domain.RoleAdmin}
	default:
		return []domain.Role{domain.RoleParticipant}
	}
}

func studentYearDates(now time.Time) (time.Time, time.Time) {
	y := now.Year()
	if now.Month() < time.September {
		y--
	}
	from := time.Date(y, time.September, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(y+1, time.August, 31, 0, 0, 0, 0, time.UTC)
	return from, to
}

func thisYearDates(now time.Time) (time.Time, time.Time) {
	y := now.Year()
	from := time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(y, time.December, 31, 0, 0, 0, 0, time.UTC)
	return from, to
}

func parseDateRange(c *gin.Context) (from, to time.Time, reversed bool) {
	from, _ = time.ParseInLocation("2006-01-02", c.Query("from"), time.Local)
	to, _ = time.ParseInLocation("2006-01-02", c.Query("to"), time.Local)
	reversed = !from.IsZero() && !to.IsZero() && from.After(to)
	return from, to, reversed
}

func formatDateInput(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// Export generates a pivot-style CSV: rows = occurrences, columns = users.
func (h *LeaderboardHandler) Export(c *gin.Context) {
	lang := i18n.GetLang(c)
	from, to, reversed := parseDateRange(c)
	if reversed {
		SetFlash(c, "error", i18n.T(lang, "flash.dateRangeReversed"))
		c.Redirect(http.StatusFound, "/leaderboard")
		return
	}
	if from.IsZero() || to.IsZero() {
		from, _ = studentYearDates(time.Now())
		_, to = studentYearDates(time.Now())
	}

	roleFilter := c.Query("role_filter")
	if domain.ValidateRoleFilter(roleFilter) != nil {
		roleFilter = ""
	}
	if roleFilter == "" {
		roleFilter = "participants"
	}
	roles := leaderboardRoles(roleFilter)

	groupIDStr := c.Query("group")
	groupID, _ := strconv.ParseInt(groupIDStr, 10, 64)

	rows, err := h.occurrences.GetExportData(c.Request.Context(), from, to, roles, groupID)
	if err != nil {
		slog.Error("export: query failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Get all users matching filters (including those with 0 participations).
	entries, err := h.occurrences.GetLeaderboard(c.Request.Context(), from, to, roles, groupID)
	if err != nil {
		slog.Error("export: leaderboard query failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Get ALL occurrences in range (even those with no participants).
	allOccs, err := h.occurrences.GetOccurrencesInRange(c.Request.Context(), from, to, groupID)
	if err != nil {
		slog.Error("export: occurrences query failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Build group name map.
	allGroups, _ := h.groups.List(c.Request.Context())
	groupNames := make(map[int64]string, len(allGroups))
	for _, g := range allGroups {
		groupNames[g.ID] = g.Name
	}

	// Build pivot: collect unique users and occurrences, then mark participation.
	type occKey struct {
		Date  string
		Title string
		Group string
	}

	// User list from leaderboard (includes 0-count users), sorted alphabetically.
	var userOrder []string
	totals := make(map[string]int)
	for _, e := range entries {
		userOrder = append(userOrder, e.User.Name)
		totals[e.User.Name] = e.Count
	}
	sort.Strings(userOrder)

	// Build occurrence rows from ALL occurrences (not just those with participations).
	var occOrder []occKey
	participated := map[occKey]map[string]bool{}
	for _, o := range allOccs {
		key := occKey{Date: o.Date.Format("02.01.2006"), Title: o.Title, Group: groupNames[o.GroupID]}
		occOrder = append(occOrder, key)
		participated[key] = map[string]bool{}
	}

	// Mark participations.
	for _, r := range rows {
		dateStr := r.OccurrenceDate.Format("02.01.2006")
		key := occKey{Date: dateStr, Title: r.OccurrenceTitle, Group: r.GroupName}
		if participated[key] == nil {
			participated[key] = map[string]bool{}
		}
		participated[key][r.UserName] = true
	}

	// Write CSV with BOM for Excel compatibility.
	filename := fmt.Sprintf("dutyround-export-%s-%s.csv", from.Format("2006-01-02"), to.Format("2006-01-02"))
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	// UTF-8 BOM so Excel interprets encoding correctly.
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)

	// Header row: Date | Duty | Group | User1 | User2 | ...
	header := append([]string{"Date", "Duty", "Group"}, userOrder...)
	w.Write(header)

	// Totals row right under header.
	totalRow := []string{"", "Total", ""}
	for _, user := range userOrder {
		totalRow = append(totalRow, strconv.Itoa(totals[user]))
	}
	w.Write(totalRow)

	// Data rows.
	for _, occ := range occOrder {
		row := []string{occ.Date, occ.Title, occ.Group}
		for _, user := range userOrder {
			if participated[occ][user] {
				row = append(row, "x")
			} else {
				row = append(row, "")
			}
		}
		w.Write(row)
	}

	w.Flush()
}
