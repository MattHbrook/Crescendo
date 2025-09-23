package handlers

import (
	"crescendo/api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SearchHandler handles search endpoints
type SearchHandler struct{}

// NewSearchHandler creates a new search handler
func NewSearchHandler() *SearchHandler {
	return &SearchHandler{}
}

// Search performs a search for tracks or albums
func (h *SearchHandler) Search(c *gin.Context) {
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
			"error":   "search failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"type":    searchType,
		"results": results,
	})
}