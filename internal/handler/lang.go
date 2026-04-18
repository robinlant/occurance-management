package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SwitchLang(c *gin.Context) {
	lang := c.Query("lang")
	if lang != "en" && lang != "de" && lang != "ua" {
		lang = "de"
	}
	c.SetCookie("dr-lang", lang, 86400*365, "/", "", false, false)
	ref := c.Request.Referer()
	if ref == "" {
		ref = "/"
	}
	c.Redirect(http.StatusFound, ref)
}
