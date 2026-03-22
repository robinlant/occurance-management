package handler

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/i18n"
	"github.com/robinlant/occurance-management/internal/repository"
)

type AuthHandler struct {
	users repository.UserRepository
}

func NewAuthHandler(users repository.UserRepository) *AuthHandler {
	return &AuthHandler{users: users}
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	// Redirect already-authenticated users
	s := sessions.Default(c)
	if s.Get(sessionUserID) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	Page(c, "login.html", gin.H{
		"CurrentUser": domain.User{},
		"Flash":       popFlash(c),
		"Lang":        i18n.GetLang(c),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	lang := i18n.GetLang(c)

	user, err := h.users.FindByEmail(c.Request.Context(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("login_failed", "reason", "user_not_found", "ip", c.ClientIP())
			SetFlash(c, "error", i18n.T(lang, "flash.invalidEmailOrPassword"))
			c.Redirect(http.StatusFound, "/login")
			return
		}
		slog.Error("login: db error", "error", err, "ip", c.ClientIP())
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		slog.Warn("login_failed", "reason", "wrong_password", "user_id", user.ID, "ip", c.ClientIP())
		SetFlash(c, "error", i18n.T(lang, "flash.invalidEmailOrPassword"))
		c.Redirect(http.StatusFound, "/login")
		return
	}

	s := sessions.Default(c)
	s.Clear()
	s.Set(sessionUserID, user.ID)
	s.Save()
	slog.Info("login_success", "user_id", user.ID, "ip", c.ClientIP())
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if user, ok := CurrentUser(c); ok {
		slog.Info("logout", "user_id", user.ID, "ip", c.ClientIP())
	}
	s := sessions.Default(c)
	s.Clear()
	s.Save()
	c.Redirect(http.StatusFound, "/login")
}
