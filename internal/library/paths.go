package library

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var illegalChars = regexp.MustCompile(`[/\\:*?"<>|]`)
var multiUnder = regexp.MustCompile(`_{2,}`)

// SanitizeName replaces filesystem-unsafe characters with underscores,
// collapses runs of underscores, and trims spaces/dots so the result is
// safe on Windows, Linux, and macOS.
func SanitizeName(name string) string {
	s := illegalChars.ReplaceAllString(name, "_")
	s = multiUnder.ReplaceAllString(s, "_")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ".")
	if s == "" {
		return "Unknown"
	}
	return s
}

// ArtistDir returns the directory path for an artist inside the music root.
func ArtistDir(musicRoot, artistName string) string {
	return filepath.Join(musicRoot, SanitizeName(artistName))
}

// AlbumDir returns the directory path for an album nested under its artist.
func AlbumDir(musicRoot, artistName, albumTitle string) string {
	return filepath.Join(musicRoot, SanitizeName(artistName), SanitizeName(albumTitle))
}

// TrackFilename returns a zero-padded, sanitized FLAC filename for a track.
func TrackFilename(trackNumber int, title string) string {
	return fmt.Sprintf("%02d - %s.flac", trackNumber, SanitizeName(title))
}

// TrackPath returns the full path for a track file on disk.
func TrackPath(musicRoot, artistName, albumTitle string, trackNumber int, title string) string {
	return filepath.Join(AlbumDir(musicRoot, artistName, albumTitle), TrackFilename(trackNumber, title))
}
