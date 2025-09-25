package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var Env = map[string]string{
	"DAB_ENDPOINT":      os.Getenv("DAB_ENDPOINT"),
	"DOWNLOAD_LOCATION": os.Getenv("DOWNLOAD_LOCATION"),
}

func GetEndpoint() string {
	endpoint := Env["DAB_ENDPOINT"]
	if endpoint != "" {
		return endpoint
	}
	return "https://dabmusic.xyz"
}

func GetDownloadLocation() string {
	// First check environment variable for custom location
	if customPath := os.Getenv("CRESCENDO_DOWNLOADS"); customPath != "" {
		return customPath
	}

	// Use standard OS-appropriate download location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if can't get home dir
		return filepath.Join(".", "downloads")
	}

	// Create OS-appropriate downloads folder
	// This works for Windows, Mac, and Linux
	return filepath.Join(homeDir, "Music", "Crescendo")
}

// UserSettings represents the user's personal settings
type UserSettings struct {
	DownloadLocation string `json:"downloadLocation"`
}

// getSettingsFilePath returns the path to the settings file
func getSettingsFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".crescendo-settings.json")
}

// getUserDownloadLocation loads the user's preferred download location from settings file
func getUserDownloadLocation() string {
	settingsPath := getSettingsFilePath()

	// If file doesn't exist, return empty string to fall back to env vars
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return ""
	}

	// Read and parse the settings file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}

	var settings UserSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return ""
	}

	return settings.DownloadLocation
}
