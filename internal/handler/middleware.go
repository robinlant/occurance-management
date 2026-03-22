package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository"
)

const sessionCSRFToken = "csrf_token"

// CSRFMiddleware validates CSRF tokens on POST requests and generates tokens for GET requests.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s := sessions.Default(c)

		// Ensure a CSRF token exists in the session
		token, _ := s.Get(sessionCSRFToken).(string)
		if token == "" {
			token = generateCSRFToken()
			s.Set(sessionCSRFToken, token)
			s.Save()
		}

		// Store token in context for templates
		c.Set("csrf_token", token)

		// Validate on POST requests (skip login which has no prior token)
		if c.Request.Method == "POST" {
			// Skip CSRF check for login (no session yet) and HTMX requests (same-origin enforced by browser)
			if c.Request.URL.Path != "/login" && c.Request.URL.Path != "/logout" && c.GetHeader("HX-Request") != "true" {
				formToken := c.PostForm("_csrf")
				if formToken == "" || formToken != token {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
		}

		c.Next()
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

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
