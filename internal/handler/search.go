package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/service"
)

type SearchHandler struct {
	occurrences *service.OccurrenceService
	users       *service.UserService
}

func NewSearchHandler(occ *service.OccurrenceService, users *service.UserService) *SearchHandler {
	return &SearchHandler{occurrences: occ, users: users}
}

type SearchResultOccurrence struct {
	ID               int64
	Title            string
	ParticipantCount int
	Status           string
}

func (h *SearchHandler) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		c.String(http.StatusOK, "")
		return
	}

	// Search occurrences
	occs, err := h.occurrences.SearchOccurrences(c.Request.Context(), q, 6)
	if err != nil {
		slog.Error("search: occurrences query failed", "query", q, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	counts, err := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())
	if err != nil {
		slog.Error("search: get participant counts failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	occResults := make([]SearchResultOccurrence, 0, len(occs))
	for _, o := range occs {
		count := counts[o.ID]
		occResults = append(occResults, SearchResultOccurrence{
			ID:               o.ID,
			Title:            o.Title,
			ParticipantCount: count,
			Status:           service.ComputeOccStatus(o, count),
		})
	}

	// Search users via SQL LIKE (no N+1)
	userResults, err := h.users.SearchUsers(c.Request.Context(), q, 4)
	if err != nil {
		slog.Error("search: users query failed", "query", q, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	Partial(c, "search_results.html", gin.H{
		"OccResults":  occResults,
		"UserResults": userResults,
		"Query":       q,
	})
}
