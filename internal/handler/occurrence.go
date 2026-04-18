package handler

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/repository"
	"github.com/robinlant/dutyround/internal/service"
)

type OccurrenceHandler struct {
	occurrences *service.OccurrenceService
	groups      *service.GroupService
	users       *service.UserService
	comments    repository.CommentRepository
}

func NewOccurrenceHandler(occ *service.OccurrenceService, grp *service.GroupService, users *service.UserService, comments repository.CommentRepository) *OccurrenceHandler {
	return &OccurrenceHandler{occurrences: occ, groups: grp, users: users, comments: comments}
}

type OccurrenceListItem struct {
	domain.Occurrence
	ParticipantCount int
	Status           string // "under" | "good" | "over"
}

func (h *OccurrenceHandler) List(c *gin.Context) {
	lang := i18n.GetLang(c)
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
	if domain.ValidateStatusFilter(statusFilter) != nil {
		statusFilter = ""
	}
	hidePast := c.Query("hide_past") != "0"

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
		"PageTitle":    i18n.T(lang, "title.occurrences"),
	}))
}

func (h *OccurrenceHandler) ShowCreate(c *gin.Context) {
	lang := i18n.GetLang(c)
	groups, _ := h.groups.List(c.Request.Context())

	// All occurrences sorted by date descending for the copy-from feature
	allOccs, _ := h.occurrences.ListOccurrences(c.Request.Context())
	sort.Slice(allOccs, func(i, j int) bool {
		return allOccs[i].Date.After(allOccs[j].Date)
	})

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
		"PageTitle":       i18n.T(lang, "title.newOccurrence"),
		"PastOccurrences": allOccs,
	}))
}

func (h *OccurrenceHandler) Create(c *gin.Context) {
	lang := i18n.GetLang(c)
	occ, err := h.occurrenceFromForm(c, lang)
	if err != nil {
		SetFlash(c, "error", err.Error())
		c.Redirect(http.StatusFound, "/duties/new")
		return
	}
	user, _ := CurrentUser(c)

	// Parse recurrence fields
	repeat := c.PostForm("repeat") // "", "daily", "weekly", "biweekly", "monthly"
	if err := domain.ValidateRepeatType(repeat); err != nil {
		SetFlash(c, "error", i18n.T(lang, "flash.invalidRepeatType"))
		c.Redirect(http.StatusFound, "/duties/new")
		return
	}
	untilStr := c.PostForm("repeat_until")

	dates := []time.Time{occ.Date}
	if repeat != "" && untilStr != "" {
		until, err := time.ParseInLocation("2006-01-02", untilStr, time.Local)
		if err != nil {
			SetFlash(c, "error", i18n.T(lang, "flash.invalidFormData"))
			c.Redirect(http.StatusFound, "/duties/new")
			return
		}
		until = time.Date(until.Year(), until.Month(), until.Day(), 23, 59, 59, 0, until.Location())
		if until.Before(occ.Date) {
			SetFlash(c, "error", i18n.T(lang, "flash.untilBeforeDate"))
			c.Redirect(http.StatusFound, "/duties/new")
			return
		}
		dates = generateRecurrenceDates(occ.Date, until, repeat)
		if len(dates) > 365 {
			dates = dates[:365]
		}
	}

	var recurrenceID string
	if len(dates) > 1 {
		recurrenceID = newRecurrenceID()
	}

	createdCount := 0
	for _, d := range dates {
		o := occ
		o.Date = d
		o.RecurrenceID = recurrenceID
		_, err := h.occurrences.CreateOccurrence(c.Request.Context(), o)
		if err != nil {
			slog.Error("occurrence: create failed", "user_id", user.ID, "date", d, "error", err)
			continue
		}
		createdCount++
	}

	if createdCount == 0 {
		SetFlash(c, "error", i18n.T(lang, "flash.failedCreateOccurrence"))
		c.Redirect(http.StatusFound, "/duties/new")
		return
	}

	slog.Info("occurrences_created", "user_id", user.ID, "count", createdCount, "recurrence_id", recurrenceID)
	if createdCount > 1 {
		SetFlash(c, "success", fmt.Sprintf("%d %s", createdCount, i18n.T(lang, "flash.occurrencesCreatedRecurring")))
	} else if occ.Date.Before(time.Now()) {
		SetFlash(c, "warning", i18n.T(lang, "flash.occurrenceCreatedInPast"))
	} else {
		SetFlash(c, "success", i18n.T(lang, "flash.occurrenceCreated"))
	}
	c.Redirect(http.StatusFound, "/duties")
}

func (h *OccurrenceHandler) ShowEdit(c *gin.Context) {
	lang := i18n.GetLang(c)
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	occ, err := h.occurrences.GetOccurrence(c.Request.Context(), id)
	if err != nil {
		Page(c, "error.html", pageData(c, gin.H{"Code": 404, "Message": "Duty not found"}))
		return
	}
	groups, _ := h.groups.List(c.Request.Context())

	var seriesCount int
	if occ.RecurrenceID != "" {
		siblings, _ := h.occurrences.GetSeriesSiblings(c.Request.Context(), occ.RecurrenceID)
		seriesCount = len(siblings)
	}

	Page(c, "occurrence_form.html", pageData(c, gin.H{
		"Groups":      groups,
		"Occurrence":  occ,
		"GroupID":     occ.GroupID,
		"ActivePage":  "occurrences",
		"PageTitle":   i18n.T(lang, "title.editOccurrence"),
		"IsRecurring": occ.RecurrenceID != "",
		"SeriesCount": seriesCount,
	}))
}

func (h *OccurrenceHandler) Update(c *gin.Context) {
	lang := i18n.GetLang(c)
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	occ, err := h.occurrenceFromForm(c, lang)
	if err != nil {
		SetFlash(c, "error", err.Error())
		c.Redirect(http.StatusFound, "/duties/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	occ.ID = id
	existing, err := h.occurrences.GetOccurrence(c.Request.Context(), id)
	if err != nil {
		slog.Error("occurrence: fetch for update failed", "occurrence_id", id, "error", err)
		SetFlash(c, "error", i18n.T(lang, "flash.failedUpdateOccurrence"))
		c.Redirect(http.StatusFound, "/duties/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	occ.RecurrenceID = existing.RecurrenceID

	editScope := c.PostForm("edit_scope") // "single", "future", "all"
	if editScope != "" {
		if err := domain.ValidateEditScope(editScope); err != nil {
			SetFlash(c, "error", i18n.T(lang, "flash.invalidEditScope"))
			c.Redirect(http.StatusFound, "/duties/"+strconv.FormatInt(id, 10)+"/edit")
			return
		}
	}

	if editScope == "future" && existing.RecurrenceID != "" {
		count, err := h.occurrences.UpdateSeriesFromDate(c.Request.Context(), occ, existing.RecurrenceID, existing.Date)
		if err != nil {
			slog.Error("occurrence: series update (future) failed", "occurrence_id", id, "error", err)
			SetFlash(c, "error", i18n.T(lang, "flash.failedUpdateOccurrence"))
			c.Redirect(http.StatusFound, "/duties/"+strconv.FormatInt(id, 10)+"/edit")
			return
		}
		user, _ := CurrentUser(c)
		slog.Info("occurrence_series_updated_future", "user_id", user.ID, "occurrence_id", id, "count", count)
		SetFlash(c, "success", fmt.Sprintf("%d %s", count, i18n.T(lang, "flash.seriesUpdated")))
		c.Redirect(http.StatusFound, "/duties")
		return
	}

	if editScope == "all" && existing.RecurrenceID != "" {
		count, err := h.occurrences.UpdateEntireSeries(c.Request.Context(), occ, existing.RecurrenceID)
		if err != nil {
			slog.Error("occurrence: series update (all) failed", "occurrence_id", id, "error", err)
			SetFlash(c, "error", i18n.T(lang, "flash.failedUpdateOccurrence"))
			c.Redirect(http.StatusFound, "/duties/"+strconv.FormatInt(id, 10)+"/edit")
			return
		}
		user, _ := CurrentUser(c)
		slog.Info("occurrence_series_updated_all", "user_id", user.ID, "occurrence_id", id, "count", count)
		SetFlash(c, "success", fmt.Sprintf("%d %s", count, i18n.T(lang, "flash.seriesUpdated")))
		c.Redirect(http.StatusFound, "/duties")
		return
	}

	// Default: single update
	updated, err := h.occurrences.UpdateOccurrence(c.Request.Context(), occ)
	if err != nil {
		slog.Error("occurrence: update failed", "occurrence_id", id, "error", err)
		SetFlash(c, "error", i18n.T(lang, "flash.failedUpdateOccurrence"))
		c.Redirect(http.StatusFound, "/duties/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	user, _ := CurrentUser(c)
	slog.Info("occurrence_updated", "user_id", user.ID, "occurrence_id", updated.ID)
	SetFlash(c, "success", i18n.T(lang, "flash.occurrenceUpdated"))
	c.Redirect(http.StatusFound, "/duties/"+strconv.FormatInt(id, 10))
}

func (h *OccurrenceHandler) Delete(c *gin.Context) {
	lang := i18n.GetLang(c)
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	user, _ := CurrentUser(c)

	deleteScope := c.PostForm("delete_scope") // "single", "future", "all"
	if deleteScope != "" {
		if err := domain.ValidateDeleteScope(deleteScope); err != nil {
			SetFlash(c, "error", i18n.T(lang, "flash.invalidDeleteScope"))
			c.Redirect(http.StatusFound, "/duties")
			return
		}
	}

	occ, fetchErr := h.occurrences.GetOccurrence(c.Request.Context(), id)

	if deleteScope == "future" && fetchErr == nil && occ.RecurrenceID != "" {
		count, err := h.occurrences.DeleteSeriesFromDate(c.Request.Context(), occ.RecurrenceID, occ.Date)
		if err != nil {
			slog.Error("occurrence: series delete (future) failed", "user_id", user.ID, "occurrence_id", id, "error", err)
			SetFlash(c, "error", i18n.T(lang, "flash.failedDeleteOccurrence"))
		} else {
			slog.Info("occurrence_series_deleted_future", "user_id", user.ID, "occurrence_id", id, "count", count)
			SetFlash(c, "success", fmt.Sprintf("%d %s", count, i18n.T(lang, "flash.seriesDeleted")))
		}
		c.Redirect(http.StatusFound, "/duties")
		return
	}

	if deleteScope == "all" && fetchErr == nil && occ.RecurrenceID != "" {
		_, err := h.occurrences.DeleteEntireSeries(c.Request.Context(), occ.RecurrenceID)
		if err != nil {
			slog.Error("occurrence: series delete (all) failed", "user_id", user.ID, "occurrence_id", id, "error", err)
			SetFlash(c, "error", i18n.T(lang, "flash.failedDeleteOccurrence"))
		} else {
			slog.Info("occurrence_series_deleted_all", "user_id", user.ID, "occurrence_id", id)
			SetFlash(c, "success", i18n.T(lang, "flash.seriesDeletedAll"))
		}
		c.Redirect(http.StatusFound, "/duties")
		return
	}

	// Default: single delete
	if err := h.occurrences.DeleteOccurrence(c.Request.Context(), id); err != nil {
		slog.Error("occurrence: delete failed", "user_id", user.ID, "occurrence_id", id, "error", err)
		SetFlash(c, "error", i18n.T(lang, "flash.failedDeleteOccurrence"))
	} else {
		slog.Info("occurrence_deleted", "user_id", user.ID, "occurrence_id", id)
		SetFlash(c, "success", i18n.T(lang, "flash.occurrenceDeleted"))
	}
	c.Redirect(http.StatusFound, "/duties")
}

func (h *OccurrenceHandler) Detail(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	occ, err := h.occurrences.GetOccurrence(c.Request.Context(), id)
	if err != nil {
		Page(c, "error.html", pageData(c, gin.H{"Code": 404, "Message": "Duty not found"}))
		return
	}
	currentUser, _ := CurrentUser(c)
	participants, err := h.occurrences.GetParticipants(c.Request.Context(), id)
	if err != nil {
		slog.Error("failed to get participants", "occurrence_id", id, "error", err)
		Page(c, "error.html", pageData(c, gin.H{"Code": 500, "Message": "Internal error"}))
		return
	}
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

	comments, err := h.comments.FindByOccurrence(c.Request.Context(), id)
	if err != nil {
		slog.Error("failed to get comments", "occurrence_id", id, "error", err)
	}

	var seriesCount, futureCount int
	if occ.RecurrenceID != "" {
		siblings, _ := h.occurrences.GetSeriesSiblings(c.Request.Context(), occ.RecurrenceID)
		seriesCount = len(siblings)
		for _, s := range siblings {
			if !s.Date.Before(occ.Date) {
				futureCount++
			}
		}
	}

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
		"IsRecurring":  occ.RecurrenceID != "" && seriesCount > 1,
		"SeriesCount":  seriesCount,
		"FutureCount":  futureCount,
	}), "participant_list.html", "comment_list.html")
}

// --- HTMX actions ---

func (h *OccurrenceHandler) SignUp(c *gin.Context) {
	lang := i18n.GetLang(c)
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
			c.String(http.StatusConflict, i18n.T(lang, "flash.userOOOSignup"))
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
			c.String(http.StatusConflict, i18n.T(lang, "flash.occurrenceFull"))
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
	lang := i18n.GetLang(c)
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
	if _, err := h.users.GetUser(c.Request.Context(), userID); err != nil {
		c.Header("HX-Reswap", "none")
		c.String(http.StatusBadRequest, i18n.T(lang, "flash.userNotFound"))
		return
	}
	currentUser, _ := CurrentUser(c)
	isOverMax, err := h.occurrences.AssignParticipant(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserOOO) {
			slog.Warn("assign: user is OOO", "actor_user_id", currentUser.ID, "user_id", userID, "occurrence_id", id)
			c.Header("HX-Reswap", "none")
			c.String(http.StatusConflict, i18n.T(lang, "flash.userOOOAssign"))
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
	if _, err := h.users.GetUser(c.Request.Context(), targetUserID); err != nil {
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
	lang := i18n.GetLang(c)
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
	if len([]rune(body)) > 1000 {
		c.Header("HX-Reswap", "none")
		c.String(http.StatusBadRequest, i18n.T(lang, "flash.commentTooLong"))
		return
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

func (h *OccurrenceHandler) occurrenceFromForm(c *gin.Context, lang string) (domain.Occurrence, error) {
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.titleRequired"))
	}
	if len([]rune(title)) > 255 {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.titleTooLong"))
	}
	description := c.PostForm("description")
	if len([]rune(description)) > 5000 {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.descriptionTooLong"))
	}
	dateStr := c.PostForm("date")
	minStr := c.PostForm("min_participants")
	maxStr := c.PostForm("max_participants")

	// Parse in local timezone so the "date in the past" check is accurate
	date, err := time.ParseInLocation("2006-01-02T15:04", dateStr, time.Local)
	if err != nil {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.invalidFormData"))
	}
	min, err := strconv.Atoi(minStr)
	if err != nil {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.invalidFormData"))
	}
	max, err := strconv.Atoi(maxStr)
	if err != nil {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.invalidFormData"))
	}
	if min < 1 || max < 1 {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.participantsMin"))
	}
	if min > 1000 || max > 1000 {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.participantsMax"))
	}
	if min > max {
		return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.minExceedsMax"))
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
			if _, err := h.groups.GetByID(c.Request.Context(), gid); err != nil {
				return domain.Occurrence{}, errors.New(i18n.T(lang, "flash.groupNotFound"))
			}
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

func generateRecurrenceDates(start, until time.Time, repeat string) []time.Time {
	dates := []time.Time{start}
	cur := start
	for {
		switch repeat {
		case "daily":
			cur = cur.AddDate(0, 0, 1)
		case "weekly":
			cur = cur.AddDate(0, 0, 7)
		case "biweekly":
			cur = cur.AddDate(0, 0, 14)
		case "monthly":
			cur = cur.AddDate(0, 1, 0)
		default:
			return dates
		}
		if cur.After(until) {
			break
		}
		dates = append(dates, cur)
	}
	return dates
}

func newRecurrenceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
