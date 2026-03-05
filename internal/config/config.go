package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration sourced from environment variables.
type Config struct {
	Port                   string
	HiFiAPIURL             string
	MusicPath              string
	DataPath               string
	DefaultQuality         string
	MaxConcurrentDownloads int
}

// Load reads configuration from environment variables (optionally preceded by
// a .env file) and returns a validated Config. Missing variables fall back to
// sensible defaults so the app can run with zero configuration in Docker.
func Load() (*Config, error) {
	// Best-effort .env load; missing file is normal in production.
	_ = godotenv.Load()

	quality := envOrDefault("DEFAULT_QUALITY", "LOSSLESS")
	if quality != "LOSSLESS" && quality != "HI_RES_LOSSLESS" {
		return nil, fmt.Errorf("config: invalid DEFAULT_QUALITY %q, must be LOSSLESS or HI_RES_LOSSLESS", quality)
	}

	rawConcurrent := envOrDefault("MAX_CONCURRENT_DOWNLOADS", "3")
	concurrent, err := strconv.Atoi(rawConcurrent)
	if err != nil {
		return nil, fmt.Errorf("config: invalid MAX_CONCURRENT_DOWNLOADS %q: %w", rawConcurrent, err)
	}
	if concurrent < 1 {
		return nil, fmt.Errorf("config: MAX_CONCURRENT_DOWNLOADS must be >= 1, got %d", concurrent)
	}

	return &Config{
		Port:                   envOrDefault("PORT", "8888"),
		HiFiAPIURL:             envOrDefault("HIFI_API_URL", "http://localhost:8000"),
		MusicPath:              envOrDefault("MUSIC_PATH", "/music"),
		DataPath:               envOrDefault("DATA_PATH", "/data"),
		DefaultQuality:         quality,
		MaxConcurrentDownloads: concurrent,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
