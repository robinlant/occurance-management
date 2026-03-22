package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/service"
)

type UserAdminHandler struct {
	users *service.UserService
}

func NewUserAdminHandler(users *service.UserService) *UserAdminHandler {
	return &UserAdminHandler{users: users}
}

func (h *UserAdminHandler) List(c *gin.Context) {
	users, err := h.users.ListUsers(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	Page(c, "users.html", pageData(c, gin.H{
		"Users":      users,
		"ActivePage": "users",
		"PageTitle":  "Users",
	}))
}

func (h *UserAdminHandler) Create(c *gin.Context) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	password := c.PostForm("password")
	role := domain.Role(c.PostForm("role"))

	actor, _ := CurrentUser(c)
	created, err := h.users.CreateUser(c.Request.Context(), name, email, password, role)
	if err != nil {
		msg := "Failed to create user."
		if errors.Is(err, service.ErrPasswordTooShort) {
			msg = "Password must be at least 8 characters."
		} else if errors.Is(err, service.ErrInvalidRole) {
			msg = "Invalid role selected."
		}
		slog.Error("user_admin: create user failed", "actor_user_id", actor.ID, "error", err)
		SetFlash(c, "error", msg)
	} else {
		slog.Info("user_created", "actor_user_id", actor.ID, "user_id", created.ID, "role", created.Role)
		SetFlash(c, "success", "User created.")
	}
	c.Redirect(http.StatusFound, "/users")
}

func (h *UserAdminHandler) SetPassword(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	password := c.PostForm("password")
	actor, _ := CurrentUser(c)
	if err := h.users.SetPassword(c.Request.Context(), id, password); err != nil {
		slog.Error("user_admin: set password failed", "actor_user_id", actor.ID, "user_id", id, "error", err)
		if errors.Is(err, service.ErrPasswordTooShort) {
			SetFlash(c, "error", "Password must be at least 8 characters.")
		} else {
			SetFlash(c, "error", "Failed to set password.")
		}
	} else {
		slog.Info("user_password_set", "actor_user_id", actor.ID, "user_id", id)
		SetFlash(c, "success", "Password updated.")
	}
	c.Redirect(http.StatusFound, "/users")
}

func (h *UserAdminHandler) SetEmail(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	email := c.PostForm("email")
	actor, _ := CurrentUser(c)
	if err := h.users.SetEmail(c.Request.Context(), id, email); err != nil {
		slog.Error("user_admin: set email failed", "actor_user_id", actor.ID, "user_id", id, "error", err)
		if errors.Is(err, service.ErrEmailTaken) {
			SetFlash(c, "error", "That email is already in use.")
		} else {
			SetFlash(c, "error", "Failed to update email.")
		}
	} else {
		slog.Info("user_email_set", "actor_user_id", actor.ID, "user_id", id)
		SetFlash(c, "success", "Email updated.")
	}
	c.Redirect(http.StatusFound, "/users")
}

func (h *UserAdminHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	// Prevent self-deletion
	current, _ := CurrentUser(c)
	if current.ID == id {
		SetFlash(c, "error", "You cannot delete your own account.")
		c.Redirect(http.StatusFound, "/users")
		return
	}
	if err := h.users.DeleteUser(c.Request.Context(), id); err != nil {
		slog.Error("user_admin: delete user failed", "actor_user_id", current.ID, "user_id", id, "error", err)
		SetFlash(c, "error", "Failed to delete user.")
	} else {
		slog.Info("user_deleted", "actor_user_id", current.ID, "user_id", id)
		SetFlash(c, "success", "User deleted.")
	}
	c.Redirect(http.StatusFound, "/users")
}
