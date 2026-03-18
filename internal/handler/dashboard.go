package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/service"
)

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
	top, _ := h.occurrences.GetLeaderboard(c.Request.Context(), time.Time{}, time.Time{})
	if len(top) > 5 {
		top = top[:5]
	}
	Page(c, "dashboard.html", pageData(c, gin.H{
		"OpenOccurrences": open,
		"Leaderboard":     top,
		"ActivePage":      "dashboard",
		"PageTitle":       "Dashboard",
	}))
}
