package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoint tests the basic health check endpoint
func TestHealthEndpoint(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	var response map[string]interface{}
	resp := helper.GetJSON(t, "/health", &response)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "crescendo-test", response["service"])
}

// TestSearchEndpoint tests the search functionality
func TestSearchEndpoint(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	tests := []struct {
		name           string
		query          string
		searchType     string
		expectedStatus int
		expectResults  bool
	}{
		{
			name:           "valid album search",
			query:          "test",
			searchType:     "album",
			expectedStatus: http.StatusOK,
			expectResults:  true,
		},
		{
			name:           "valid track search",
			query:          "test",
			searchType:     "track",
			expectedStatus: http.StatusOK,
			expectResults:  true,
		},
		{
			name:           "empty query",
			query:          "",
			searchType:     "album",
			expectedStatus: http.StatusBadRequest,
			expectResults:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/search?q=%s&type=%s", tt.query, tt.searchType)
			var response map[string]interface{}
			resp := helper.GetJSON(t, path, &response)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectResults {
				assert.Contains(t, response, "results")
				assert.Equal(t, tt.query, response["query"])
				assert.Equal(t, tt.searchType, response["type"])
			} else {
				assert.Contains(t, response, "error")
			}
		})
	}
}

// TestDownloadWorkflow tests the complete download workflow
func TestDownloadWorkflow(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Step 1: Queue an album download
	var downloadResponse struct {
		Message string       `json:"message"`
		Job     *DownloadJob `json:"job"`
	}

	resp := helper.PostJSON(t, "/api/downloads/album/test-album-1", nil, &downloadResponse)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, downloadResponse.Job)
	require.NotEmpty(t, downloadResponse.Job.ID)

	jobID := downloadResponse.Job.ID

	// Verify initial job state
	assert.Equal(t, JobTypeAlbum, downloadResponse.Job.Type)
	assert.Equal(t, "test-album-1", downloadResponse.Job.ItemID)
	assert.Equal(t, JobStatusProcessing, downloadResponse.Job.Status)

	// Step 2: Check job status
	var statusResponse struct {
		Job *DownloadJob `json:"job"`
	}

	resp = helper.GetJSON(t, "/api/downloads/"+jobID, &statusResponse)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, statusResponse.Job)
	assert.Equal(t, jobID, statusResponse.Job.ID)

	// Step 3: Wait for job completion (mock jobs complete immediately)
	completedJob := helper.WaitForJobCompletion(t, jobID, 5*time.Second)
	require.NotNil(t, completedJob)

	// For mock jobs, we expect them to complete quickly
	// In real tests with actual downloads, you'd check for proper completion
}

// TestFileListingEndpoint tests the file listing functionality
func TestFileListingEndpoint(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	var response struct {
		Files []AudioFile `json:"files"`
		Count int         `json:"count"`
	}

	resp := helper.GetJSON(t, "/api/files", &response)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Should have at least the test files we created
	assert.GreaterOrEqual(t, response.Count, 2)
	assert.GreaterOrEqual(t, len(response.Files), 2)

	// Check that files have required fields
	for _, file := range response.Files {
		assert.NotEmpty(t, file.Filename)
		assert.NotEmpty(t, file.Path)
		assert.NotEmpty(t, file.Format)
		assert.Greater(t, file.Size, int64(0))

		// Check that metadata is present (may be from path fallback)
		if file.Metadata != nil {
			assert.NotEmpty(t, file.Metadata.Title)
		}
	}
}

// TestFileFormatPrioritization tests FLAC vs MP3 prioritization
func TestFileFormatPrioritization(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Create both FLAC and MP3 versions of the same track
	flacContent := createMinimalFLACFile()
	mp3Content := createMinimalMP3File()

	helper.CreateTestFile(t, "Test Artist/Priority Album/Same Song.flac", flacContent)
	helper.CreateTestFile(t, "Test Artist/Priority Album/Same Song.mp3", mp3Content)

	var response struct {
		Files []AudioFile `json:"files"`
		Count int         `json:"count"`
	}

	resp := helper.GetJSON(t, "/api/files", &response)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Find files from our priority test
	var priorityFiles []AudioFile
	for _, file := range response.Files {
		if file.Path == "Test Artist/Priority Album/Same Song.flac" ||
			file.Path == "Test Artist/Priority Album/Same Song.mp3" {
			priorityFiles = append(priorityFiles, file)
		}
	}

	// Should only have one file (FLAC should be prioritized)
	require.Equal(t, 1, len(priorityFiles))
	assert.Equal(t, "flac", priorityFiles[0].Format)
	assert.Equal(t, "Test Artist/Priority Album/Same Song.flac", priorityFiles[0].Path)
}

// TestMetadataExtraction tests metadata extraction functionality
func TestMetadataExtraction(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Create a file with known path structure for fallback testing
	testContent := createMinimalFLACFile()
	helper.CreateTestFile(t, "Known Artist/Known Album/01 - Known Song.flac", testContent)

	var response struct {
		Files []AudioFile `json:"files"`
		Count int         `json:"count"`
	}

	resp := helper.GetJSON(t, "/api/files", &response)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Find our test file
	var testFile *AudioFile
	for _, file := range response.Files {
		if file.Path == "Known Artist/Known Album/01 - Known Song.flac" {
			testFile = &file
			break
		}
	}

	require.NotNil(t, testFile)
	require.NotNil(t, testFile.Metadata)

	// Check fallback metadata extraction from path
	assert.Equal(t, "Known Song", testFile.Metadata.Title)
	assert.Equal(t, "Known Artist", testFile.Metadata.Artist)
	assert.Equal(t, "Known Album", testFile.Metadata.Album)
}

// TestJobNotFound tests the behavior when requesting a non-existent job
func TestJobNotFound(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	var response map[string]interface{}
	resp := helper.GetJSON(t, "/api/downloads/non-existent-job", &response)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, response, "error")
	assert.Equal(t, "job not found", response["error"])
}

// TestConcurrentDownloads tests multiple simultaneous downloads
func TestConcurrentDownloads(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Queue multiple downloads concurrently
	numDownloads := 3
	jobIDs := make([]string, numDownloads)

	for i := 0; i < numDownloads; i++ {
		var response struct {
			Job *DownloadJob `json:"job"`
		}

		albumID := fmt.Sprintf("test-album-%d", i+1)
		resp := helper.PostJSON(t, "/api/downloads/album/"+albumID, nil, &response)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.NotNil(t, response.Job)

		jobIDs[i] = response.Job.ID
	}

	// Verify all jobs were created with unique IDs
	uniqueIDs := make(map[string]bool)
	for _, id := range jobIDs {
		assert.False(t, uniqueIDs[id], "Job ID should be unique: %s", id)
		uniqueIDs[id] = true
	}

	// Wait for all jobs to complete (allowing more time for concurrent processing)
	for _, jobID := range jobIDs {
		job := helper.WaitForJobCompletion(t, jobID, 10*time.Second)
		require.NotNil(t, job)
	}
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid endpoint",
			path:           "/api/invalid",
			expectedStatus: http.StatusNotFound,
			expectedError:  "",
		},
		{
			name:           "missing album ID",
			path:           "/api/downloads/album/",
			expectedStatus: http.StatusNotFound,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := helper.MakeRequest(t, "GET", tt.path, nil)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestJobQueue tests job queue functionality
func TestJobQueue(t *testing.T) {
	// Test job queue operations directly
	jobQueue := NewJobQueue(1)
	jobQueue.Start()

	// Add a job
	job := jobQueue.AddJob(JobTypeAlbum, "test-album", "Test Album", "Test Artist")
	require.NotNil(t, job)
	assert.NotEmpty(t, job.ID)
	assert.Equal(t, JobTypeAlbum, job.Type)
	assert.Equal(t, "test-album", job.ItemID)

	// Get the job
	retrievedJob, exists := jobQueue.GetJob(job.ID)
	require.True(t, exists)
	require.NotNil(t, retrievedJob)
	assert.Equal(t, job.ID, retrievedJob.ID)

	// Get all jobs
	allJobs := jobQueue.GetAllJobs()
	assert.GreaterOrEqual(t, len(allJobs), 1)

	// Check for our job
	found := false
	for _, j := range allJobs {
		if j.ID == job.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Job should be found in all jobs list")
}