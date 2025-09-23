package types

import "time"

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