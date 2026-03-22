package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/i18n"
	"github.com/robinlant/occurance-management/internal/repository"
	"github.com/robinlant/occurance-management/internal/service"
)

type OccurrenceHandler struct {
	occurrences *service.OccurrenceService
	groups      *service.GroupService
	comments    repository.CommentRepository
}

func NewOccurrenceHandler(occ *service.OccurrenceService, grp *service.GroupService, comments repository.CommentRepository) *OccurrenceHandler {
	return &OccurrenceHandler{occurrences: occ, groups: grp, comments: comments}
}

type OccurrenceListItem struct {
	domain.Occurrence
	ParticipantCount int
	Status           string // "under" | "good" | "over"
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
	statusFilter := c.Query("status") // "under" | "good" | "over"
	hidePast := c.Query("hide_past") == "1"

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

	counts, err := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	now := time.Now()
	items := make([]OccurrenceListItem, 0, len(occs))
	for _, o := range occs {
		if hidePast && o.Date.Before(now) {
			continue
		}
		count := counts[o.ID]
		status := service.ComputeOccStatus(o, count)
		if statusFilter != "" && status != statusFilter {
			continue
		}
		items = append(items, OccurrenceListItem{
			Occurrence:       o,
			ParticipantCount: count,
			Status:           status,
		})
	}

	// Sort: upcoming ASC (earliest first), then past DESC (most recent past first)
	sort.SliceStable(items, func(i, j int) bool {
		iPast := items[i].Date.Before(now)
		jPast := items[j].Date.Before(now)
		if iPast != jPast {
			return !iPast // upcoming before past
		}
		if iPast {
			return items[i].Date.After(items[j].Date) // most recent past first
		}
		return items[i].Date.Before(items[j].Date) // earliest upcoming first
	})

	Page(c, "occurrences.html", pageData(c, gin.H{
		"Occurrences":  items,
		"Groups":       groupMap,
		"GroupList":    groups,
		"ActiveGroup":  activeGroup,
		"StatusFilter": statusFilter,
		"HidePast":     hidePast,
		"ActivePage":   "occurrences",
		"PageTitle":    "Occurrences",
	}))
}

func (h *OccurrenceHandler) ShowCreate(c *gin.Context) {
	groups, _ := h.groups.List(c.Request.Context())
	occs, _ := h.occurrences.ListOccurrences(c.Request.Context())

	defaultDate := time.Now()
	if dateStr := c.Query("date"); dateStr != "" {
		if d, err := time.ParseInLocation("2006-01-02", dateStr, time.Local); err == nil {
			defaultDate = d
		}
	}
	defaultDate = time.Date(defaultDate.Year(), defaultDate.Month(), defaultDate.Day(), 8, 0, 0, 0, defaultDate.Location())

	Page(c, "occurrence_form.html", pageData(c, gin.H{
		"Groups":          groups,
		"Occurrence":      nil,
		"GroupID":         int64(0),
		"DefaultDate":     defaultDate.Format("2006-01-02T15:04"),
		"ActivePage":      "occurrences",
		"PageTitle":       "New Occurrence",
		"PastOccurrences": occs,
	}))
}

func (h *OccurrenceHandler) Create(c *gin.Context) {
	occ, err := h.occurrenceFromForm(c)
	if err != nil {
		SetFlash(c, "error", err.Error())
		c.Redirect(http.StatusFound, "/occurrences/new")
		return
	}
	if occ.Date.Before(time.Now()) {
		SetFlash(c, "error", "Date cannot be in the past.")
		c.Redirect(http.StatusFound, "/occurrences/new")
		return
	}
	created, err := h.occurrences.CreateOccurrence(c.Request.Context(), occ)
	if err != nil {
		slog.Error("occurrence: create failed", "error", err)
		SetFlash(c, "error", "Failed to create occurrence.")
		c.Redirect(http.StatusFound, "/occurrences/new")
		return
	}
	user, _ := CurrentUser(c)
	slog.Info("occurrence_created", "user_id", user.ID, "occurrence_id", created.ID)
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
		SetFlash(c, "error", err.Error())
		c.Redirect(http.StatusFound, "/occurrences/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	occ.ID = id
	updated, err := h.occurrences.UpdateOccurrence(c.Request.Context(), occ)
	if err != nil {
		slog.Error("occurrence: update failed", "occurrence_id", id, "error", err)
		SetFlash(c, "error", "Failed to update occurrence.")
		c.Redirect(http.StatusFound, "/occurrences/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	user, _ := CurrentUser(c)
	slog.Info("occurrence_updated", "user_id", user.ID, "occurrence_id", updated.ID)
	SetFlash(c, "success", "Occurrence updated.")
	c.Redirect(http.StatusFound, "/occurrences/"+strconv.FormatInt(id, 10))
}

func (h *OccurrenceHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	user, _ := CurrentUser(c)
	if err := h.occurrences.DeleteOccurrence(c.Request.Context(), id); err != nil {
		slog.Error("occurrence: delete failed", "user_id", user.ID, "occurrence_id", id, "error", err)
		SetFlash(c, "error", "Failed to delete occurrence.")
	} else {
		slog.Info("occurrence_deleted", "user_id", user.ID, "occurrence_id", id)
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
	isFull := len(participants) >= occ.MaxParticipants
	status := service.ComputeOccStatus(occ, len(participants))

	var group *domain.Group
	if occ.GroupID != 0 {
		g, err := h.groups.GetByID(c.Request.Context(), occ.GroupID)
		if err == nil {
			group = &g
		}
	}

	comments, _ := h.comments.FindByOccurrence(c.Request.Context(), id)

	Page(c, "occurrence_detail.html", pageData(c, gin.H{
		"Occurrence":   occ,
		"Group":        group,
		"Participants": participants,
		"IsSignedUp":   isSignedUp,
		"IsFull":       isFull,
		"Status":       status,
		"ActivePage":   "occurrences",
		"PageTitle":    occ.Title,
		"Comments":     comments,
		"OccurrenceID": id,
	}), "participant_list.html", "comment_list.html")
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
			slog.Warn("signup: user is OOO", "user_id", user.ID, "occurrence_id", id)
			c.Header("HX-Reswap", "none")
			c.String(http.StatusConflict, "You are out of office on this date.")
			return
		}
		if errors.Is(err, service.ErrAlreadySignedUp) {
			c.Header("HX-Reswap", "none")
			c.Status(http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrOccurrenceFull) {
			slog.Warn("signup: occurrence full", "user_id", user.ID, "occurrence_id", id)
			c.Header("HX-Reswap", "none")
			c.String(http.StatusConflict, "This occurrence is full.")
			return
		}
		slog.Error("signup: failed", "user_id", user.ID, "occurrence_id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("occurrence_signup", "user_id", user.ID, "occurrence_id", id)
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
		slog.Error("withdraw: failed", "user_id", user.ID, "occurrence_id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("occurrence_withdraw", "user_id", user.ID, "occurrence_id", id)
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
			slog.Warn("assign: user is OOO", "actor_user_id", currentUser.ID, "user_id", userID, "occurrence_id", id)
			c.Header("HX-Reswap", "none")
			c.String(http.StatusConflict, "User is out of office on this date.")
			return
		}
		slog.Error("assign: failed", "actor_user_id", currentUser.ID, "user_id", userID, "occurrence_id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("participant_assigned", "actor_user_id", currentUser.ID, "user_id", userID, "occurrence_id", id)
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
	currentUser, _ := CurrentUser(c)
	if err := h.occurrences.RemoveParticipant(c.Request.Context(), id, targetUserID); err != nil {
		slog.Error("remove_participant: failed", "actor_user_id", currentUser.ID, "user_id", targetUserID, "occurrence_id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("participant_removed", "actor_user_id", currentUser.ID, "user_id", targetUserID, "occurrence_id", id)
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

func (h *OccurrenceHandler) AddComment(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	user, _ := CurrentUser(c)
	body := strings.TrimSpace(c.PostForm("body"))
	if body == "" {
		h.renderCommentList(c, id)
		return
	}
	if len(body) > 1000 {
		body = body[:1000]
	}
	comment, err := h.comments.Save(c.Request.Context(), domain.Comment{
		OccurrenceID: id,
		UserID:       user.ID,
		Body:         body,
	})
	if err != nil {
		slog.Error("comment: create failed", "user_id", user.ID, "occurrence_id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("comment_created", "user_id", user.ID, "occurrence_id", id, "comment_id", comment.ID)
	h.renderCommentList(c, id)
}

func (h *OccurrenceHandler) DeleteComment(c *gin.Context) {
	occID, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	user, _ := CurrentUser(c)

	comment, err := h.comments.FindByID(c.Request.Context(), cid)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	canDelete := comment.UserID == user.ID || user.Role == domain.RoleAdmin || user.Role == domain.RoleOrganizer
	if !canDelete {
		c.Status(http.StatusForbidden)
		return
	}

	if err := h.comments.Delete(c.Request.Context(), cid); err != nil {
		slog.Error("comment: delete failed", "user_id", user.ID, "comment_id", cid, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("comment_deleted", "user_id", user.ID, "comment_id", cid, "occurrence_id", occID)
	h.renderCommentList(c, occID)
}

func (h *OccurrenceHandler) renderCommentList(c *gin.Context, occID int64) {
	comments, _ := h.comments.FindByOccurrence(c.Request.Context(), occID)
	user, _ := CurrentUser(c)
	lang := i18n.GetLang(c)
	Partial(c, "comment_list.html", gin.H{
		"OccurrenceID": occID,
		"Comments":     comments,
		"CurrentUser":  user,
		"Lang":         lang,
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
		"IsFull":       isOverMax || len(participants) >= occ.MaxParticipants,
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
	if min < 1 || max < 1 {
		return domain.Occurrence{}, errors.New("participants must be at least 1")
	}
	if min > max {
		return domain.Occurrence{}, errors.New("min participants cannot exceed max")
	}
	occ := domain.Occurrence{
		Title:           title,
		Description:     description,
		Date:            date,
		MinParticipants: min,
		MaxParticipants: max,
		AllowOverLimit:  c.PostForm("allow_over_limit") == "on",
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
