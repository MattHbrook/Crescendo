package services

import (
	"crescendo/types"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
)

// FileService interface defines methods for file management
type FileService interface {
	ScanAudioFiles(rootPath string) ([]types.AudioFile, error)
	ExtractAudioMetadata(filePath string) *types.AudioMetadata
	ValidateFilePath(path string) error
	GetContentType(filePath string) string
}

// fileService implements the FileService interface
type fileService struct{}

// NewFileService creates a new file service
func NewFileService() FileService {
	return &fileService{}
}

// ScanAudioFiles recursively scans a directory for audio files (FLAC priority, MP3 fallback)
func (fs *fileService) ScanAudioFiles(rootPath string) ([]types.AudioFile, error) {
	var allFiles []types.AudioFile

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
			metadata := fs.ExtractAudioMetadata(path)

			// Determine format
			format := "flac"
			if ext == ".mp3" {
				format = "mp3"
			}

			audioFile := types.AudioFile{
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
	return fs.applyFlacPrioritization(allFiles), nil
}

// applyFlacPrioritization prioritizes FLAC files over MP3 files for the same track
func (fs *fileService) applyFlacPrioritization(files []types.AudioFile) []types.AudioFile {
	// Group files by their base name (without extension)
	fileGroups := make(map[string][]types.AudioFile)

	for _, file := range files {
		// Create a key based on the file path without extension
		basePath := strings.TrimSuffix(file.Path, filepath.Ext(file.Path))
		fileGroups[basePath] = append(fileGroups[basePath], file)
	}

	var result []types.AudioFile

	// For each group, prefer FLAC over MP3
	for _, group := range fileGroups {
		var selectedFile *types.AudioFile

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

// GetContentType returns the appropriate MIME type for an audio file
func (fs *fileService) GetContentType(filePath string) string {
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

// ExtractAudioMetadata extracts metadata from an audio file with fallback logic
func (fs *fileService) ExtractAudioMetadata(filePath string) *types.AudioMetadata {
	metadata := &types.AudioMetadata{}

	// Try to open and parse the audio file
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Warning: Could not open audio file %s: %v", filePath, err)
		// Use filename fallback
		return fs.extractMetadataFromPath(filePath)
	}
	defer file.Close()

	// Extract metadata using dhowden/tag library (supports FLAC, MP3, etc.)
	meta, err := tag.ReadFrom(file)
	if err != nil {
		log.Printf("Warning: Could not parse audio metadata from %s: %v", filePath, err)
		// Use filename fallback
		return fs.extractMetadataFromPath(filePath)
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
		fallback := fs.extractMetadataFromPath(filePath)
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
func (fs *fileService) extractMetadataFromPath(filePath string) *types.AudioMetadata {
	metadata := &types.AudioMetadata{}

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

// ValidateFilePath checks for path traversal attempts and other security issues
func (fs *fileService) ValidateFilePath(path string) error {
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