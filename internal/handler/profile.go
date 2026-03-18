package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/service"
)

type ProfileHandler struct {
	users *service.UserService
}

func NewProfileHandler(users *service.UserService) *ProfileHandler {
	return &ProfileHandler{users: users}
}

func (h *ProfileHandler) Show(c *gin.Context) {
	user, _ := CurrentUser(c)
	ooos, _ := h.users.GetOutOfOffice(c.Request.Context(), user.ID)
	Page(c, "profile.html", pageData(c, gin.H{
		"OOOs":       ooos,
		"ActivePage": "profile",
		"PageTitle":  "Profile",
	}), "ooo_list.html")
}

func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	user, _ := CurrentUser(c)
	if err := h.users.ChangePassword(c.Request.Context(), user.ID, c.PostForm("password")); err != nil {
		SetFlash(c, "error", "Failed to update password.")
	} else {
		SetFlash(c, "success", "Password updated.")
	}
	c.Redirect(http.StatusFound, "/profile")
}

// AddOOO — HTMX: returns updated ooo_list partial.
func (h *ProfileHandler) AddOOO(c *gin.Context) {
	user, _ := CurrentUser(c)

	from, err1 := time.Parse("2006-01-02", c.PostForm("from"))
	to, err2 := time.Parse("2006-01-02", c.PostForm("to"))
	if err1 != nil || err2 != nil || !to.After(from) {
		c.String(http.StatusBadRequest, "Invalid date range.")
		return
	}

	if _, err := h.users.AddOutOfOffice(c.Request.Context(), user.ID, from, to); err != nil {
		if errors.Is(err, service.ErrOOOConflict) {
			c.String(http.StatusConflict, "You have participations assigned in that period.")
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
	if err := h.users.RemoveOutOfOffice(c.Request.Context(), id); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	user, _ := CurrentUser(c)
	h.renderOOOList(c, user.ID)
}

func (h *ProfileHandler) renderOOOList(c *gin.Context, userID int64) {
	ooos, _ := h.users.GetOutOfOffice(c.Request.Context(), userID)
	Partial(c, "ooo_list.html", gin.H{"OOOs": ooos})
}
