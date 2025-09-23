package handlers

import (
	"crescendo/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck returns the health status of the service
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "crescendo",
		"version":   "1.0.0",
		"timestamp": time.Now().Unix(),
	})
}

// APIStatus returns the status of the API
func (h *HealthHandler) APIStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":           "Crescendo API is running",
		"download_location": config.GetDownloadLocation(),
	})
}