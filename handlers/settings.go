package handlers

import (
	"crescendo/config"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SettingsHandler handles settings-related endpoints
type SettingsHandler struct{}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

// Settings represents the user settings
type Settings struct {
	DownloadLocation string `json:"downloadLocation"`
}

// getSettingsFilePath returns the path to the settings file
func getSettingsFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".crescendo-settings.json")
}

// loadSettings loads settings from the settings file
func loadSettings() (*Settings, error) {
	settingsPath := getSettingsFilePath()

	// If file doesn't exist, return default settings
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return &Settings{
			DownloadLocation: config.GetDownloadLocation(),
		}, nil
	}

	// Read and parse the settings file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// saveSettings saves settings to the settings file
func saveSettings(settings *Settings) error {
	settingsPath := getSettingsFilePath()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

// validatePath validates that the path exists and is writable
func validatePath(path string) error {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	} else if !info.IsDir() {
		return gin.Error{Err: err, Type: gin.ErrorTypePublic, Meta: "Path is not a directory"}
	}

	// Test write permissions by creating a temporary file
	testFile := filepath.Join(path, ".crescendo-write-test")
	file, err := os.Create(testFile)
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(testFile)

	return nil
}

// GetSettings returns the current settings
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	settings, err := loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load settings",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates the user settings
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var newSettings Settings
	if err := c.ShouldBindJSON(&newSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid settings format",
			"details": err.Error(),
		})
		return
	}

	// Validate the download location path
	if err := validatePath(newSettings.DownloadLocation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid download location",
			"details": err.Error(),
		})
		return
	}

	// Save the settings
	if err := saveSettings(&newSettings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save settings",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Settings updated successfully",
		"settings": newSettings,
	})
}