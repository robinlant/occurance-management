package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/service"
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
}

func NewProfileHandler(users *service.UserService, occ *service.OccurrenceService) *ProfileHandler {
	return &ProfileHandler{users: users, occurrences: occ}
}

func (h *ProfileHandler) Show(c *gin.Context) {
	user, _ := CurrentUser(c)
	ooos, _ := h.users.GetOutOfOffice(c.Request.Context(), user.ID)

	now := time.Now()
	from := now.AddDate(-1, 0, 0)
	activityMap, _ := h.occurrences.GetActivityHeatmap(c.Request.Context(), user.ID, from, now)

	heatmap, weekCount := buildHeatmap(activityMap, ooos, from, now)
	stats := computeProfileStats(activityMap, from, now)

	Page(c, "profile.html", pageData(c, gin.H{
		"OOOs":             ooos,
		"IsCurrentlyOOO":   isCurrentlyOOO(ooos),
		"Heatmap":          heatmap,
		"WeekCount":        weekCount,
		"Stats":            stats,
		"ActivePage":       "profile",
		"PageTitle":        "Profile",
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
		c.Status(http.StatusNotFound)
		return
	}

	now := time.Now()
	from := now.AddDate(-1, 0, 0)
	activityMap, _ := h.occurrences.GetActivityHeatmap(c.Request.Context(), user.ID, from, now)
	ooos, _ := h.users.GetOutOfOffice(c.Request.Context(), user.ID)

	heatmap, weekCount := buildHeatmap(activityMap, ooos, from, now)
	stats := computeProfileStats(activityMap, from, now)

	totalCount, _ := h.occurrences.GetParticipationCount(c.Request.Context(), user.ID)

	Page(c, "user_profile.html", pageData(c, gin.H{
		"ProfileUser":    user,
		"OOOs":           ooos,
		"IsCurrentlyOOO": isCurrentlyOOO(ooos),
		"Heatmap":        heatmap,
		"WeekCount":      weekCount,
		"Stats":          stats,
		"TotalCount":     totalCount,
		"ActivePage":     "",
		"PageTitle":      user.Name,
	}))
}

func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	user, _ := CurrentUser(c)
	if err := h.users.ChangePassword(c.Request.Context(), user.ID, c.PostForm("password")); err != nil {
		slog.Error("profile: change password failed", "user_id", user.ID, "error", err)
		if errors.Is(err, service.ErrPasswordTooShort) {
			SetFlash(c, "error", "Password must be at least 8 characters.")
		} else {
			SetFlash(c, "error", "Failed to update password.")
		}
	} else {
		slog.Info("password_changed", "user_id", user.ID)
		// Regenerate session to invalidate old sessions
		s := sessions.Default(c)
		s.Clear()
		s.Set(sessionUserID, user.ID)
		s.Save()
		SetFlash(c, "success", "Password updated.")
	}
	c.Redirect(http.StatusFound, "/profile")
}

// AddOOO — HTMX: returns updated ooo_list partial.
func (h *ProfileHandler) AddOOO(c *gin.Context) {
	user, _ := CurrentUser(c)

	from, err1 := time.Parse("2006-01-02", c.PostForm("from"))
	to, err2 := time.Parse("2006-01-02", c.PostForm("to"))
	if err1 != nil || err2 != nil {
		c.Header("HX-Retarget", "#ooo-error")
		c.Header("HX-Reswap", "innerHTML")
		c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; Invalid dates.</div>`)
		return
	}
	if !to.After(from) {
		c.Header("HX-Retarget", "#ooo-error")
		c.Header("HX-Reswap", "innerHTML")
		c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; End date must be after start date.</div>`)
		return
	}

	ooo, err := h.users.AddOutOfOffice(c.Request.Context(), user.ID, from, to)
	if err != nil {
		if errors.Is(err, service.ErrOOOConflict) {
			slog.Warn("ooo: conflict with existing participation", "user_id", user.ID)
			c.Header("HX-Retarget", "#ooo-error")
			c.Header("HX-Reswap", "innerHTML")
			c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; You are signed up for one or more occurrences during this period. Please withdraw from them first before marking these dates as out of office.</div>`)
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
		if count >= 1 {
			level = 4
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
