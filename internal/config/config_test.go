package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Prevent a .env file in the repo root from polluting test results.
	clearEnv(t)

	t.Run("defaults are applied when env vars are empty", func(t *testing.T) {
		clearEnv(t)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertString(t, "Port", cfg.Port, "8888")
		assertString(t, "HiFiAPIURL", cfg.HiFiAPIURL, "http://localhost:8000")
		assertString(t, "MusicPath", cfg.MusicPath, "/music")
		assertString(t, "DataPath", cfg.DataPath, "/data")
		assertString(t, "DefaultQuality", cfg.DefaultQuality, "LOSSLESS")
		assertInt(t, "MaxConcurrentDownloads", cfg.MaxConcurrentDownloads, 3)
	})

	envOverrides := []struct {
		name   string
		envKey string
		envVal string
		check  func(*testing.T, *Config)
	}{
		{
			name:   "PORT override",
			envKey: "PORT",
			envVal: "9090",
			check:  func(t *testing.T, c *Config) { assertString(t, "Port", c.Port, "9090") },
		},
		{
			name:   "HIFI_API_URL override",
			envKey: "HIFI_API_URL",
			envVal: "http://hifi:5000",
			check:  func(t *testing.T, c *Config) { assertString(t, "HiFiAPIURL", c.HiFiAPIURL, "http://hifi:5000") },
		},
		{
			name:   "MUSIC_PATH override",
			envKey: "MUSIC_PATH",
			envVal: "/mnt/nas/music",
			check:  func(t *testing.T, c *Config) { assertString(t, "MusicPath", c.MusicPath, "/mnt/nas/music") },
		},
		{
			name:   "DATA_PATH override",
			envKey: "DATA_PATH",
			envVal: "/var/lib/crescendo",
			check:  func(t *testing.T, c *Config) { assertString(t, "DataPath", c.DataPath, "/var/lib/crescendo") },
		},
		{
			name:   "DEFAULT_QUALITY override to HI_RES_LOSSLESS",
			envKey: "DEFAULT_QUALITY",
			envVal: "HI_RES_LOSSLESS",
			check:  func(t *testing.T, c *Config) { assertString(t, "DefaultQuality", c.DefaultQuality, "HI_RES_LOSSLESS") },
		},
		{
			name:   "MAX_CONCURRENT_DOWNLOADS override",
			envKey: "MAX_CONCURRENT_DOWNLOADS",
			envVal: "10",
			check:  func(t *testing.T, c *Config) { assertInt(t, "MaxConcurrentDownloads", c.MaxConcurrentDownloads, 10) },
		},
	}

	for _, tc := range envOverrides {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(tc.envKey, tc.envVal)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.check(t, cfg)
		})
	}

	validationErrors := []struct {
		name   string
		envKey string
		envVal string
		errSub string
	}{
		{
			name:   "invalid quality value",
			envKey: "DEFAULT_QUALITY",
			envVal: "MP3",
			errSub: "invalid DEFAULT_QUALITY",
		},
		{
			name:   "non-numeric max concurrent downloads",
			envKey: "MAX_CONCURRENT_DOWNLOADS",
			envVal: "abc",
			errSub: "invalid MAX_CONCURRENT_DOWNLOADS",
		},
		{
			name:   "zero max concurrent downloads",
			envKey: "MAX_CONCURRENT_DOWNLOADS",
			envVal: "0",
			errSub: "must be >= 1",
		},
		{
			name:   "negative max concurrent downloads",
			envKey: "MAX_CONCURRENT_DOWNLOADS",
			envVal: "-5",
			errSub: "must be >= 1",
		},
	}

	for _, tc := range validationErrors {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(tc.envKey, tc.envVal)

			_, err := Load()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tc.errSub) {
				t.Fatalf("error %q should contain %q", err.Error(), tc.errSub)
			}
		})
	}
}

// clearEnv unsets all config-related env vars for test isolation.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PORT",
		"HIFI_API_URL",
		"MUSIC_PATH",
		"DATA_PATH",
		"DEFAULT_QUALITY",
		"MAX_CONCURRENT_DOWNLOADS",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func assertString(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertInt(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", field, got, want)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
