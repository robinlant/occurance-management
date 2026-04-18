package handler

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/i18n"
	apptmpl "github.com/robinlant/dutyround/internal/templates"
)

var funcMap = template.FuncMap{
	"formatDate": func(t time.Time) string {
		return t.Format("02.01.2006")
	},
	"formatDateTime": func(t time.Time) string {
		return t.Format("02.01.2006 15:04")
	},
	"formatTime": func(t time.Time) string {
		return t.Format("15:04")
	},
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"initial": func(s string) string {
		if len(s) == 0 {
			return "?"
		}
		r := []rune(s)
		return string(r[0])
	},
	"formatISO": func(t time.Time) string {
		return t.Format("2006-01-02")
	},
	"formatMonth": func(t time.Time) string {
		return t.Format("2006-01")
	},
	"monthName": func(t time.Time) string {
		return t.Format("January 2006")
	},
	"dayNum": func(t time.Time) string {
		return t.Format("2")
	},
	"isPast": func(t time.Time) bool {
		return t.Before(time.Now())
	},
	"username": func(email string) string {
		if i := strings.Index(email, "@"); i > 0 {
			return "@" + email[:i]
		}
		return "@" + email
	},
	"multiply": func(a, b int) int { return a * b },
	"percent": func(value, max int) int {
		if max == 0 {
			return 0
		}
		return value * 100 / max
	},
	"formatFloat": func(f float64) string {
		return fmt.Sprintf("%.1f", f)
	},
	"daysUntil": func(t time.Time) int {
		now := time.Now().Truncate(24 * time.Hour)
		target := t.Truncate(24 * time.Hour)
		days := int(target.Sub(now).Hours() / 24)
		return days
	},
	"relativeDay": func(lang string, t time.Time) string {
		d := int(t.Truncate(24*time.Hour).Sub(time.Now().Truncate(24*time.Hour)).Hours() / 24)
		switch {
		case d == 0:
			return i18n.T(lang, "rel.today")
		case d == 1:
			return i18n.T(lang, "rel.tomorrow")
		case d == -1:
			return i18n.T(lang, "rel.yesterday")
		case d > 1:
			return fmt.Sprintf(i18n.T(lang, "rel.inDays"), d)
		default:
			return fmt.Sprintf(i18n.T(lang, "rel.daysAgo"), -d)
		}
	},
	"weekday": func(t time.Time) string {
		return t.Weekday().String()
	},
	"t": func(lang, key string) string {
		return i18n.T(lang, key)
	},
	"monthNameT": func(lang string, t time.Time) string {
		if lang == "de" {
			months := [...]string{
				"Januar", "Februar", "M\u00e4rz", "April", "Mai", "Juni",
				"Juli", "August", "September", "Oktober", "November", "Dezember",
			}
			return months[t.Month()-1] + " " + t.Format("2006")
		}
		return t.Format("January 2006")
	},
	"weekdayT": func(lang string, t time.Time) string {
		if lang == "de" {
			days := [...]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}
			return days[t.Weekday()]
		}
		return t.Weekday().String()
	},
	"list": func(args ...string) []string { return args },
}

// templateCache caches parsed templates to avoid re-parsing on every request.
var (
	templateCache   = make(map[string]*template.Template)
	templateCacheMu sync.RWMutex
)

func getCachedTemplate(key string, patterns []string) (*template.Template, error) {
	templateCacheMu.RLock()
	t, ok := templateCache[key]
	templateCacheMu.RUnlock()
	if ok {
		return t, nil
	}

	t, err := template.New("").Funcs(funcMap).ParseFS(apptmpl.FS, patterns...)
	if err != nil {
		return nil, err
	}

	templateCacheMu.Lock()
	templateCache[key] = t
	templateCacheMu.Unlock()
	return t, nil
}

// Page renders a full page (base layout + page template).
// data must include "CurrentUser" and optionally "Flash".
// Extra partial filenames can be passed to include their definitions (e.g. for inline partials).
func Page(c *gin.Context, page string, data gin.H, extraPartials ...string) {
	patterns := []string{"layouts/base.html", "partials/icons.html", "pages/" + page}
	for _, p := range extraPartials {
		patterns = append(patterns, "partials/"+p)
	}
	cacheKey := strings.Join(patterns, "|")
	t, err := getCachedTemplate(cacheKey, patterns)
	if err != nil {
		slog.Error("renderer: template parse error", "page", page, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "base", data); err != nil {
		slog.Error("renderer: template execute error", "page", page, "error", err)
	}
}

// Partial renders an HTMX partial (no layout).
// The partial file must contain {{define "partialName"}}...{{end}}.
// If data is a gin.H and lacks "Lang", it is injected automatically.
func Partial(c *gin.Context, partial string, data any) {
	if m, ok := data.(gin.H); ok {
		if _, exists := m["Lang"]; !exists {
			m["Lang"] = i18n.GetLang(c)
		}
	}
	patterns := []string{"partials/icons.html", "partials/" + partial}
	cacheKey := "partial:" + partial
	t, err := getCachedTemplate(cacheKey, patterns)
	if err != nil {
		slog.Error("renderer: partial parse error", "partial", partial, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	// execute the named define block (strip .html extension for name)
	name := partial[:len(partial)-5]
	if err := t.ExecuteTemplate(c.Writer, name, data); err != nil {
		slog.Error("renderer: partial execute error", "partial", partial, "error", err)
	}
}
