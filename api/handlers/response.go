package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"study-tracker-go/internal/repository"
)

func respondError(c *gin.Context, status int, err error) {
	if errors.Is(err, repository.ErrDataBusy) {
		c.Header("Retry-After", "1")
		c.JSON(http.StatusServiceUnavailable, gin.H{"detail": repository.ErrDataBusy.Error()})
		return
	}
	c.JSON(status, gin.H{"detail": err.Error()})
}
