package cmd

import (
	"crescendo/handlers"
	"crescendo/middleware"
	"crescendo/services"
	"crescendo/websocket"
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// StartWebServer starts the web server
func StartWebServer(port int) {
	// Set production mode if not specified
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize services
	hub := websocket.NewHub()
	go hub.Run()

	jobQueue := services.NewJobQueue(2, hub)
	jobQueue.Start()

	fileService := services.NewFileService()

	// Initialize handlers
	downloadHandler := handlers.NewDownloadHandler(jobQueue, hub)
	fileHandler := handlers.NewFileHandler(fileService)
	searchHandler := handlers.NewSearchHandler()
	healthHandler := handlers.NewHealthHandler()
	settingsHandler := handlers.NewSettingsHandler()

	// Setup router
	r := gin.Default()

	// Apply middleware
	r.Use(middleware.CORS())
	r.Use(middleware.Logging())
	r.Use(middleware.Security())

	// Setup routes
	setupRoutes(r, downloadHandler, fileHandler, searchHandler, healthHandler, settingsHandler)

	// Start server
	portStr := strconv.Itoa(port)
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		portStr = serverPort
	}

	log.Printf("Crescendo web server starting on port %s", portStr)
	if err := r.Run(":" + portStr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupRoutes configures all the HTTP routes
func setupRoutes(r *gin.Engine, downloadHandler *handlers.DownloadHandler, fileHandler *handlers.FileHandler, searchHandler *handlers.SearchHandler, healthHandler *handlers.HealthHandler, settingsHandler *handlers.SettingsHandler) {
	// Health check endpoint
	r.GET("/health", healthHandler.HealthCheck)

	// API routes group
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/status", healthHandler.APIStatus)

		// Search endpoint
		apiGroup.GET("/search", searchHandler.Search)

		// Download Management Endpoints
		downloadsGroup := apiGroup.Group("/downloads")
		{
			// Queue downloads
			downloadsGroup.POST("/album/:id", downloadHandler.QueueAlbum)
			downloadsGroup.POST("/track/:id", downloadHandler.QueueTrack)
			downloadsGroup.POST("/artist/:id", downloadHandler.QueueArtist)

			// Manage downloads
			downloadsGroup.GET("", downloadHandler.GetAllJobs)
			downloadsGroup.GET("/:jobId", downloadHandler.GetJob)
			downloadsGroup.DELETE("/:jobId", downloadHandler.CancelJob)
		}

		// WebSocket endpoints for real-time progress
		wsGroup := apiGroup.Group("/ws")
		{
			// WebSocket endpoint for specific job progress
			wsGroup.GET("/downloads/:jobId", downloadHandler.HandleWebSocketConnection)

			// WebSocket endpoint for all downloads progress
			wsGroup.GET("/downloads", downloadHandler.HandleWebSocketAllConnection)
		}

		// File discovery and streaming endpoints
		apiGroup.GET("/files", fileHandler.ListFiles)
		apiGroup.GET("/files/stream/*filepath", fileHandler.StreamFile)

		// Settings endpoints
		apiGroup.GET("/settings", settingsHandler.GetSettings)
		apiGroup.POST("/settings", settingsHandler.UpdateSettings)
	}
}