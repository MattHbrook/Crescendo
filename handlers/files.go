package handlers

import (
	"crescendo/config"
	"crescendo/services"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// FileHandler handles file management endpoints
type FileHandler struct {
	fileService services.FileService
}

// NewFileHandler creates a new file handler
func NewFileHandler(fs services.FileService) *FileHandler {
	return &FileHandler{
		fileService: fs,
	}
}

// ListFiles returns a list of all discovered audio files
func (h *FileHandler) ListFiles(c *gin.Context) {
	downloadLocation := config.GetDownloadLocation()

	// Scan for audio files
	audioFiles, err := h.fileService.ScanAudioFiles(downloadLocation)
	if err != nil {
		log.Printf("Error scanning audio files: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to scan files",
			"details": err.Error(),
		})
		return
	}

	// Return the file list
	c.JSON(http.StatusOK, gin.H{
		"files": audioFiles,
		"count": len(audioFiles),
	})
}

// StreamFile streams an audio file with support for range requests
func (h *FileHandler) StreamFile(c *gin.Context) {
	requestedPath := c.Param("filepath")

	// Remove leading slash from filepath param
	if strings.HasPrefix(requestedPath, "/") {
		requestedPath = requestedPath[1:]
	}

	// Security: Validate file path
	if err := h.fileService.ValidateFilePath(requestedPath); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "path security violation",
			"details": err.Error(),
		})
		return
	}

	// Only allow audio files (FLAC and MP3)
	ext := strings.ToLower(filepath.Ext(requestedPath))
	if ext != ".flac" && ext != ".mp3" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "file extension not allowed",
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
				"path":  requestedPath,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "file access error",
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
			"error":   "failed to open file",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	// Set appropriate headers for audio streaming
	c.Header("Content-Type", h.fileService.GetContentType(requestedPath))
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Access-Control-Allow-Origin", "*")

	// Handle range requests for seeking
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		h.handleRangeRequest(c, file, fileInfo.Size(), rangeHeader, requestedPath)
		return
	}

	// Stream the entire file
	c.Status(http.StatusOK)
	_, err = io.Copy(c.Writer, file)
	if err != nil {
		log.Printf("Error streaming file %s: %v", requestedPath, err)
	}
}

// handleRangeRequest handles HTTP range requests for efficient seeking
func (h *FileHandler) handleRangeRequest(c *gin.Context, file *os.File, fileSize int64, rangeHeader string, filePath string) {
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
	c.Header("Content-Type", h.fileService.GetContentType(filePath))
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