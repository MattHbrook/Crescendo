package types

import "time"

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
	ID          string     `json:"id"`
	Type        JobType    `json:"type"`
	Status      JobStatus  `json:"status"`
	ItemID      string     `json:"itemId"`
	Title       string     `json:"title"`
	Artist      string     `json:"artist"`
	Progress    int        `json:"progress"`
	Total       int        `json:"total"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}