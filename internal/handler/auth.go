package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/robinlant/occurance-management/internal/domain"
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
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	user, err := h.users.FindByEmail(c.Request.Context(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SetFlash(c, "error", "Invalid email or password.")
			c.Redirect(http.StatusFound, "/login")
			return
		}
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		SetFlash(c, "error", "Invalid email or password.")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	s := sessions.Default(c)
	s.Set(sessionUserID, user.ID)
	s.Save()
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Save()
	c.Redirect(http.StatusFound, "/login")
}
