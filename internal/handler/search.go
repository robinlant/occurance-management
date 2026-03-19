package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/service"
)

type SearchHandler struct {
	occurrences *service.OccurrenceService
	users       *service.UserService
}

func NewSearchHandler(occ *service.OccurrenceService, users *service.UserService) *SearchHandler {
	return &SearchHandler{occurrences: occ, users: users}
}

type SearchResultOccurrence struct {
	domain.Occurrence
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
		c.Status(http.StatusInternalServerError)
		return
	}
	counts, _ := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())
	occResults := make([]SearchResultOccurrence, 0, len(occs))
	for _, o := range occs {
		count := counts[o.ID]
		status := "good"
		if count < o.MinParticipants {
			status = "under"
		} else if count > o.MaxParticipants {
			status = "over"
		}
		occResults = append(occResults, SearchResultOccurrence{
			Occurrence:       o,
			ParticipantCount: count,
			Status:           status,
		})
	}

	// Search users
	allUsers, _ := h.users.ListUsers(c.Request.Context())
	qLower := strings.ToLower(q)
	var userResults []domain.User
	for _, u := range allUsers {
		if strings.Contains(strings.ToLower(u.Name), qLower) || strings.Contains(strings.ToLower(u.Email), qLower) {
			userResults = append(userResults, u)
			if len(userResults) >= 4 {
				break
			}
		}
	}

	Partial(c, "search_results.html", gin.H{
		"OccResults":  occResults,
		"UserResults": userResults,
		"Query":       q,
	})
}
