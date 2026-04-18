package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/service"
)

type UserAdminHandler struct {
	users *service.UserService
}

func NewUserAdminHandler(users *service.UserService) *UserAdminHandler {
	return &UserAdminHandler{users: users}
}

func (h *UserAdminHandler) List(c *gin.Context) {
	lang := i18n.GetLang(c)
	users, err := h.users.ListUsers(c.Request.Context())
	if err != nil {
		slog.Error("user_admin: list users failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	Page(c, "users.html", pageData(c, gin.H{
		"Users":      users,
		"ActivePage": "users",
		"PageTitle":  i18n.T(lang, "title.users"),
	}))
}

func (h *UserAdminHandler) Create(c *gin.Context) {
	lang := i18n.GetLang(c)
	name := c.PostForm("name")
	email := c.PostForm("email")
	password := c.PostForm("password")
	role := domain.Role(c.PostForm("role"))

	if err := domain.ValidateEmail(email); err != nil {
		SetFlash(c, "error", i18n.T(lang, "flash.invalidEmail"))
		c.Redirect(http.StatusFound, "/users")
		return
	}

	actor, _ := CurrentUser(c)
	created, err := h.users.CreateUser(c.Request.Context(), name, email, password, role)
	if err != nil {
		var msg string
		if errors.Is(err, service.ErrPasswordTooShort) {
			msg = i18n.T(lang, "flash.passwordTooShort")
		} else if errors.Is(err, service.ErrInvalidRole) {
			msg = i18n.T(lang, "flash.invalidRole")
		} else {
			msg = i18n.T(lang, "flash.failedCreateUser")
		}
		slog.Error("user_admin: create user failed", "actor_user_id", actor.ID, "error", err)
		SetFlash(c, "error", msg)
	} else {
		slog.Info("user_created", "actor_user_id", actor.ID, "user_id", created.ID, "role", created.Role)
		SetFlash(c, "success", i18n.T(lang, "flash.userCreated"))
	}
	c.Redirect(http.StatusFound, "/users")
}

func (h *UserAdminHandler) SetPassword(c *gin.Context) {
	lang := i18n.GetLang(c)
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
			SetFlash(c, "error", i18n.T(lang, "flash.passwordTooShort"))
		} else {
			SetFlash(c, "error", i18n.T(lang, "flash.failedSetPassword"))
		}
	} else {
		slog.Info("user_password_set", "actor_user_id", actor.ID, "user_id", id)
		SetFlash(c, "success", i18n.T(lang, "flash.passwordUpdated"))
	}
	c.Redirect(http.StatusFound, "/users")
}

func (h *UserAdminHandler) SetEmail(c *gin.Context) {
	lang := i18n.GetLang(c)
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	email := c.PostForm("email")
	if err := domain.ValidateEmail(email); err != nil {
		SetFlash(c, "error", i18n.T(lang, "flash.invalidEmail"))
		c.Redirect(http.StatusFound, "/users")
		return
	}
	actor, _ := CurrentUser(c)
	if err := h.users.SetEmail(c.Request.Context(), id, email); err != nil {
		slog.Error("user_admin: set email failed", "actor_user_id", actor.ID, "user_id", id, "error", err)
		if errors.Is(err, service.ErrEmailTaken) {
			SetFlash(c, "error", i18n.T(lang, "flash.emailAlreadyInUse"))
		} else {
			SetFlash(c, "error", i18n.T(lang, "flash.failedUpdateEmail"))
		}
	} else {
		slog.Info("user_email_set", "actor_user_id", actor.ID, "user_id", id)
		SetFlash(c, "success", i18n.T(lang, "flash.emailUpdated"))
	}
	c.Redirect(http.StatusFound, "/users")
}

func (h *UserAdminHandler) Delete(c *gin.Context) {
	lang := i18n.GetLang(c)
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	// Prevent self-deletion
	current, _ := CurrentUser(c)
	if current.ID == id {
		SetFlash(c, "error", i18n.T(lang, "flash.cannotDeleteSelf"))
		c.Redirect(http.StatusFound, "/users")
		return
	}
	if err := h.users.DeleteUser(c.Request.Context(), id); err != nil {
		slog.Error("user_admin: delete user failed", "actor_user_id", current.ID, "user_id", id, "error", err)
		SetFlash(c, "error", i18n.T(lang, "flash.failedDeleteUser"))
	} else {
		slog.Info("user_deleted", "actor_user_id", current.ID, "user_id", id)
		SetFlash(c, "success", i18n.T(lang, "flash.userDeleted"))
	}
	c.Redirect(http.StatusFound, "/users")
}
