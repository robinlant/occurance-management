package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/service"
)

type LeaderboardHandler struct {
	occurrences *service.OccurrenceService
}

func NewLeaderboardHandler(occ *service.OccurrenceService) *LeaderboardHandler {
	return &LeaderboardHandler{occurrences: occ}
}

func (h *LeaderboardHandler) Show(c *gin.Context) {
	from, to := parseDateRange(c)
	entries, err := h.occurrences.GetLeaderboard(c.Request.Context(), from, to)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

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

	data := gin.H{
		"Entries":    entries,
		"MaxCount":   maxCount,
		"Average":    average,
		"From":       formatDateInput(from),
		"To":         formatDateInput(to),
		"ActivePage": "leaderboard",
		"PageTitle":  "Leaderboard",
	}

	if c.GetHeader("HX-Request") == "true" {
		Partial(c, "leaderboard_table.html", data)
		return
	}
	Page(c, "leaderboard.html", pageData(c, data), "leaderboard_table.html")
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
