package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
)

type ErrorHandler struct{}

func NewErrorHandler() *ErrorHandler { return &ErrorHandler{} }

func (h *ErrorHandler) NotFound(c *gin.Context) {
	c.Status(http.StatusNotFound)
	Page(c, "error.html", gin.H{
		"CurrentUser": domain.User{},
		"Code":        404,
		"Message":     "Page not found",
	})
}

func (h *ErrorHandler) InternalError(c *gin.Context, err error) {
	c.Status(http.StatusInternalServerError)
	Page(c, "error.html", gin.H{
		"CurrentUser": domain.User{},
		"Code":        500,
		"Message":     "Something went wrong",
	})
}
