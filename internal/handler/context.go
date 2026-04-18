package handler

import (
	"encoding/gob"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
)

func init() {
	gob.Register(Flash{})
	gob.Register(int64(0))
}

const (
	sessionUserID = "user_id"
	sessionFlash  = "flash"
	ctxUser       = "currentUser"
)

type Flash struct {
	Type    string // "success" | "error" | "warning"
	Message string
}

func CurrentUser(c *gin.Context) (domain.User, bool) {
	u, ok := c.Get(ctxUser)
	if !ok {
		return domain.User{}, false
	}
	return u.(domain.User), true
}

func SetFlash(c *gin.Context, flashType, message string) {
	s := sessions.Default(c)
	s.Set(sessionFlash, Flash{Type: flashType, Message: message})
	s.Save()
}

func popFlash(c *gin.Context) *Flash {
	s := sessions.Default(c)
	v := s.Get(sessionFlash)
	if v == nil {
		return nil
	}
	s.Delete(sessionFlash)
	s.Save()
	f, ok := v.(Flash)
	if !ok {
		return nil
	}
	return &f
}

// pageData builds the base gin.H with CurrentUser, Flash, CSRFToken, and Lang, ready for Page().
func pageData(c *gin.Context, extra gin.H) gin.H {
	user, _ := CurrentUser(c)
	csrfToken, _ := c.Get("csrf_token")
	data := gin.H{
		"CurrentUser": user,
		"Flash":       popFlash(c),
		"CSRFToken":   csrfToken,
		"Lang":        i18n.GetLang(c),
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}
