package handlers

import (
	"crescendo/services"
	"crescendo/types"
	"crescendo/websocket"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DownloadHandler handles download management endpoints
type DownloadHandler struct {
	jobQueue services.JobQueue
	hub      websocket.Hub
}

// NewDownloadHandler creates a new download handler
func NewDownloadHandler(jq services.JobQueue, hub websocket.Hub) *DownloadHandler {
	return &DownloadHandler{
		jobQueue: jq,
		hub:      hub,
	}
}

// QueueAlbum queues an album download
func (h *DownloadHandler) QueueAlbum(c *gin.Context) {
	albumID := c.Param("id")
	if albumID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "album ID is required",
		})
		return
	}

	job := h.jobQueue.AddJob(types.JobTypeAlbum, albumID, "", "")
	c.JSON(http.StatusCreated, gin.H{
		"message": "Album download queued successfully",
		"job":     job,
	})
}

// QueueTrack queues a track download
func (h *DownloadHandler) QueueTrack(c *gin.Context) {
	trackID := c.Param("id")
	if trackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "track ID is required",
		})
		return
	}

	job := h.jobQueue.AddJob(types.JobTypeTrack, trackID, "", "")
	c.JSON(http.StatusCreated, gin.H{
		"message": "Track download queued successfully",
		"job":     job,
	})
}

// QueueArtist queues an artist discography download
func (h *DownloadHandler) QueueArtist(c *gin.Context) {
	artistID := c.Param("id")
	if artistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "artist ID is required",
		})
		return
	}

	job := h.jobQueue.AddJob(types.JobTypeArtist, artistID, "", "")
	c.JSON(http.StatusCreated, gin.H{
		"message": "Artist discography download queued successfully",
		"job":     job,
	})
}

// GetAllJobs returns all download jobs
func (h *DownloadHandler) GetAllJobs(c *gin.Context) {
	jobs := h.jobQueue.GetAllJobs()
	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

// GetJob returns a specific download job by ID
func (h *DownloadHandler) GetJob(c *gin.Context) {
	jobID := c.Param("jobId")
	job, exists := h.jobQueue.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "job not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job": job,
	})
}

// CancelJob cancels a download job
func (h *DownloadHandler) CancelJob(c *gin.Context) {
	jobID := c.Param("jobId")
	cancelled := h.jobQueue.CancelJob(jobID)
	if !cancelled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job cannot be cancelled (not found or already processing)",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "job cancelled successfully",
	})
}

// HandleWebSocketConnection handles WebSocket connections for specific job progress
func (h *DownloadHandler) HandleWebSocketConnection(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job ID is required"})
		return
	}

	// Check if job exists
	_, exists := h.jobQueue.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	upgrader := websocket.GetUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := websocket.NewClient(h.hub, conn, jobID)
	h.hub.RegisterClient(client)

	// Start client pumps
	client.StartPumps()
}

// HandleWebSocketAllConnection handles WebSocket connections for all job progress
func (h *DownloadHandler) HandleWebSocketAllConnection(c *gin.Context) {
	upgrader := websocket.GetUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := websocket.NewClient(h.hub, conn, "all")
	h.hub.RegisterClient(client)

	// Start client pumps
	client.StartPumps()
}