package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/service"
)

type GroupHandler struct {
	groups *service.GroupService
}

func NewGroupHandler(grp *service.GroupService) *GroupHandler {
	return &GroupHandler{groups: grp}
}

func (h *GroupHandler) List(c *gin.Context) {
	lang := i18n.GetLang(c)
	groups, err := h.groups.List(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	Page(c, "groups.html", pageData(c, gin.H{
		"Groups":     groups,
		"ActivePage": "groups",
		"PageTitle":  i18n.T(lang, "title.groups"),
	}))
}

func (h *GroupHandler) Create(c *gin.Context) {
	lang := i18n.GetLang(c)
	name := c.PostForm("name")
	if name == "" {
		SetFlash(c, "error", i18n.T(lang, "flash.nameRequired"))
		c.Redirect(http.StatusFound, "/groups")
		return
	}
	color := c.PostForm("color")
	if err := domain.ValidateColor(color); err != nil {
		SetFlash(c, "error", i18n.T(lang, "flash.invalidColor"))
		c.Redirect(http.StatusFound, "/groups")
		return
	}
	actor, _ := CurrentUser(c)
	created, err := h.groups.Create(c.Request.Context(), name, color)
	if err != nil {
		slog.Error("group: create failed", "actor_user_id", actor.ID, "name", name, "error", err)
		SetFlash(c, "error", i18n.T(lang, "flash.failedCreateGroup"))
	} else {
		slog.Info("group_created", "actor_user_id", actor.ID, "group_id", created.ID)
		SetFlash(c, "success", i18n.T(lang, "flash.groupCreated"))
	}
	c.Redirect(http.StatusFound, "/groups")
}

func (h *GroupHandler) UpdateColor(c *gin.Context) {
	lang := i18n.GetLang(c)
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	color := c.PostForm("color")
	if err := domain.ValidateColor(color); err != nil {
		SetFlash(c, "error", i18n.T(lang, "flash.invalidColor"))
		c.Redirect(http.StatusFound, "/groups")
		return
	}
	actor, _ := CurrentUser(c)
	if err := h.groups.UpdateColor(c.Request.Context(), id, color); err != nil {
		slog.Error("group: update color failed", "actor_user_id", actor.ID, "group_id", id, "error", err)
		SetFlash(c, "error", i18n.T(lang, "flash.failedUpdateGroup"))
	} else {
		slog.Info("group_color_updated", "actor_user_id", actor.ID, "group_id", id, "color", color)
	}
	c.Redirect(http.StatusFound, "/groups")
}

func (h *GroupHandler) Delete(c *gin.Context) {
	lang := i18n.GetLang(c)
	id, err := pathID(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	actor, _ := CurrentUser(c)
	if err := h.groups.Delete(c.Request.Context(), id); err != nil {
		slog.Error("group: delete failed", "actor_user_id", actor.ID, "group_id", id, "error", err)
		SetFlash(c, "error", i18n.T(lang, "flash.failedDeleteGroup"))
	} else {
		slog.Info("group_deleted", "actor_user_id", actor.ID, "group_id", id)
		SetFlash(c, "success", i18n.T(lang, "flash.groupDeleted"))
	}
	c.Redirect(http.StatusFound, "/groups")
}
