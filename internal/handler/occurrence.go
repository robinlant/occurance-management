package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/service"
)

type OccurrenceHandler struct {
	occurrences *service.OccurrenceService
	groups      *service.GroupService
}

func NewOccurrenceHandler(occ *service.OccurrenceService, grp *service.GroupService) *OccurrenceHandler {
	return &OccurrenceHandler{occurrences: occ, groups: grp}
}

func (h *OccurrenceHandler) List(c *gin.Context) {
	groups, _ := h.groups.List(c.Request.Context())
	groupMap := make(map[int64]domain.Group, len(groups))
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	var activeGroup int64
	if raw := c.Query("group"); raw != "" {
		if gid, err := strconv.ParseInt(raw, 10, 64); err == nil {
			activeGroup = gid
		}
	}

	var occs []domain.Occurrence
	var err error
	if activeGroup != 0 {
		occs, err = h.occurrences.ListOccurrencesByGroup(c.Request.Context(), activeGroup)
	} else {
		occs, err = h.occurrences.ListOccurrences(c.Request.Context())
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	Page(c, "occurrences.html", pageData(c, gin.H{
		"Occurrences": occs,
		"Groups":      groupMap,
		"GroupList":   groups,
		"ActiveGroup": activeGroup,
		"ActivePage":  "occurrences",
		"PageTitle":   "Occurrences",
	}))
}

func (h *OccurrenceHandler) ShowCreate(c *gin.Context) {
	groups, _ := h.groups.List(c.Request.Context())
	Page(c, "occurrence_form.html", pageData(c, gin.H{
		"Groups":          groups,
		"Occurrence":      nil,
		"GroupID":         int64(0),
		"ActivePage":      "occurrences",
		"PageTitle":       "New Occurrence",
	}))
}

func (h *OccurrenceHandler) Create(c *gin.Context) {
	occ, err := h.occurrenceFromForm(c)
	if err != nil {
		SetFlash(c, "error", "Invalid form data.")
		c.Redirect(http.StatusFound, "/occurrences/new")
		return
	}
	if _, err := h.occurrences.CreateOccurrence(c.Request.Context(), occ); err != nil {
		SetFlash(c, "error", "Failed to create occurrence.")
		c.Redirect(http.StatusFound, "/occurrences/new")
		return
	}
	SetFlash(c, "success", "Occurrence created.")
	c.Redirect(http.StatusFound, "/occurrences")
}

func (h *OccurrenceHandler) ShowEdit(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	occ, err := h.occurrences.GetOccurrence(c.Request.Context(), id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	groups, _ := h.groups.List(c.Request.Context())
	Page(c, "occurrence_form.html", pageData(c, gin.H{
		"Groups":     groups,
		"Occurrence": occ,
		"GroupID":    occ.GroupID,
		"ActivePage": "occurrences",
		"PageTitle":  "Edit Occurrence",
	}))
}

func (h *OccurrenceHandler) Update(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	occ, err := h.occurrenceFromForm(c)
	if err != nil {
		SetFlash(c, "error", "Invalid form data.")
		c.Redirect(http.StatusFound, "/occurrences/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	occ.ID = id
	if _, err := h.occurrences.UpdateOccurrence(c.Request.Context(), occ); err != nil {
		SetFlash(c, "error", "Failed to update occurrence.")
		c.Redirect(http.StatusFound, "/occurrences/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	SetFlash(c, "success", "Occurrence updated.")
	c.Redirect(http.StatusFound, "/occurrences/"+strconv.FormatInt(id, 10))
}

func (h *OccurrenceHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := h.occurrences.DeleteOccurrence(c.Request.Context(), id); err != nil {
		SetFlash(c, "error", "Failed to delete occurrence.")
	} else {
		SetFlash(c, "success", "Occurrence deleted.")
	}
	c.Redirect(http.StatusFound, "/occurrences")
}

func (h *OccurrenceHandler) Detail(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	occ, err := h.occurrences.GetOccurrence(c.Request.Context(), id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	currentUser, _ := CurrentUser(c)
	participants, _ := h.occurrences.GetParticipants(c.Request.Context(), id)
	isSignedUp := containsUser(participants, currentUser.ID)
	isOverMax := len(participants) >= occ.MaxParticipants

	var group *domain.Group
	if occ.GroupID != 0 {
		g, err := h.groups.GetByID(c.Request.Context(), occ.GroupID)
		if err == nil {
			group = &g
		}
	}

	Page(c, "occurrence_detail.html", pageData(c, gin.H{
		"Occurrence":   occ,
		"Group":        group,
		"Participants": participants,
		"IsSignedUp":   isSignedUp,
		"IsOverMax":    isOverMax,
		"ActivePage":   "occurrences",
		"PageTitle":    occ.Title,
	}), "participant_list.html")
}

// --- HTMX actions ---

func (h *OccurrenceHandler) SignUp(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	user, _ := CurrentUser(c)
	isOverMax, err := h.occurrences.SignUp(c.Request.Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, service.ErrUserOOO) {
			c.Header("HX-Reswap", "none")
			c.String(http.StatusConflict, "You are out of office on this date.")
			return
		}
		if errors.Is(err, service.ErrAlreadySignedUp) {
			c.Header("HX-Reswap", "none")
			c.Status(http.StatusConflict)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	h.renderParticipantList(c, id, user.ID, isOverMax)
}

func (h *OccurrenceHandler) Withdraw(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	user, _ := CurrentUser(c)
	if err := h.occurrences.Withdraw(c.Request.Context(), id, user.ID); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	h.renderParticipantList(c, id, user.ID, false)
}

func (h *OccurrenceHandler) Assign(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseInt(c.PostForm("user_id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	currentUser, _ := CurrentUser(c)
	isOverMax, err := h.occurrences.AssignParticipant(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserOOO) {
			c.Header("HX-Reswap", "none")
			c.String(http.StatusConflict, "User is out of office on this date.")
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	h.renderParticipantList(c, id, currentUser.ID, isOverMax)
}

func (h *OccurrenceHandler) RemoveParticipant(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	targetUserID, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := h.occurrences.RemoveParticipant(c.Request.Context(), id, targetUserID); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	currentUser, _ := CurrentUser(c)
	h.renderParticipantList(c, id, currentUser.ID, false)
}

func (h *OccurrenceHandler) AvailableUsers(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	occ, err := h.occurrences.GetOccurrence(c.Request.Context(), id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	available, err := h.occurrences.GetAvailableUsersForDate(c.Request.Context(), occ.Date)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	Partial(c, "available_users.html", gin.H{
		"OccurrenceID": id,
		"Users":        available,
	})
}

// --- helpers ---

func (h *OccurrenceHandler) renderParticipantList(c *gin.Context, occID, currentUserID int64, isOverMax bool) {
	occ, _ := h.occurrences.GetOccurrence(c.Request.Context(), occID)
	participants, _ := h.occurrences.GetParticipants(c.Request.Context(), occID)
	currentUser, _ := CurrentUser(c)
	Partial(c, "participant_list.html", gin.H{
		"Occurrence":   occ,
		"Participants": participants,
		"CurrentUser":  currentUser,
		"IsSignedUp":   containsUser(participants, currentUserID),
		"IsOverMax":    isOverMax || len(participants) >= occ.MaxParticipants,
	})
}

func (h *OccurrenceHandler) occurrenceFromForm(c *gin.Context) (domain.Occurrence, error) {
	title := c.PostForm("title")
	description := c.PostForm("description")
	dateStr := c.PostForm("date")
	minStr := c.PostForm("min_participants")
	maxStr := c.PostForm("max_participants")

	date, err := time.Parse("2006-01-02T15:04", dateStr)
	if err != nil {
		return domain.Occurrence{}, err
	}
	min, err := strconv.Atoi(minStr)
	if err != nil {
		return domain.Occurrence{}, err
	}
	max, err := strconv.Atoi(maxStr)
	if err != nil {
		return domain.Occurrence{}, err
	}
	occ := domain.Occurrence{
		Title:           title,
		Description:     description,
		Date:            date,
		MinParticipants: min,
		MaxParticipants: max,
	}
	if groupIDStr := c.PostForm("group_id"); groupIDStr != "" {
		gid, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err == nil {
			occ.GroupID = gid
		}
	}
	return occ, nil
}

func pathID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

func containsUser(users []domain.User, userID int64) bool {
	for _, u := range users {
		if u.ID == userID {
			return true
		}
	}
	return false
}
