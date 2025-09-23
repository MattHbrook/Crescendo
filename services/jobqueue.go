package services

import (
	"crescendo/api"
	"crescendo/types"
	"crescendo/websocket"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobQueue interface defines the methods for managing download jobs
type JobQueue interface {
	Start()
	AddJob(jobType types.JobType, itemID, title, artist string) *types.DownloadJob
	GetJob(id string) (*types.DownloadJob, bool)
	GetAllJobs() []*types.DownloadJob
	CancelJob(id string) bool
	UpdateJobProgress(id string, progress, total int)
	SetJobStatus(id string, status types.JobStatus, errorMsg string)
}

// jobQueue manages download jobs
type jobQueue struct {
	jobs        map[string]*types.DownloadJob
	queue       chan *types.DownloadJob
	activeJobs  map[string]*types.DownloadJob
	mu          sync.RWMutex
	maxWorkers  int
	workerCount int
	hub         websocket.Hub
}

// NewJobQueue creates a new job queue
func NewJobQueue(maxWorkers int, hub websocket.Hub) JobQueue {
	return &jobQueue{
		jobs:       make(map[string]*types.DownloadJob),
		queue:      make(chan *types.DownloadJob, 100), // Buffer for 100 jobs
		activeJobs: make(map[string]*types.DownloadJob),
		maxWorkers: maxWorkers,
		hub:        hub,
	}
}

// AddJob adds a new job to the queue
func (jq *jobQueue) AddJob(jobType types.JobType, itemID, title, artist string) *types.DownloadJob {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	job := &types.DownloadJob{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    types.JobStatusQueued,
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
func (jq *jobQueue) GetJob(id string) (*types.DownloadJob, bool) {
	jq.mu.RLock()
	defer jq.mu.RUnlock()
	job, exists := jq.jobs[id]
	return job, exists
}

// GetAllJobs returns all jobs
func (jq *jobQueue) GetAllJobs() []*types.DownloadJob {
	jq.mu.RLock()
	defer jq.mu.RUnlock()

	jobs := make([]*types.DownloadJob, 0, len(jq.jobs))
	for _, job := range jq.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// CancelJob cancels a queued job
func (jq *jobQueue) CancelJob(id string) bool {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	job, exists := jq.jobs[id]
	if !exists {
		return false
	}

	if job.Status == types.JobStatusQueued {
		job.Status = types.JobStatusCancelled
		now := time.Now()
		job.CompletedAt = &now
		return true
	}

	return false
}

// UpdateJobProgress updates job progress
func (jq *jobQueue) UpdateJobProgress(id string, progress, total int) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	if job, exists := jq.jobs[id]; exists {
		job.Progress = progress
		job.Total = total

		// Broadcast progress update via WebSocket
		if jq.hub != nil && total > 0 {
			progressPercent := float64(progress) / float64(total) * 100
			currentFile := ""
			if progress < total {
				currentFile = fmt.Sprintf("Track %d of %d", progress+1, total)
			}

			jq.hub.BroadcastProgress(id, "progress", string(job.Status), currentFile, "",
				fmt.Sprintf("Downloaded %d of %d tracks", progress, total), progressPercent)
		}
	}
}

// SetJobStatus updates job status
func (jq *jobQueue) SetJobStatus(id string, status types.JobStatus, errorMsg string) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	if job, exists := jq.jobs[id]; exists {
		job.Status = status
		if errorMsg != "" {
			job.Error = errorMsg
		}

		now := time.Now()
		if status == types.JobStatusProcessing && job.StartedAt == nil {
			job.StartedAt = &now
			jq.activeJobs[id] = job
		} else if status == types.JobStatusCompleted || status == types.JobStatusFailed || status == types.JobStatusCancelled {
			job.CompletedAt = &now
			delete(jq.activeJobs, id)
		}

		// Broadcast status update via WebSocket
		if jq.hub != nil {
			msgType := "status"
			message := string(status)
			progress := float64(job.Progress) / float64(job.Total) * 100

			if status == types.JobStatusCompleted {
				msgType = "complete"
				progress = 100.0
				message = fmt.Sprintf("%s download completed", job.Title)
			} else if status == types.JobStatusFailed {
				msgType = "error"
				message = errorMsg
			} else if status == types.JobStatusProcessing {
				message = fmt.Sprintf("Started downloading %s", job.Title)
			}

			jq.hub.BroadcastProgress(id, msgType, string(status), "", "", message, progress)
		}
	}
}

// Start begins processing jobs
func (jq *jobQueue) Start() {
	for i := 0; i < jq.maxWorkers; i++ {
		go jq.worker()
	}
}

// worker processes jobs from the queue
func (jq *jobQueue) worker() {
	for job := range jq.queue {
		if job.Status == types.JobStatusCancelled {
			continue
		}

		jq.SetJobStatus(job.ID, types.JobStatusProcessing, "")

		var err error
		switch job.Type {
		case types.JobTypeAlbum:
			err = jq.processAlbumJob(job)
		case types.JobTypeTrack:
			err = jq.processTrackJob(job)
		case types.JobTypeArtist:
			err = jq.processArtistJob(job)
		}

		if err != nil {
			jq.SetJobStatus(job.ID, types.JobStatusFailed, err.Error())
			log.Printf("Job %s failed: %v", job.ID, err)
		} else {
			jq.SetJobStatus(job.ID, types.JobStatusCompleted, "")
			log.Printf("Job %s completed successfully", job.ID)
		}
	}
}

// processAlbumJob processes an album download job
func (jq *jobQueue) processAlbumJob(job *types.DownloadJob) error {
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
func (jq *jobQueue) processTrackJob(job *types.DownloadJob) error {
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
func (jq *jobQueue) processArtistJob(job *types.DownloadJob) error {
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