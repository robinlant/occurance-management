package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/service"
)

type GroupHandler struct {
	groups *service.GroupService
}

func NewGroupHandler(grp *service.GroupService) *GroupHandler {
	return &GroupHandler{groups: grp}
}

func (h *GroupHandler) List(c *gin.Context) {
	groups, err := h.groups.List(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	Page(c, "groups.html", pageData(c, gin.H{
		"Groups":     groups,
		"ActivePage": "groups",
		"PageTitle":  "Groups",
	}))
}

func (h *GroupHandler) Create(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		SetFlash(c, "error", "Name is required.")
		c.Redirect(http.StatusFound, "/groups")
		return
	}
	if _, err := h.groups.Create(c.Request.Context(), name); err != nil {
		SetFlash(c, "error", "Failed to create group.")
	} else {
		SetFlash(c, "success", "Group created.")
	}
	c.Redirect(http.StatusFound, "/groups")
}

func (h *GroupHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := h.groups.Delete(c.Request.Context(), id); err != nil {
		SetFlash(c, "error", "Failed to delete group.")
	} else {
		SetFlash(c, "success", "Group deleted.")
	}
	c.Redirect(http.StatusFound, "/groups")
}
