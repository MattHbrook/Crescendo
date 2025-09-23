package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketConnection tests basic WebSocket connectivity
func TestWebSocketConnection(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// First, create a job to monitor
	var downloadResponse struct {
		Job *DownloadJob `json:"job"`
	}

	resp := helper.PostJSON(t, "/api/downloads/album/test-album-1", nil, &downloadResponse)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, downloadResponse.Job)

	jobID := downloadResponse.Job.ID

	// Connect to WebSocket for this job
	wsPath := fmt.Sprintf("/api/ws/downloads/%s", jobID)
	conn := helper.ConnectWebSocket(t, wsPath)
	defer conn.Close()

	// Set read deadline to avoid hanging
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read initial message
	messageType, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, messageType)

	// Parse the progress message
	var progressMsg struct {
		JobID      string  `json:"jobId"`
		Status     string  `json:"status"`
		Progress   int     `json:"progress"`
		Total      int     `json:"total"`
		Percentage float64 `json:"percentage"`
		Message    string  `json:"message"`
	}

	err = json.Unmarshal(message, &progressMsg)
	require.NoError(t, err)

	// Verify message structure
	assert.Equal(t, jobID, progressMsg.JobID)
	assert.NotEmpty(t, progressMsg.Status)
	assert.GreaterOrEqual(t, progressMsg.Percentage, 0.0)
	assert.LessOrEqual(t, progressMsg.Percentage, 100.0)
}

// TestWebSocketGlobalConnection tests the global WebSocket endpoint
func TestWebSocketGlobalConnection(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Connect to global WebSocket endpoint
	conn := helper.ConnectWebSocket(t, "/api/ws/downloads")
	defer conn.Close()

	// Start multiple downloads to generate messages
	numJobs := 2
	for i := 0; i < numJobs; i++ {
		albumID := fmt.Sprintf("test-album-%d", i+1)
		helper.PostJSON(t, "/api/downloads/album/"+albumID, nil, nil)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read messages for a short period
	messagesReceived := 0
	timeout := time.After(3 * time.Second)

	for messagesReceived < numJobs && messagesReceived < 10 { // Safety limit
		select {
		case <-timeout:
			// Timeout reached, break out
			goto done
		default:
			// Try to read a message with a short timeout
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			messageType, message, err := conn.ReadMessage()

			if err != nil {
				// Timeout or other error, continue
				continue
			}

			assert.Equal(t, websocket.TextMessage, messageType)

			// Parse the progress message
			var progressMsg map[string]interface{}
			err = json.Unmarshal(message, &progressMsg)
			if err == nil {
				// Valid progress message
				assert.Contains(t, progressMsg, "jobId")
				assert.Contains(t, progressMsg, "status")
				messagesReceived++
			}
		}
	}

done:
	// We should have received at least some messages
	assert.Greater(t, messagesReceived, 0, "Should have received at least one WebSocket message")
}

// TestWebSocketInvalidJob tests WebSocket connection to non-existent job
func TestWebSocketInvalidJob(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Try to connect to WebSocket for non-existent job
	wsPath := "/api/ws/downloads/non-existent-job"

	// This should either fail to connect or close immediately
	conn, _, err := websocket.DefaultDialer.Dial("ws"+helper.Server.URL[4:]+wsPath, nil)

	if err != nil {
		// Connection failed - this is acceptable behavior
		return
	}

	defer conn.Close()

	// If connection succeeded, it should close quickly or send an error message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	messageType, message, err := conn.ReadMessage()
	if err != nil {
		// Connection closed or error - acceptable
		return
	}

	// If we get a message, it should be an error message
	assert.Equal(t, websocket.TextMessage, messageType)

	var response map[string]interface{}
	err = json.Unmarshal(message, &response)
	if err == nil {
		// Should contain error information
		assert.Contains(t, response, "error")
	}
}

// TestWebSocketConcurrentConnections tests multiple concurrent WebSocket connections
func TestWebSocketConcurrentConnections(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Create multiple jobs
	numJobs := 3
	jobIDs := make([]string, numJobs)

	for i := 0; i < numJobs; i++ {
		var response struct {
			Job *DownloadJob `json:"job"`
		}

		albumID := fmt.Sprintf("test-album-%d", i+1)
		resp := helper.PostJSON(t, "/api/downloads/album/"+albumID, nil, &response)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		jobIDs[i] = response.Job.ID
	}

	// Connect to each job's WebSocket
	connections := make([]*websocket.Conn, numJobs)
	for i, jobID := range jobIDs {
		wsPath := fmt.Sprintf("/api/ws/downloads/%s", jobID)
		conn := helper.ConnectWebSocket(t, wsPath)
		connections[i] = conn
	}

	// Cleanup connections
	defer func() {
		for _, conn := range connections {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	// Read at least one message from each connection
	for i, conn := range connections {
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		messageType, message, err := conn.ReadMessage()

		require.NoError(t, err, "Connection %d should receive a message", i)
		assert.Equal(t, websocket.TextMessage, messageType)

		var progressMsg struct {
			JobID string `json:"jobId"`
		}

		err = json.Unmarshal(message, &progressMsg)
		require.NoError(t, err)
		assert.Equal(t, jobIDs[i], progressMsg.JobID)
	}
}

// TestWebSocketMessageFormat tests the format of WebSocket progress messages
func TestWebSocketMessageFormat(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Create a download job
	var downloadResponse struct {
		Job *DownloadJob `json:"job"`
	}

	resp := helper.PostJSON(t, "/api/downloads/album/test-album-1", nil, &downloadResponse)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	jobID := downloadResponse.Job.ID

	// Connect to WebSocket
	wsPath := fmt.Sprintf("/api/ws/downloads/%s", jobID)
	conn := helper.ConnectWebSocket(t, wsPath)
	defer conn.Close()

	// Read a message
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, messageType)

	// Parse and validate message structure
	var progressMsg map[string]interface{}
	err = json.Unmarshal(message, &progressMsg)
	require.NoError(t, err)

	// Check required fields
	requiredFields := []string{"jobId", "status", "progress", "total", "percentage"}
	for _, field := range requiredFields {
		assert.Contains(t, progressMsg, field, "Message should contain field: %s", field)
	}

	// Check field types and values
	assert.IsType(t, "", progressMsg["jobId"])
	assert.IsType(t, "", progressMsg["status"])
	assert.IsType(t, float64(0), progressMsg["progress"])
	assert.IsType(t, float64(0), progressMsg["total"])
	assert.IsType(t, float64(0), progressMsg["percentage"])

	// Validate ranges
	percentage := progressMsg["percentage"].(float64)
	assert.GreaterOrEqual(t, percentage, 0.0)
	assert.LessOrEqual(t, percentage, 100.0)

	progress := progressMsg["progress"].(float64)
	total := progressMsg["total"].(float64)
	assert.GreaterOrEqual(t, progress, 0.0)
	assert.GreaterOrEqual(t, total, 0.0)

	if total > 0 {
		expectedPercentage := (progress / total) * 100
		assert.InDelta(t, expectedPercentage, percentage, 0.1)
	}
}

// TestWebSocketConnectionCleanup tests that WebSocket connections are properly cleaned up
func TestWebSocketConnectionCleanup(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup(t)

	// Create a job
	var downloadResponse struct {
		Job *DownloadJob `json:"job"`
	}

	resp := helper.PostJSON(t, "/api/downloads/album/test-album-1", nil, &downloadResponse)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	jobID := downloadResponse.Job.ID

	// Connect and read initial message
	wsPath := fmt.Sprintf("/api/ws/downloads/%s", jobID)
	conn := helper.ConnectWebSocket(t, wsPath)

	// Read the initial message (the mock WebSocket sends one message)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	require.NoError(t, err, "Should be able to read initial message")

	// Close connection
	err = conn.Close()
	require.NoError(t, err)

	// Wait a moment for cleanup
	time.Sleep(100 * time.Millisecond)

	// Try to read from closed connection (should fail)
	_, _, err = conn.ReadMessage()
	assert.Error(t, err, "Reading from closed connection should fail")
}