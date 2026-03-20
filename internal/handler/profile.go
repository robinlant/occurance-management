package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/service"
)

type HeatmapCell struct {
	Date  string
	Count int
	Level int // 0=none, 1-4
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

	heatmap, weekCount := buildHeatmap(activityMap, from, now)
	stats := computeProfileStats(activityMap, from, now)

	Page(c, "profile.html", pageData(c, gin.H{
		"OOOs":       ooos,
		"Heatmap":    heatmap,
		"WeekCount":  weekCount,
		"Stats":      stats,
		"ActivePage": "profile",
		"PageTitle":  "Profile",
	}), "ooo_list.html")
}

func (h *ProfileHandler) ShowPublic(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
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

	heatmap, weekCount := buildHeatmap(activityMap, from, now)
	stats := computeProfileStats(activityMap, from, now)

	totalCount, _ := h.occurrences.GetParticipationCount(c.Request.Context(), user.ID)

	Page(c, "user_profile.html", pageData(c, gin.H{
		"ProfileUser": user,
		"Heatmap":     heatmap,
		"WeekCount":   weekCount,
		"Stats":       stats,
		"TotalCount":  totalCount,
		"ActivePage":  "",
		"PageTitle":   user.Name,
	}))
}

func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	user, _ := CurrentUser(c)
	if err := h.users.ChangePassword(c.Request.Context(), user.ID, c.PostForm("password")); err != nil {
		if errors.Is(err, service.ErrPasswordTooShort) {
			SetFlash(c, "error", "Password must be at least 8 characters.")
		} else {
			SetFlash(c, "error", "Failed to update password.")
		}
	} else {
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

	if _, err := h.users.AddOutOfOffice(c.Request.Context(), user.ID, from, to); err != nil {
		if errors.Is(err, service.ErrOOOConflict) {
			c.Header("HX-Retarget", "#ooo-error")
			c.Header("HX-Reswap", "innerHTML")
			c.String(http.StatusOK, `<div class="flash flash-error" style="margin-top:8px">&#10005; You have participations assigned in that period.</div>`)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}

	h.renderOOOList(c, user.ID)
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
			c.Status(http.StatusForbidden)
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}
	h.renderOOOList(c, user.ID)
}

func (h *ProfileHandler) renderOOOList(c *gin.Context, userID int64) {
	ooos, _ := h.users.GetOutOfOffice(c.Request.Context(), userID)
	Partial(c, "ooo_list.html", gin.H{"OOOs": ooos})
}

func buildHeatmap(activityMap map[string]int, from, to time.Time) ([]HeatmapCell, int) {
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
		cells = append(cells, HeatmapCell{
			Date:  key,
			Count: count,
			Level: level,
		})
	}

	weekCount := (len(cells) + 6) / 7
	return cells, weekCount
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
