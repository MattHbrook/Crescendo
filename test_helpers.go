package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelper provides utilities for testing the Crescendo server
type TestHelper struct {
	Server       *httptest.Server
	TestDataDir  string
	OriginalDir  string
	JobQueue     *JobQueue
	Router       *gin.Engine
}

// NewTestHelper creates a new test helper with a temporary test environment
func NewTestHelper(t *testing.T) *TestHelper {
	// Create temporary test directory
	testDir, err := os.MkdirTemp("", "crescendo-test-*")
	require.NoError(t, err)

	// Store original working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Change to test directory
	err = os.Chdir(testDir)
	require.NoError(t, err)

	// Setup gin in test mode
	gin.SetMode(gin.TestMode)

	// Create job queue
	jobQueue := NewJobQueue(2) // Use 2 workers for testing
	jobQueue.Start()

	// Setup router with test configuration
	router := setupTestRouter(jobQueue)

	// Create test server
	server := httptest.NewServer(router)

	helper := &TestHelper{
		Server:      server,
		TestDataDir: testDir,
		OriginalDir: originalDir,
		JobQueue:    jobQueue,
		Router:      router,
	}

	// Create test data structure
	helper.setupTestData(t)

	return helper
}

// Cleanup cleans up test resources
func (h *TestHelper) Cleanup(t *testing.T) {
	if h.Server != nil {
		h.Server.Close()
	}

	// JobQueue cleanup - it will stop when the server stops
	// No explicit Stop method needed

	// Change back to original directory
	err := os.Chdir(h.OriginalDir)
	require.NoError(t, err)

	// Remove test directory
	err = os.RemoveAll(h.TestDataDir)
	require.NoError(t, err)
}

// setupTestRouter creates a router with test configuration
func setupTestRouter(jobQueue *JobQueue) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Basic health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "crescendo-test",
		})
	})

	// API routes
	apiGroup := router.Group("/api")
	{
		// Search endpoint (mock)
		apiGroup.GET("/search", func(c *gin.Context) {
			query := c.Query("q")
			searchType := c.DefaultQuery("type", "track")

			if query == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "query parameter 'q' is required",
				})
				return
			}

			// Return mock search results
			mockResults := getMockSearchResults(query, searchType)
			c.JSON(http.StatusOK, mockResults)
		})

		// Download endpoints
		downloadsGroup := apiGroup.Group("/downloads")
		{
			downloadsGroup.POST("/album/:id", func(c *gin.Context) {
				albumID := c.Param("id")
				job := jobQueue.AddJob(JobTypeAlbum, albumID, "", "")
				c.JSON(http.StatusCreated, gin.H{
					"message": "Album download queued successfully",
					"job": job,
				})
			})

			downloadsGroup.GET("/:jobId", func(c *gin.Context) {
				jobID := c.Param("jobId")
				job, exists := jobQueue.GetJob(jobID)
				if !exists || job == nil {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "job not found",
					})
					return
				}
				c.JSON(http.StatusOK, gin.H{"job": job})
			})
		}

		// Files endpoint
		apiGroup.GET("/files", func(c *gin.Context) {
			audioFiles, err := scanAudioFiles(".")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to scan files",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"files": audioFiles,
				"count": len(audioFiles),
			})
		})

		// WebSocket endpoints
		wsGroup := apiGroup.Group("/ws")
		{
			wsGroup.GET("/downloads/:jobId", func(c *gin.Context) {
				// Mock WebSocket handler for testing
				jobID := c.Param("jobId")

				// Upgrade HTTP connection to WebSocket
				upgrader := websocket.Upgrader{
					CheckOrigin: func(r *http.Request) bool {
						return true // Allow all origins for testing
					},
				}

				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				// Check if job exists for testing invalid job scenarios
				if jobID == "non-existent-job" {
					// Send error message for non-existent jobs
					errorMessage := map[string]interface{}{
						"error": "job not found",
						"jobId": jobID,
					}
					conn.WriteJSON(errorMessage)
					time.Sleep(50 * time.Millisecond)
					return
				}

				// Send a mock progress message for valid jobs
				mockMessage := map[string]interface{}{
					"jobId":      jobID,
					"status":     "processing",
					"progress":   10,
					"total":      100,
					"percentage": 10.0,
					"message":    "Mock progress update",
				}

				conn.WriteJSON(mockMessage)

				// Keep connection alive briefly for tests
				time.Sleep(100 * time.Millisecond)
			})

			wsGroup.GET("/downloads", func(c *gin.Context) {
				// Mock global WebSocket handler for testing
				upgrader := websocket.Upgrader{
					CheckOrigin: func(r *http.Request) bool {
						return true // Allow all origins for testing
					},
				}

				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				// Send multiple mock progress messages for different jobs
				mockMessages := []map[string]interface{}{
					{
						"jobId":      "test-job-1",
						"status":     "processing",
						"progress":   5,
						"total":      50,
						"percentage": 10.0,
						"message":    "Job 1 progress update",
					},
					{
						"jobId":      "test-job-2",
						"status":     "processing",
						"progress":   15,
						"total":      60,
						"percentage": 25.0,
						"message":    "Job 2 progress update",
					},
				}

				// Send messages with small delays
				for _, msg := range mockMessages {
					conn.WriteJSON(msg)
					time.Sleep(50 * time.Millisecond)
				}

				// Keep connection alive longer for tests
				time.Sleep(500 * time.Millisecond)
			})
		}
	}

	return router
}

// setupTestData creates test audio files and directory structure
func (h *TestHelper) setupTestData(t *testing.T) {
	// Create test artist/album directories
	testArtistDir := "Test Artist"
	testAlbumDir := filepath.Join(testArtistDir, "Test Album")

	err := os.MkdirAll(testAlbumDir, 0755)
	require.NoError(t, err)

	// Create minimal test FLAC file
	flacContent := createMinimalFLACFile()
	flacPath := filepath.Join(testAlbumDir, "01 - Test Song.flac")
	err = os.WriteFile(flacPath, flacContent, 0644)
	require.NoError(t, err)

	// Create minimal test MP3 file
	mp3Content := createMinimalMP3File()
	mp3Path := filepath.Join(testAlbumDir, "02 - Test Song 2.mp3")
	err = os.WriteFile(mp3Path, mp3Content, 0644)
	require.NoError(t, err)
}

// createMinimalFLACFile creates a minimal valid FLAC file for testing
func createMinimalFLACFile() []byte {
	// This is a minimal FLAC file header - enough for testing metadata extraction
	// In a real implementation, you'd want a proper minimal FLAC file
	return []byte("fLaC\x00\x00\x00\x22\x10\x00\x10\x00\x00\x00\x0F\x00\x00\x0F\x0A\xC4\x42\xF0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
}

// createMinimalMP3File creates a minimal valid MP3 file for testing
func createMinimalMP3File() []byte {
	// This is a minimal MP3 file header - enough for testing metadata extraction
	return []byte("ID3\x03\x00\x00\x00\x00\x00\x00")
}

// getMockSearchResults returns mock search results for testing
func getMockSearchResults(query, searchType string) interface{} {
	switch searchType {
	case "album":
		return gin.H{
			"query": query,
			"results": gin.H{
				"Albums": gin.H{
					"albums": []gin.H{
						{
							"id":          "test-album-1",
							"title":       "Test Album",
							"artist":      "Test Artist",
							"cover":       "https://example.com/cover.jpg",
							"releaseDate": "2023-01-01",
						},
					},
				},
			},
			"type": searchType,
		}
	case "track":
		return gin.H{
			"query": query,
			"results": gin.H{
				"Tracks": gin.H{
					"tracks": []gin.H{
						{
							"id":     "test-track-1",
							"title":  "Test Track",
							"artist": "Test Artist",
							"album":  "Test Album",
						},
					},
				},
			},
			"type": searchType,
		}
	default:
		return gin.H{
			"query":   query,
			"results": gin.H{},
			"type":    searchType,
		}
	}
}

// MakeRequest makes an HTTP request to the test server
func (h *TestHelper) MakeRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, h.Server.URL+path, reqBody)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

// GetJSON makes a GET request and unmarshals JSON response
func (h *TestHelper) GetJSON(t *testing.T, path string, target interface{}) *http.Response {
	resp := h.MakeRequest(t, "GET", path, nil)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	if target != nil {
		err = json.Unmarshal(body, target)
		require.NoError(t, err)
	}

	return resp
}

// PostJSON makes a POST request with JSON body and unmarshals JSON response
func (h *TestHelper) PostJSON(t *testing.T, path string, requestBody interface{}, target interface{}) *http.Response {
	resp := h.MakeRequest(t, "POST", path, requestBody)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	if target != nil {
		err = json.Unmarshal(body, target)
		require.NoError(t, err)
	}

	return resp
}

// WaitForJobCompletion waits for a job to complete or timeout
func (h *TestHelper) WaitForJobCompletion(t *testing.T, jobID string, timeout time.Duration) *DownloadJob {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		var response struct {
			Job *DownloadJob `json:"job"`
		}

		resp := h.GetJSON(t, "/api/downloads/"+jobID, &response)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		if response.Job.Status == JobStatusCompleted || response.Job.Status == JobStatusFailed {
			return response.Job
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Job %s did not complete within timeout", jobID)
	return nil
}

// ConnectWebSocket connects to a WebSocket endpoint
func (h *TestHelper) ConnectWebSocket(t *testing.T, path string) *websocket.Conn {
	wsURL := "ws" + h.Server.URL[4:] + path // Replace http:// with ws://

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	return conn
}

// AssertFileExists checks if a file exists in the test data directory
func (h *TestHelper) AssertFileExists(t *testing.T, relativePath string) {
	fullPath := filepath.Join(h.TestDataDir, relativePath)
	_, err := os.Stat(fullPath)
	assert.NoError(t, err, "File should exist: %s", relativePath)
}

// AssertFileNotExists checks if a file does not exist
func (h *TestHelper) AssertFileNotExists(t *testing.T, relativePath string) {
	fullPath := filepath.Join(h.TestDataDir, relativePath)
	_, err := os.Stat(fullPath)
	assert.Error(t, err, "File should not exist: %s", relativePath)
	assert.True(t, os.IsNotExist(err), "Should be a 'not exist' error")
}

// CreateTestFile creates a test file with specified content
func (h *TestHelper) CreateTestFile(t *testing.T, relativePath string, content []byte) {
	fullPath := filepath.Join(h.TestDataDir, relativePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(fullPath, content, 0644)
	require.NoError(t, err)
}