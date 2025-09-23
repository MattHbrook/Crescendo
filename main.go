package main

import (
	"flag"
	"fmt"
	"godab/api"
	"godab/config"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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

// ProgressMessage represents a WebSocket progress update message
type ProgressMessage struct {
	JobID       string    `json:"jobId"`
	Type        string    `json:"type"`        // "progress", "status", "complete", "error"
	Progress    float64   `json:"progress"`    // 0-100 percentage
	Status      string    `json:"status"`      // current job status
	CurrentFile string    `json:"currentFile"` // name of file currently downloading
	Speed       string    `json:"speed"`       // download speed like "2.1 MB/s"
	Message     string    `json:"message,omitempty"` // status or error messages
	Timestamp   time.Time `json:"timestamp"`   // when the update occurred
}

// WebSocket upgrader with CORS support
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		// In production, check against allowed origins
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan ProgressMessage
	jobID  string
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients mapped by job ID
	clients map[string]map[*Client]bool

	// Broadcast channel for sending messages to all clients of a job
	broadcast chan ProgressMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

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

// AudioFile represents a discovered audio file (FLAC, MP3, etc.)
type AudioFile struct {
	Filename string        `json:"filename"`
	Path     string        `json:"path"`
	Size     int64         `json:"size"`
	Format   string        `json:"format"`         // "flac", "mp3", etc.
	Metadata *AudioMetadata `json:"metadata,omitempty"`
}

type AudioMetadata struct {
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	Duration    string `json:"duration,omitempty"`
	TrackNumber int    `json:"trackNumber,omitempty"`
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

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan ProgressMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.jobID] == nil {
				h.clients[client.jobID] = make(map[*Client]bool)
			}
			h.clients[client.jobID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected for job %s", client.jobID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.jobID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.jobID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected for job %s", client.jobID)

		case message := <-h.broadcast:
			h.mu.RLock()
			// Send to specific job clients
			if clients, ok := h.clients[message.JobID]; ok {
				for client := range clients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
				if len(clients) == 0 {
					delete(h.clients, message.JobID)
				}
			}

			// Also send to "all" clients for any job update
			if allClients, ok := h.clients["all"]; ok {
				for client := range allClients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(allClients, client)
					}
				}
				if len(allClients) == 0 {
					delete(h.clients, "all")
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastProgress sends a progress message to all clients of a specific job
func (h *Hub) BroadcastProgress(jobID, msgType, status, currentFile, speed, message string, progress float64) {
	progressMsg := ProgressMessage{
		JobID:       jobID,
		Type:        msgType,
		Progress:    progress,
		Status:      status,
		CurrentFile: currentFile,
		Speed:       speed,
		Message:     message,
		Timestamp:   time.Now(),
	}

	select {
	case h.broadcast <- progressMsg:
	default:
		log.Printf("WebSocket broadcast channel full, dropping message for job %s", jobID)
	}
}

// readPump handles reading from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump handles writing to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
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

		// Broadcast progress update via WebSocket
		if hub != nil && total > 0 {
			progressPercent := float64(progress) / float64(total) * 100
			currentFile := ""
			if progress < total {
				currentFile = fmt.Sprintf("Track %d of %d", progress+1, total)
			}

			hub.BroadcastProgress(id, "progress", string(job.Status), currentFile, "",
				fmt.Sprintf("Downloaded %d of %d tracks", progress, total), progressPercent)
		}
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

		// Broadcast status update via WebSocket
		if hub != nil {
			msgType := "status"
			message := string(status)
			progress := float64(job.Progress) / float64(job.Total) * 100

			if status == JobStatusCompleted {
				msgType = "complete"
				progress = 100.0
				message = fmt.Sprintf("%s download completed", job.Title)
			} else if status == JobStatusFailed {
				msgType = "error"
				message = errorMsg
			} else if status == JobStatusProcessing {
				message = fmt.Sprintf("Started downloading %s", job.Title)
			}

			hub.BroadcastProgress(id, msgType, string(status), "", "", message, progress)
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

// scanAudioFiles recursively scans a directory for audio files (FLAC priority, MP3 fallback)
func scanAudioFiles(rootPath string) ([]AudioFile, error) {
	var allFiles []AudioFile

	// First pass: collect all audio files
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue walking, don't fail entire scan
		}

		// Check if it's an audio file (FLAC or MP3)
		ext := strings.ToLower(filepath.Ext(path))
		if !info.IsDir() && (ext == ".flac" || ext == ".mp3") {
			// Get relative path from root
			relativePath, err := filepath.Rel(rootPath, path)
			if err != nil {
				relativePath = path // fallback to absolute path
			}

			// Extract metadata from the audio file
			metadata := extractAudioMetadata(path)

			// Determine format
			format := "flac"
			if ext == ".mp3" {
				format = "mp3"
			}

			audioFile := AudioFile{
				Filename: info.Name(),
				Path:     relativePath,
				Size:     info.Size(),
				Format:   format,
				Metadata: metadata,
			}
			allFiles = append(allFiles, audioFile)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: apply FLAC prioritization
	return applyFlacPrioritization(allFiles), nil
}

// applyFlacPrioritization prioritizes FLAC files over MP3 files for the same track
func applyFlacPrioritization(files []AudioFile) []AudioFile {
	// Group files by their base name (without extension)
	fileGroups := make(map[string][]AudioFile)

	for _, file := range files {
		// Create a key based on the file path without extension
		basePath := strings.TrimSuffix(file.Path, filepath.Ext(file.Path))
		fileGroups[basePath] = append(fileGroups[basePath], file)
	}

	var result []AudioFile

	// For each group, prefer FLAC over MP3
	for _, group := range fileGroups {
		var selectedFile *AudioFile

		// Look for FLAC first
		for _, file := range group {
			if file.Format == "flac" {
				selectedFile = &file
				break
			}
		}

		// If no FLAC found, use MP3
		if selectedFile == nil {
			for _, file := range group {
				if file.Format == "mp3" {
					selectedFile = &file
					break
				}
			}
		}

		// Add the selected file to result
		if selectedFile != nil {
			result = append(result, *selectedFile)
		}
	}

	return result
}

// getContentType returns the appropriate MIME type for an audio file
func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".flac":
		return "audio/flac"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

// extractAudioMetadata extracts metadata from an audio file with fallback logic
func extractAudioMetadata(filePath string) *AudioMetadata {
	metadata := &AudioMetadata{}

	// Try to open and parse the audio file
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Warning: Could not open audio file %s: %v", filePath, err)
		// Use filename fallback
		return extractMetadataFromPath(filePath)
	}
	defer file.Close()

	// Extract metadata using dhowden/tag library (supports FLAC, MP3, etc.)
	meta, err := tag.ReadFrom(file)
	if err != nil {
		log.Printf("Warning: Could not parse audio metadata from %s: %v", filePath, err)
		// Use filename fallback
		return extractMetadataFromPath(filePath)
	}

	// Extract basic metadata
	metadata.Title = meta.Title()
	metadata.Artist = meta.Artist()
	metadata.Album = meta.Album()

	// Extract track number
	track, _ := meta.Track()
	metadata.TrackNumber = track

	// Note: Duration is not available through dhowden/tag library
	// We could implement duration extraction using a different library if needed

	// Use filename fallback for missing fields
	if metadata.Title == "" || metadata.Artist == "" || metadata.Album == "" {
		fallback := extractMetadataFromPath(filePath)
		if metadata.Title == "" {
			metadata.Title = fallback.Title
		}
		if metadata.Artist == "" {
			metadata.Artist = fallback.Artist
		}
		if metadata.Album == "" {
			metadata.Album = fallback.Album
		}
	}

	return metadata
}

// extractMetadataFromPath extracts metadata from file path as fallback
func extractMetadataFromPath(filePath string) *AudioMetadata {
	metadata := &AudioMetadata{}

	// Parse path components: Artist/Album/Track.flac or Track.mp3
	parts := strings.Split(filepath.ToSlash(filePath), "/")
	filename := filepath.Base(filePath)

	// Extract artist from path (grandparent directory)
	if len(parts) >= 3 {
		metadata.Artist = parts[len(parts)-3]
	}

	// Extract album from path (parent directory)
	if len(parts) >= 2 {
		metadata.Album = parts[len(parts)-2]
	}

	// Extract title from filename, removing track number prefix and extension
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove common track number prefixes like "01 - ", "1. ", etc.
	re := regexp.MustCompile(`^(\d+)[\.\-\s]+(.+)`)
	if matches := re.FindStringSubmatch(title); len(matches) > 2 {
		title = matches[2]
		// Try to extract track number
		if trackNum, err := strconv.Atoi(matches[1]); err == nil {
			metadata.TrackNumber = trackNum
		}
	}

	metadata.Title = title

	return metadata
}

// validateFilePath checks for path traversal attempts and other security issues
func validateFilePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Check for absolute paths
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("absolute paths not allowed")
	}

	// Check for empty path
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("empty path not allowed")
	}

	return nil
}

// handleRangeRequest handles HTTP range requests for efficient seeking
func handleRangeRequest(c *gin.Context, file *os.File, fileSize int64, rangeHeader string, filePath string) {
	// Parse range header (e.g., "bytes=0-1023" or "bytes=1024-")
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	ranges := strings.Split(rangeSpec, "-")

	if len(ranges) != 2 {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	var start, end int64
	var err error

	// Parse start position
	if ranges[0] != "" {
		start, err = strconv.ParseInt(ranges[0], 10, 64)
		if err != nil || start < 0 {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	}

	// Parse end position
	if ranges[1] != "" {
		end, err = strconv.ParseInt(ranges[1], 10, 64)
		if err != nil || end < start {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	} else {
		end = fileSize - 1
	}

	// Validate range bounds
	if start >= fileSize {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	if end >= fileSize {
		end = fileSize - 1
	}

	contentLength := end - start + 1

	// Seek to start position
	_, err = file.Seek(start, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to seek file",
		})
		return
	}

	// Set partial content headers
	c.Header("Content-Type", getContentType(filePath))
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Status(http.StatusPartialContent)

	// Copy only the requested range
	_, err = io.CopyN(c.Writer, file, contentLength)
	if err != nil {
		log.Printf("Error streaming range %d-%d: %v", start, end, err)
	}
}

// Global job queue instance
var jobQueue *JobQueue

// Global WebSocket hub instance
var hub *Hub

// handleWebSocketConnection handles WebSocket connections for specific job progress
func handleWebSocketConnection(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job ID is required"})
		return
	}

	// Verify job exists
	if _, exists := jobQueue.GetJob(jobID); !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create new client
	client := &Client{
		hub:   hub,
		conn:  conn,
		send:  make(chan ProgressMessage, 256),
		jobID: jobID,
	}

	// Register client and start pumps
	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// handleWebSocketAllConnection handles WebSocket connections for all downloads
func handleWebSocketAllConnection(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create new client with special "all" job ID
	client := &Client{
		hub:   hub,
		conn:  conn,
		send:  make(chan ProgressMessage, 256),
		jobID: "all",
	}

	// Register client and start pumps
	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// startWebServer initializes and starts the HTTP server
func startWebServer(port int) {
	// Initialize job queue with max 2 concurrent downloads
	jobQueue = NewJobQueue(2)
	jobQueue.Start()

	// Initialize WebSocket hub
	hub = NewHub()
	go hub.Run()

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

		// WebSocket endpoints for real-time progress
		wsGroup := apiGroup.Group("/ws")
		{
			// WebSocket endpoint for specific job progress
			wsGroup.GET("/downloads/:jobId", handleWebSocketConnection)

			// WebSocket endpoint for all downloads progress
			wsGroup.GET("/downloads", handleWebSocketAllConnection)
		}

		// File discovery endpoint
		apiGroup.GET("/files", func(c *gin.Context) {
			downloadLocation := config.GetDownloadLocation()

			// Scan for audio files
			audioFiles, err := scanAudioFiles(downloadLocation)
			if err != nil {
				log.Printf("Error scanning audio files: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to scan files",
					"details": err.Error(),
				})
				return
			}

			// Return the file list
			c.JSON(http.StatusOK, gin.H{
				"files": audioFiles,
				"count": len(audioFiles),
			})
		})

		// File streaming endpoint
		apiGroup.GET("/files/stream/*filepath", func(c *gin.Context) {
			requestedPath := c.Param("filepath")

			// Remove leading slash from filepath param
			if strings.HasPrefix(requestedPath, "/") {
				requestedPath = requestedPath[1:]
			}

			// Security: Validate file path
			if err := validateFilePath(requestedPath); err != nil {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "path security violation",
					"details": err.Error(),
				})
				return
			}

			// Only allow audio files (FLAC and MP3)
			ext := strings.ToLower(filepath.Ext(requestedPath))
			if ext != ".flac" && ext != ".mp3" {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "file extension not allowed",
					"details": "only .flac and .mp3 files can be streamed",
				})
				return
			}

			downloadLocation := config.GetDownloadLocation()
			fullPath := filepath.Join(downloadLocation, requestedPath)

			// Security: Ensure resolved path is within download location
			absDownloadPath, err := filepath.Abs(downloadLocation)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "server configuration error",
				})
				return
			}

			absRequestPath, err := filepath.Abs(fullPath)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid file path",
				})
				return
			}

			if !strings.HasPrefix(absRequestPath, absDownloadPath) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "path traversal not allowed",
				})
				return
			}

			// Check if file exists and is readable
			fileInfo, err := os.Stat(fullPath)
			if err != nil {
				if os.IsNotExist(err) {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "file not found",
						"path": requestedPath,
					})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "file access error",
					"details": err.Error(),
				})
				return
			}

			// Ensure it's a file, not a directory
			if fileInfo.IsDir() {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "path is a directory, not a file",
				})
				return
			}

			// Open the file
			file, err := os.Open(fullPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to open file",
					"details": err.Error(),
				})
				return
			}
			defer file.Close()

			// Set appropriate headers for audio streaming
			c.Header("Content-Type", getContentType(requestedPath))
			c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
			c.Header("Accept-Ranges", "bytes")
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("Access-Control-Allow-Origin", "*")

			// Handle range requests for seeking
			rangeHeader := c.GetHeader("Range")
			if rangeHeader != "" {
				handleRangeRequest(c, file, fileInfo.Size(), rangeHeader, requestedPath)
				return
			}

			// Stream the entire file
			c.Status(http.StatusOK)
			_, err = io.Copy(c.Writer, file)
			if err != nil {
				log.Printf("Error streaming file %s: %v", requestedPath, err)
			}
		})
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
