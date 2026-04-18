package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository"
)

const sessionCSRFToken = "csrf_token"

// RequestLogger logs method, path, status, duration, user_id, and IP for every request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		userID := int64(0)
		if u, ok := CurrentUser(c); ok {
			userID = u.ID
		}

		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"user_id", userID,
			"ip", c.ClientIP(),
		)
	}
}

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

		// Validate on POST requests (skip login/logout which have no prior token)
		if c.Request.Method == "POST" {
			if c.Request.URL.Path != "/login" && c.Request.URL.Path != "/logout" {
				formToken := c.PostForm("_csrf")
				if formToken == "" || formToken != token {
					slog.Warn("csrf: token mismatch", "path", c.Request.URL.Path, "ip", c.ClientIP())
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
				slog.Warn("auth: session user not found", "user_id", userID)
				redirectToLogin(c)
				return
			}
			slog.Error("auth: db error looking up session user", "user_id", userID, "error", err)
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
			slog.Warn("authz: access denied", "user_id", user.ID, "role", user.Role, "path", c.Request.URL.Path)
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
