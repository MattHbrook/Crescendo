package types

// AudioFile represents a discovered audio file (FLAC, MP3, etc.)
type AudioFile struct {
	Filename string         `json:"filename"`
	Path     string         `json:"path"`
	Size     int64          `json:"size"`
	Format   string         `json:"format"`         // "flac", "mp3", etc.
	Metadata *AudioMetadata `json:"metadata,omitempty"`
}

// AudioMetadata represents metadata for an audio file
type AudioMetadata struct {
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	Duration    string `json:"duration,omitempty"`
	TrackNumber int    `json:"trackNumber,omitempty"`
}