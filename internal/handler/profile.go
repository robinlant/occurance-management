package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/service"
)

type HeatmapCell struct {
	Date  string
	Count int
	Level int  // 0=none, 1-4
	IsOOO bool // day is within an out-of-office period
}

type ProfileStats struct {
	Total         int
	CurrentStreak int
	LongestStreak int
	Average       string
}

type ProfileHandler struct {
	users       *service.UserService
	occurrences *service.OccurrenceService
	groups      *service.GroupService
}

func NewProfileHandler(users *service.UserService, occ *service.OccurrenceService, grp *service.GroupService) *ProfileHandler {
	return &ProfileHandler{users: users, occurrences: occ, groups: grp}
}

func (h *ProfileHandler) Show(c *gin.Context) {
	user, _ := CurrentUser(c)
	ooos, _ := h.users.GetOutOfOffice(c.Request.Context(), user.ID)

	now := time.Now()
	from := now.AddDate(-1, 0, 0)
	activityMap, _ := h.occurrences.GetActivityHeatmap(c.Request.Context(), user.ID, from, now)

	heatmap, weekCount := buildHeatmap(activityMap, ooos, from, now)
	stats := computeProfileStats(activityMap, from, now)

	occItems, groupMap := h.buildUserOccurrences(c, user.ID)

	Page(c, "profile.html", pageData(c, gin.H{
		"OOOs":           ooos,
		"IsCurrentlyOOO": isCurrentlyOOO(ooos),
		"Heatmap":        heatmap,
		"WeekCount":      weekCount,
		"Stats":          stats,
		"Occurrences":    occItems,
		"Groups":         groupMap,
		"ActivePage":     "profile",
		"PageTitle":      "Profile",
	}), "ooo_list.html")
}

func (h *ProfileHandler) ShowPublic(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// Redirect to own profile so the user can edit it
	if current, ok := CurrentUser(c); ok && current.ID == id {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	user, err := h.users.GetUser(c.Request.Context(), id)
	if err != nil {
		Page(c, "error.html", pageData(c, gin.H{"Code": 404, "Message": "User not found"}))
		return
	}

	now := time.Now()
	from := now.AddDate(-1, 0, 0)
	activityMap, _ := h.occurrences.GetActivityHeatmap(c.Request.Context(), user.ID, from, now)
	ooos, _ := h.users.GetOutOfOffice(c.Request.Context(), user.ID)

	heatmap, weekCount := buildHeatmap(activityMap, ooos, from, now)
	stats := computeProfileStats(activityMap, from, now)

	totalCount, _ := h.occurrences.GetParticipationCount(c.Request.Context(), user.ID)

	occItems, groupMap := h.buildUserOccurrences(c, user.ID)

	Page(c, "user_profile.html", pageData(c, gin.H{
		"ProfileUser":    user,
		"OOOs":           ooos,
		"IsCurrentlyOOO": isCurrentlyOOO(ooos),
		"Heatmap":        heatmap,
		"WeekCount":      weekCount,
		"Stats":          stats,
		"TotalCount":     totalCount,
		"Occurrences":    occItems,
		"Groups":         groupMap,
		"ActivePage":     "",
		"PageTitle":      user.Name,
	}))
}

func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	lang := i18n.GetLang(c)
	user, _ := CurrentUser(c)
	if err := h.users.ChangePassword(c.Request.Context(), user.ID, c.PostForm("current_password"), c.PostForm("password")); err != nil {
		slog.Error("profile: change password failed", "user_id", user.ID, "error", err)
		if errors.Is(err, service.ErrPasswordTooShort) {
			SetFlash(c, "error", i18n.T(lang, "flash.passwordTooShort"))
		} else if errors.Is(err, service.ErrWrongPassword) {
			SetFlash(c, "error", i18n.T(lang, "flash.wrongPassword"))
		} else {
			SetFlash(c, "error", i18n.T(lang, "flash.failedUpdatePassword"))
		}
	} else {
		slog.Info("password_changed", "user_id", user.ID)
		// Regenerate session to invalidate old sessions
		s := sessions.Default(c)
		s.Clear()
		s.Set(sessionUserID, user.ID)
		s.Save()
		SetFlash(c, "success", i18n.T(lang, "flash.passwordUpdated"))
	}
	c.Redirect(http.StatusFound, "/profile")
}

// AddOOO — HTMX: returns updated ooo_list partial.
func (h *ProfileHandler) AddOOO(c *gin.Context) {
	lang := i18n.GetLang(c)
	user, _ := CurrentUser(c)

	from, err1 := time.ParseInLocation("2006-01-02", c.PostForm("from"), time.Local)
	to, err2 := time.ParseInLocation("2006-01-02", c.PostForm("to"), time.Local)
	if err1 != nil || err2 != nil {
		c.Header("HX-Retarget", "#ooo-error")
		c.Header("HX-Reswap", "innerHTML")
		c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; `+i18n.T(lang, "flash.invalidDates")+`</div>`)
		return
	}
	if !to.After(from) {
		c.Header("HX-Retarget", "#ooo-error")
		c.Header("HX-Reswap", "innerHTML")
		c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; `+i18n.T(lang, "flash.endDateAfterStart")+`</div>`)
		return
	}

	ooo, err := h.users.AddOutOfOffice(c.Request.Context(), user.ID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrOOOConflict) {
			slog.Warn("ooo: conflict with existing participation", "user_id", user.ID)
			c.Header("HX-Retarget", "#ooo-error")
			c.Header("HX-Reswap", "innerHTML")
			c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; `+i18n.T(lang, "flash.oooConflictDetail")+`</div>`)
			return
		}
		if errors.Is(err, service.ErrOOOOverlap) {
			slog.Warn("ooo: overlap with existing OOO", "user_id", user.ID)
			c.Header("HX-Retarget", "#ooo-error")
			c.Header("HX-Reswap", "innerHTML")
			c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; `+i18n.T(lang, "flash.oooOverlap")+`</div>`)
			return
		}
		slog.Error("ooo: add failed", "user_id", user.ID, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("ooo_added", "user_id", user.ID, "ooo_id", ooo.ID)
	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}

// DeleteOOO — HTMX: returns updated ooo_list partial.
func (h *ProfileHandler) DeleteOOO(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	user, _ := CurrentUser(c)
	if err := h.users.RemoveOutOfOffice(c.Request.Context(), id, user.ID); err != nil {
		if errors.Is(err, service.ErrOOONotOwner) {
			slog.Warn("ooo: delete denied — not owner", "user_id", user.ID, "ooo_id", id)
			c.Status(http.StatusForbidden)
			return
		}
		slog.Error("ooo: delete failed", "user_id", user.ID, "ooo_id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	slog.Info("ooo_deleted", "user_id", user.ID, "ooo_id", id)
	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}

func (h *ProfileHandler) buildUserOccurrences(c *gin.Context, userID int64) ([]OccurrenceListItem, map[int64]domain.Group) {
	occs, _ := h.occurrences.GetAllForUser(c.Request.Context(), userID)
	counts, _ := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())
	groups, _ := h.groups.List(c.Request.Context())

	groupMap := make(map[int64]domain.Group, len(groups))
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	now := time.Now()
	items := make([]OccurrenceListItem, 0, len(occs))
	for _, o := range occs {
		count := counts[o.ID]
		items = append(items, OccurrenceListItem{
			Occurrence:       o,
			ParticipantCount: count,
			Status:           service.ComputeOccStatus(o, count),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		iPast := items[i].Date.Before(now)
		jPast := items[j].Date.Before(now)
		if iPast != jPast {
			return !iPast
		}
		if iPast {
			return items[i].Date.After(items[j].Date)
		}
		return items[i].Date.Before(items[j].Date)
	})

	return items, groupMap
}

func buildHeatmap(activityMap map[string]int, ooos []domain.OutOfOffice, from, to time.Time) ([]HeatmapCell, int) {
	// Start from Monday on or before `from`
	start := from
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}

	var cells []HeatmapCell
	for d := start; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		count := activityMap[key]
		level := 0
		switch {
		case count >= 4:
			level = 4
		case count >= 3:
			level = 3
		case count >= 2:
			level = 2
		case count >= 1:
			level = 1
		}
		isOOO := false
		dNorm := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		for _, o := range ooos {
			oFrom := time.Date(o.From.Year(), o.From.Month(), o.From.Day(), 0, 0, 0, 0, time.UTC)
			oTo := time.Date(o.To.Year(), o.To.Month(), o.To.Day(), 0, 0, 0, 0, time.UTC)
			if !dNorm.Before(oFrom) && !dNorm.After(oTo) {
				isOOO = true
				break
			}
		}
		cells = append(cells, HeatmapCell{
			Date:  key,
			Count: count,
			Level: level,
			IsOOO: isOOO,
		})
	}

	weekCount := (len(cells) + 6) / 7
	return cells, weekCount
}

func isCurrentlyOOO(ooos []domain.OutOfOffice) bool {
	today := time.Now()
	t := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	for _, o := range ooos {
		oFrom := time.Date(o.From.Year(), o.From.Month(), o.From.Day(), 0, 0, 0, 0, time.UTC)
		oTo := time.Date(o.To.Year(), o.To.Month(), o.To.Day(), 0, 0, 0, 0, time.UTC)
		if !t.Before(oFrom) && !t.After(oTo) {
			return true
		}
	}
	return false
}

func computeProfileStats(activityMap map[string]int, from, to time.Time) ProfileStats {
	var total int

	// Aggregate participations by month
	monthCounts := make(map[string]int)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		count := activityMap[key]
		total += count
		if count > 0 {
			monthKey := d.Format("2006-01")
			monthCounts[monthKey] += count
		}
	}

	// Build sorted list of months in range
	var months []string
	m := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !m.After(end) {
		months = append(months, m.Format("2006-01"))
		m = m.AddDate(0, 1, 0)
	}

	// Compute monthly streaks
	var currentStreak, longestStreak, streak int
	for _, mk := range months {
		if monthCounts[mk] > 0 {
			streak++
			if streak > longestStreak {
				longestStreak = streak
			}
		} else {
			streak = 0
		}
	}
	currentStreak = streak

	// Average per month
	avg := 0.0
	if len(months) > 0 {
		avg = float64(total) / float64(len(months))
	}

	return ProfileStats{
		Total:         total,
		CurrentStreak: currentStreak,
		LongestStreak: longestStreak,
		Average:       fmt.Sprintf("%.1f", avg),
	}
}
