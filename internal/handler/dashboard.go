package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/service"
)

type DashboardOccurrence struct {
	domain.Occurrence
	ParticipantCount int
	Status           string
	Needed           int
}

type DashboardHandler struct {
	occurrences *service.OccurrenceService
}

func NewDashboardHandler(occ *service.OccurrenceService) *DashboardHandler {
	return &DashboardHandler{occurrences: occ}
}

func (h *DashboardHandler) Show(c *gin.Context) {
	open, err := h.occurrences.ListOpenOccurrences(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	counts, _ := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())

	dashOccs := make([]DashboardOccurrence, 0, len(open))
	for _, o := range open {
		count := counts[o.ID]
		status := "good"
		needed := 0
		if count < o.MinParticipants {
			status = "under"
			needed = o.MinParticipants - count
		} else if count > o.MaxParticipants {
			status = "over"
		}
		dashOccs = append(dashOccs, DashboardOccurrence{
			Occurrence:       o,
			ParticipantCount: count,
			Status:           status,
			Needed:           needed,
		})
	}

	top, _ := h.occurrences.GetLeaderboard(c.Request.Context(), time.Time{}, time.Time{})
	if len(top) > 5 {
		top = top[:5]
	}
	var maxCount int
	for _, e := range top {
		if e.Count > maxCount {
			maxCount = e.Count
		}
	}
	Page(c, "dashboard.html", pageData(c, gin.H{
		"OpenOccurrences": dashOccs,
		"Leaderboard":     top,
		"MaxCount":        maxCount,
		"ActivePage":      "dashboard",
		"PageTitle":       "Dashboard",
	}))
}
