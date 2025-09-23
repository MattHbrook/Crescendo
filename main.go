package main

import (
	"flag"
	"fmt"
	"godab/api"
	"godab/config"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	if !api.DirExists(config.GetDownloadLocation()) {
		log.Fatalf("You must provide a valid DOWNLOAD_LOCATION folder")
	}

	asciiArt := `
  ____           _       _     
 / ___| ___   __| | __ _| |__  
| |  _ / _ \ / _\` + "`" + ` |/ _\` + "`" + ` | '_ \ 
| |_| | (_) | (_| | (_| | |_) |
 \____|\___/ \__,_|\__,_|_.__/ 
`

	var (
		album  string
		track  string
		artist string
		server bool
		port   int
	)

	flag.StringVar(&album, "album", "", "Album URL to download")
	flag.StringVar(&track, "track", "", "Track URL to download")
	flag.StringVar(&artist, "artist", "", "Artist URL to download")
	flag.BoolVar(&server, "server", false, "Start in web server mode")
	flag.IntVar(&port, "port", 8080, "Port for web server mode")
	flag.Parse()

	// Server mode takes precedence
	if server {
		startWebServer(port)
		return
	}

	// Existing CLI validation
	if album == "" && track == "" && artist == "" {
		flag.Usage()
		return
	}

	if (album != "" && track != "") || (artist != "" && track != "") || (album != "" && artist != "") {
		log.Fatalf("You can download only one between `album` and `track` at a time.")
		flag.Usage()
	}

	// fmt.Println(asciiArt)
	api.PrintColor(api.COLOR_BLUE, "%s", asciiArt)

	if album != "" {
		album, err := api.NewAlbum(album)

		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		if err := album.Download(true); err != nil {
			log.Fatalf("Cannot download album %s: %s", album.Title, err)
		}
	} else if track != "" {
		track, err := api.NewTrack(track)

		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		if err := track.Download(); err != nil {
			log.Fatalf("Cannot download track %s: %s", track.Title, err)
		}
	} else if artist != "" {
		artist, err := api.NewArtist(artist)

		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		if err := artist.Download(); err != nil {
			log.Fatalf("Cannot download artist %s: %s", artist.Name, err)
		}

	}

	log.Println()
}

// JobType represents the type of download job
type JobType string

const (
	JobTypeAlbum  JobType = "album"
	JobTypeTrack  JobType = "track"
	JobTypeArtist JobType = "artist"
)

// JobStatus represents the current status of a download job
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// DownloadJob represents a download job in the queue
type DownloadJob struct {
	ID          string    `json:"id"`
	Type        JobType   `json:"type"`
	Status      JobStatus `json:"status"`
	ItemID      string    `json:"itemId"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	Progress    int       `json:"progress"`
	Total       int       `json:"total"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// JobQueue manages download jobs
type JobQueue struct {
	jobs        map[string]*DownloadJob
	queue       chan *DownloadJob
	activeJobs  map[string]*DownloadJob
	mu          sync.RWMutex
	maxWorkers  int
	workerCount int
}

// NewJobQueue creates a new job queue
func NewJobQueue(maxWorkers int) *JobQueue {
	return &JobQueue{
		jobs:       make(map[string]*DownloadJob),
		queue:      make(chan *DownloadJob, 100), // Buffer for 100 jobs
		activeJobs: make(map[string]*DownloadJob),
		maxWorkers: maxWorkers,
	}
}

// AddJob adds a new job to the queue
func (jq *JobQueue) AddJob(jobType JobType, itemID, title, artist string) *DownloadJob {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	job := &DownloadJob{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    JobStatusQueued,
		ItemID:    itemID,
		Title:     title,
		Artist:    artist,
		Progress:  0,
		Total:     1,
		CreatedAt: time.Now(),
	}

	jq.jobs[job.ID] = job
	jq.queue <- job

	return job
}

// GetJob retrieves a job by ID
func (jq *JobQueue) GetJob(id string) (*DownloadJob, bool) {
	jq.mu.RLock()
	defer jq.mu.RUnlock()
	job, exists := jq.jobs[id]
	return job, exists
}

// GetAllJobs returns all jobs
func (jq *JobQueue) GetAllJobs() []*DownloadJob {
	jq.mu.RLock()
	defer jq.mu.RUnlock()

	jobs := make([]*DownloadJob, 0, len(jq.jobs))
	for _, job := range jq.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// CancelJob cancels a queued job
func (jq *JobQueue) CancelJob(id string) bool {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	job, exists := jq.jobs[id]
	if !exists {
		return false
	}

	if job.Status == JobStatusQueued {
		job.Status = JobStatusCancelled
		now := time.Now()
		job.CompletedAt = &now
		return true
	}

	return false
}

// UpdateJobProgress updates job progress
func (jq *JobQueue) UpdateJobProgress(id string, progress, total int) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	if job, exists := jq.jobs[id]; exists {
		job.Progress = progress
		job.Total = total
	}
}

// SetJobStatus updates job status
func (jq *JobQueue) SetJobStatus(id string, status JobStatus, errorMsg string) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	if job, exists := jq.jobs[id]; exists {
		job.Status = status
		if errorMsg != "" {
			job.Error = errorMsg
		}

		now := time.Now()
		if status == JobStatusProcessing && job.StartedAt == nil {
			job.StartedAt = &now
			jq.activeJobs[id] = job
		} else if status == JobStatusCompleted || status == JobStatusFailed || status == JobStatusCancelled {
			job.CompletedAt = &now
			delete(jq.activeJobs, id)
		}
	}
}

// Start begins processing jobs
func (jq *JobQueue) Start() {
	for i := 0; i < jq.maxWorkers; i++ {
		go jq.worker()
	}
}

// worker processes jobs from the queue
func (jq *JobQueue) worker() {
	for job := range jq.queue {
		if job.Status == JobStatusCancelled {
			continue
		}

		jq.SetJobStatus(job.ID, JobStatusProcessing, "")

		var err error
		switch job.Type {
		case JobTypeAlbum:
			err = jq.processAlbumJob(job)
		case JobTypeTrack:
			err = jq.processTrackJob(job)
		case JobTypeArtist:
			err = jq.processArtistJob(job)
		}

		if err != nil {
			jq.SetJobStatus(job.ID, JobStatusFailed, err.Error())
			log.Printf("Job %s failed: %v", job.ID, err)
		} else {
			jq.SetJobStatus(job.ID, JobStatusCompleted, "")
			log.Printf("Job %s completed successfully", job.ID)
		}
	}
}

// processAlbumJob processes an album download job
func (jq *JobQueue) processAlbumJob(job *DownloadJob) error {
	album, err := api.NewAlbum(job.ItemID)
	if err != nil {
		return fmt.Errorf("failed to get album metadata: %w", err)
	}

	// Update job with album info
	job.Title = album.Title
	job.Artist = album.Artist
	jq.UpdateJobProgress(job.ID, 0, len(album.Tracks))

	// Download album (this will handle concurrent track downloads internally)
	return album.Download(false) // Don't log to console in web mode
}

// processTrackJob processes a track download job
func (jq *JobQueue) processTrackJob(job *DownloadJob) error {
	track, err := api.NewTrack(job.ItemID)
	if err != nil {
		return fmt.Errorf("failed to get track metadata: %w", err)
	}

	// Update job with track info
	job.Title = track.Title
	job.Artist = track.Artist
	jq.UpdateJobProgress(job.ID, 0, 1)

	// Download track
	err = track.Download()
	if err != nil {
		return fmt.Errorf("failed to download track: %w", err)
	}

	jq.UpdateJobProgress(job.ID, 1, 1)
	return nil
}

// processArtistJob processes an artist discography download job
func (jq *JobQueue) processArtistJob(job *DownloadJob) error {
	artist, err := api.NewArtist(job.ItemID)
	if err != nil {
		return fmt.Errorf("failed to get artist metadata: %w", err)
	}

	// Update job with artist info
	job.Title = fmt.Sprintf("%s (Discography)", artist.Name)
	job.Artist = artist.Name
	jq.UpdateJobProgress(job.ID, 0, len(artist.Albums))

	// Download artist discography
	return artist.Download()
}

// Global job queue instance
var jobQueue *JobQueue

// startWebServer initializes and starts the HTTP server
func startWebServer(port int) {
	// Initialize job queue with max 2 concurrent downloads
	jobQueue = NewJobQueue(2)
	jobQueue.Start()

	// Set Gin to release mode for production
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	// CORS configuration
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000" // Default for React dev
	}

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{corsOrigin}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	r.Use(cors.New(corsConfig))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "crescendo",
			"version":   "1.0.0",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes group
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Crescendo API is running",
				"download_location": config.GetDownloadLocation(),
			})
		})

		// Search endpoint
		apiGroup.GET("/search", func(c *gin.Context) {
			query := c.Query("q")
			searchType := c.DefaultQuery("type", "track") // Default to track search

			if query == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "query parameter 'q' is required",
				})
				return
			}

			// Validate search type
			if searchType != "track" && searchType != "album" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "type parameter must be 'track' or 'album'",
				})
				return
			}

			// Perform search using existing API function
			results, err := api.Search(&query, searchType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "search failed",
					"details": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"query": query,
				"type": searchType,
				"results": results,
			})
		})

		// Download Management Endpoints
		downloadsGroup := apiGroup.Group("/downloads")
		{
			// Queue album download
			downloadsGroup.POST("/album/:id", func(c *gin.Context) {
				albumID := c.Param("id")
				if albumID == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "album ID is required",
					})
					return
				}

				job := jobQueue.AddJob(JobTypeAlbum, albumID, "", "")
				c.JSON(http.StatusCreated, gin.H{
					"message": "Album download queued successfully",
					"job": job,
				})
			})

			// Queue track download
			downloadsGroup.POST("/track/:id", func(c *gin.Context) {
				trackID := c.Param("id")
				if trackID == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "track ID is required",
					})
					return
				}

				job := jobQueue.AddJob(JobTypeTrack, trackID, "", "")
				c.JSON(http.StatusCreated, gin.H{
					"message": "Track download queued successfully",
					"job": job,
				})
			})

			// Queue artist discography download
			downloadsGroup.POST("/artist/:id", func(c *gin.Context) {
				artistID := c.Param("id")
				if artistID == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "artist ID is required",
					})
					return
				}

				job := jobQueue.AddJob(JobTypeArtist, artistID, "", "")
				c.JSON(http.StatusCreated, gin.H{
					"message": "Artist discography download queued successfully",
					"job": job,
				})
			})

			// Get all download jobs
			downloadsGroup.GET("", func(c *gin.Context) {
				jobs := jobQueue.GetAllJobs()
				c.JSON(http.StatusOK, gin.H{
					"jobs": jobs,
					"total": len(jobs),
				})
			})

			// Get specific download job by ID
			downloadsGroup.GET("/:jobId", func(c *gin.Context) {
				jobID := c.Param("jobId")
				job, exists := jobQueue.GetJob(jobID)
				if !exists {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "job not found",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"job": job,
				})
			})

			// Cancel download job
			downloadsGroup.DELETE("/:jobId", func(c *gin.Context) {
				jobID := c.Param("jobId")
				cancelled := jobQueue.CancelJob(jobID)
				if !cancelled {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "job cannot be cancelled (not found or already processing)",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"message": "job cancelled successfully",
				})
			})
		}
	}

	portStr := strconv.Itoa(port)
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort != "" {
		portStr = serverPort
	}

	api.PrintColor(api.COLOR_BLUE, `
  ____                                    _
 / ___|_ __ ___  ___  ___ ___ _ __   __| | ___
| |   | '__/ _ \/ __|/ __/ _ \ '_ \ / _` + "`" + ` |/ _ \
| |___| | |  __/\__ \ (_|  __/ | | | (_| | (_) |
 \____|_|  \___||___/\___\___|_| |_|\__,_|\___/

`)

	log.Printf("🚀 Crescendo web server starting on port %s", portStr)
	log.Printf("📁 Download location: %s", config.GetDownloadLocation())
	log.Printf("🌐 Health check: http://localhost:%s/health", portStr)
	log.Printf("🔗 API status: http://localhost:%s/api/status", portStr)

	if err := r.Run(":" + portStr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
