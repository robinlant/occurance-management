package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/service"
)

// CalendarOccurrence wraps an occurrence with participation data for display.
type CalendarOccurrence struct {
	domain.Occurrence
	ParticipantCount int
	Status           string // "under" | "good" | "over"
}

type CalendarDay struct {
	Date           time.Time
	IsToday        bool
	IsCurrentMonth bool
	Occurrences    []CalendarOccurrence
}

type CalendarHandler struct {
	occurrences *service.OccurrenceService
	groups      *service.GroupService
}

func NewCalendarHandler(occ *service.OccurrenceService, grp *service.GroupService) *CalendarHandler {
	return &CalendarHandler{occurrences: occ, groups: grp}
}

func (h *CalendarHandler) Show(c *gin.Context) {
	lang := i18n.GetLang(c)
	month := parseMonth(c.Query("month"))
	statusFilter := c.Query("status") // "", "under", "good", "over"
	hidePast := c.Query("hide_past") == "1"
	var groupFilter int64
	if raw := c.Query("group"); raw != "" {
		groupFilter, _ = strconv.ParseInt(raw, 10, 64)
	}

	allOccs, _ := h.occurrences.ListOccurrences(c.Request.Context())
	counts, _ := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())
	groups, _ := h.groups.List(c.Request.Context())
	groupMap := make(map[int64]domain.Group, len(groups))
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	now := time.Now()
	// Build CalendarOccurrences, applying group + status + hidePast filters.
	var calOccs []CalendarOccurrence
	for _, o := range allOccs {
		if groupFilter != 0 && o.GroupID != groupFilter {
			continue
		}
		if hidePast && o.Date.Before(now) {
			continue
		}
		count := counts[o.ID]
		status := service.ComputeOccStatus(o, count)
		if statusFilter != "" && status != statusFilter {
			continue
		}
		calOccs = append(calOccs, CalendarOccurrence{
			Occurrence:       o,
			ParticipantCount: count,
			Status:           status,
		})
	}

	occByDate := groupCalOccsByDate(calOccs)
	weeks := buildCalendar(month, occByDate)

	prevMonth := time.Date(month.Year(), month.Month()-1, 1, 0, 0, 0, 0, time.Local)
	nextMonth := time.Date(month.Year(), month.Month()+1, 1, 0, 0, 0, 0, time.Local)

	Page(c, "calendar.html", pageData(c, gin.H{
		"Month":        month,
		"PrevMonth":    prevMonth,
		"NextMonth":    nextMonth,
		"Weeks":        weeks,
		"StatusFilter": statusFilter,
		"HidePast":     hidePast,
		"GroupFilter":  groupFilter,
		"GroupList":    groups,
		"Groups":       groupMap,
		"ActivePage":   "calendar",
		"PageTitle":    i18n.T(lang, "title.calendar"),
	}))
}

func (h *CalendarHandler) DayOccurrences(c *gin.Context) {
	date, err := time.ParseInLocation("2006-01-02", c.Query("date"), time.Local)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	dayOccsList, err := h.occurrences.ListOccurrencesByDate(c.Request.Context(), date)
	if err != nil {
		slog.Error("calendar: list occurrences by date failed", "date", date.Format("2006-01-02"), "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	counts, err := h.occurrences.GetParticipantCountsByOccurrence(c.Request.Context())
	if err != nil {
		slog.Error("calendar: get participant counts failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	var dayOccs []CalendarOccurrence
	for _, o := range dayOccsList {
		count := counts[o.ID]
		dayOccs = append(dayOccs, CalendarOccurrence{
			Occurrence:       o,
			ParticipantCount: count,
			Status:           service.ComputeOccStatus(o, count),
		})
	}
	currentUser, _ := CurrentUser(c)
	Partial(c, "day_occurrences.html", gin.H{
		"Date":        date,
		"Occurrences": dayOccs,
		"CurrentUser": currentUser,
	})
}

func parseMonth(s string) time.Time {
	if s == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	t, err := time.ParseInLocation("2006-01", s, time.Local)
	if err != nil {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	return t
}

func groupCalOccsByDate(occs []CalendarOccurrence) map[string][]CalendarOccurrence {
	m := make(map[string][]CalendarOccurrence)
	for _, o := range occs {
		key := o.Date.Format("2006-01-02")
		m[key] = append(m[key], o)
	}
	return m
}

func buildCalendar(month time.Time, occByDate map[string][]CalendarOccurrence) [][]CalendarDay {
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
	lastDay := firstDay.AddDate(0, 1, -1)
	today := time.Now().Truncate(24 * time.Hour)

	start := firstDay
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}

	var weeks [][]CalendarDay
	current := start
	for current.Before(lastDay.AddDate(0, 0, 1)) || len(weeks) == 0 {
		var week []CalendarDay
		for i := 0; i < 7; i++ {
			key := current.Format("2006-01-02")
			week = append(week, CalendarDay{
				Date:           current,
				IsToday:        current.Truncate(24 * time.Hour).Equal(today),
				IsCurrentMonth: current.Month() == month.Month(),
				Occurrences:    occByDate[key],
			})
			current = current.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
	}
	return weeks
}
