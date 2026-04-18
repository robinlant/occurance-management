package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/service"
)

type SettingsHandler struct {
	settings *service.SettingsService
	email    *service.EmailService
}

func NewSettingsHandler(settings *service.SettingsService, email *service.EmailService) *SettingsHandler {
	return &SettingsHandler{settings: settings, email: email}
}

func (h *SettingsHandler) Show(c *gin.Context) {
	lang := i18n.GetLang(c)
	all, err := h.settings.GetAll(c.Request.Context())
	if err != nil {
		slog.Error("settings: load failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	Page(c, "settings.html", pageData(c, gin.H{
		"Settings":   all,
		"ActivePage": "settings",
		"PageTitle":  i18n.T(lang, "title.settings"),
	}))
}

func (h *SettingsHandler) Save(c *gin.Context) {
	lang := i18n.GetLang(c)
	settings := map[string]string{
		"smtp_host":              c.PostForm("smtp_host"),
		"smtp_port":              c.PostForm("smtp_port"),
		"smtp_username":          c.PostForm("smtp_username"),
		"smtp_password":          c.PostForm("smtp_password"),
		"sender_email":           c.PostForm("sender_email"),
		"sender_name":            c.PostForm("sender_name"),
		"max_emails_per_day":     c.PostForm("max_emails_per_day"),
		"upcoming_reminder_days": c.PostForm("upcoming_reminder_days"),
	}

	// Toggle: checkbox sends "on" when checked, absent when unchecked
	if c.PostForm("email_enabled") == "on" {
		settings["email_enabled"] = "true"
	} else {
		settings["email_enabled"] = "false"
	}

	if err := h.settings.SaveAll(c.Request.Context(), settings); err != nil {
		SetFlash(c, "error", i18n.T(lang, "flash.failedSaveSettings"))
		c.Redirect(http.StatusFound, "/settings")
		return
	}
	SetFlash(c, "success", i18n.T(lang, "flash.settingsSaved"))
	c.Redirect(http.StatusFound, "/settings")
}

func (h *SettingsHandler) SendTestEmail(c *gin.Context) {
	lang := i18n.GetLang(c)
	user, _ := CurrentUser(c)
	if err := h.email.SendTestEmail(c.Request.Context(), user.Email); err != nil {
		SetFlash(c, "error", i18n.T(lang, "flash.failedSendTestEmail")+err.Error())
	} else {
		SetFlash(c, "success", i18n.T(lang, "flash.testEmailSent")+user.Email)
	}
	c.Redirect(http.StatusFound, "/settings")
}
