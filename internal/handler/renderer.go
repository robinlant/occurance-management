package handler

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	apptmpl "github.com/robinlant/occurance-management/internal/templates"
)

var funcMap = template.FuncMap{
	"formatDate": func(t time.Time) string {
		return t.Format("02.01.2006")
	},
	"formatDateTime": func(t time.Time) string {
		return t.Format("02.01.2006 15:04")
	},
	"formatTime": func(t time.Time) string {
		return t.Format("3:04 PM")
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
}

// Page renders a full page (base layout + page template).
// data must include "CurrentUser" and optionally "Flash".
// Extra partial filenames can be passed to include their definitions (e.g. for inline partials).
func Page(c *gin.Context, page string, data gin.H, extraPartials ...string) {
	patterns := []string{"layouts/base.html", "pages/" + page}
	for _, p := range extraPartials {
		patterns = append(patterns, "partials/"+p)
	}
	t, err := template.New("").Funcs(funcMap).ParseFS(apptmpl.FS, patterns...)
	if err != nil {
		log.Printf("template parse error (%s): %v", page, err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "base", data); err != nil {
		log.Printf("template execute error (%s): %v", page, err)
		// Write visible error — headers already sent so we append to body
		fmt.Fprintf(c.Writer, "<pre style='color:red;padding:1rem'>template error: %v</pre>", err)
	}
}

// Partial renders an HTMX partial (no layout).
// The partial file must contain {{define "partialName"}}...{{end}}.
func Partial(c *gin.Context, partial string, data any) {
	t, err := template.New("").Funcs(funcMap).ParseFS(apptmpl.FS,
		"partials/"+partial,
	)
	if err != nil {
		log.Printf("partial parse error (%s): %v", partial, err)
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	// execute the named define block (strip .html extension for name)
	name := partial[:len(partial)-5]
	if err := t.ExecuteTemplate(c.Writer, name, data); err != nil {
		log.Printf("partial execute error (%s): %v", partial, err)
		fmt.Fprintf(c.Writer, "<pre style='color:red'>partial error: %v</pre>", err)
	}
}
