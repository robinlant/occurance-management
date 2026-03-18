package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository"
)

// AuthRequired loads the session user into context. Redirects to /login if not authenticated.
func AuthRequired(users repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := sessions.Default(c)
		v := s.Get(sessionUserID)
		if v == nil {
			redirectToLogin(c)
			return
		}
		userID, ok := v.(int64)
		if !ok {
			redirectToLogin(c)
			return
		}
		user, err := users.FindByID(c.Request.Context(), userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				redirectToLogin(c)
				return
			}
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Set(ctxUser, user)
		c.Next()
	}
}

// RoleRequired aborts with 403 if the current user does not have one of the allowed roles.
func RoleRequired(roles ...domain.Role) gin.HandlerFunc {
	allowed := make(map[domain.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if _, ok := allowed[user.Role]; !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func redirectToLogin(c *gin.Context) {
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/login")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Redirect(http.StatusFound, "/login")
	c.Abort()
}
