package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/i18n"
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
	counts, err := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	dashOccs := make([]DashboardOccurrence, 0, len(open))
	for _, o := range open {
		count := counts[o.ID]
		status := service.ComputeOccStatus(o, count)
		needed := 0
		if count < o.MinParticipants {
			needed = o.MinParticipants - count
		}
		dashOccs = append(dashOccs, DashboardOccurrence{
			Occurrence:       o,
			ParticipantCount: count,
			Status:           status,
			Needed:           needed,
		})
	}

	top, err := h.occurrences.GetLeaderboard(c.Request.Context(), time.Time{}, time.Time{})
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if len(top) > 5 {
		top = top[:5]
	}
	var maxCount int
	for _, e := range top {
		if e.Count > maxCount {
			maxCount = e.Count
		}
	}
	lang := i18n.GetLang(c)
	Page(c, "dashboard.html", pageData(c, gin.H{
		"OpenOccurrences": dashOccs,
		"Leaderboard":     top,
		"MaxCount":        maxCount,
		"ActivePage":      "dashboard",
		"PageTitle":       i18n.T(lang, "title.dashboard"),
	}))
}
