package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/service"
)

type LeaderboardHandler struct {
	occurrences *service.OccurrenceService
	groups      *service.GroupService
}

func NewLeaderboardHandler(occ *service.OccurrenceService, grp *service.GroupService) *LeaderboardHandler {
	return &LeaderboardHandler{occurrences: occ, groups: grp}
}

func (h *LeaderboardHandler) Show(c *gin.Context) {
	now := time.Now()
	syFrom, syTo := studentYearDates(now)
	tyFrom, tyTo := thisYearDates(now)

	from, to := parseDateRange(c)
	if from.IsZero() && to.IsZero() {
		from, to = syFrom, syTo
	}

	// Role filter: "participants" (default), "organizers", "all"
	roleFilter := c.Query("role_filter")
	if roleFilter == "" {
		roleFilter = "participants"
	}
	roles := leaderboardRoles(roleFilter)

	// Group filter
	groupIDStr := c.Query("group")
	groupID, _ := strconv.ParseInt(groupIDStr, 10, 64)

	entries, err := h.occurrences.GetLeaderboard(c.Request.Context(), from, to, roles, groupID)
	if err != nil {
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
		"PageTitle":         "Leaderboard",
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
		return []domain.Role{domain.RoleOrganizer}
	case "all":
		return []domain.Role{domain.RoleParticipant, domain.RoleOrganizer}
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

func parseDateRange(c *gin.Context) (time.Time, time.Time) {
	from, _ := time.Parse("2006-01-02", c.Query("from"))
	to, _ := time.Parse("2006-01-02", c.Query("to"))
	return from, to
}

func formatDateInput(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
