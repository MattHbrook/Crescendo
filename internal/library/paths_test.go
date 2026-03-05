package library

import (
	"path/filepath"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"clean name passes through", "Green Day", "Green Day"},
		{"slashes replaced", "AC/DC", "AC_DC"},
		{"backslash replaced", "foo\\bar", "foo_bar"},
		{"colon replaced", "Vol: 2", "Vol_ 2"},
		{"multiple illegal chars collapse", "a::b", "a_b"},
		{"asterisk replaced", "Best*Hits", "Best_Hits"},
		{"question mark replaced", "Why?", "Why_"},
		{"angle brackets", "<tag>", "_tag_"},
		{"pipe", "A|B", "A_B"},
		{"quotes", `He said "hi"`, "He said _hi_"},
		{"trailing dots trimmed", "test...", "test"},
		{"leading/trailing spaces trimmed", "  hello  ", "hello"},
		{"empty string returns Unknown", "", "Unknown"},
		{"only dots returns Unknown", "...", "Unknown"},
		{"only illegal chars", ":::", "_"},
		{"unicode preserved", "한국어", "한국어"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestArtistDir(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		artist string
		want   string
	}{
		{
			name:   "clean artist name",
			root:   "/music",
			artist: "Pink Floyd",
			want:   filepath.Join("/music", "Pink Floyd"),
		},
		{
			name:   "artist name with illegal chars",
			root:   "/music",
			artist: "AC/DC",
			want:   filepath.Join("/music", "AC_DC"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArtistDir(tt.root, tt.artist)
			if got != tt.want {
				t.Errorf("ArtistDir(%q, %q) = %q, want %q", tt.root, tt.artist, got, tt.want)
			}
		})
	}
}

func TestAlbumDir(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		artist string
		album  string
		want   string
	}{
		{
			name:   "clean names",
			root:   "/music",
			artist: "Beatles",
			album:  "Abbey Road",
			want:   filepath.Join("/music", "Beatles", "Abbey Road"),
		},
		{
			name:   "both names sanitized",
			root:   "/music",
			artist: "AC/DC",
			album:  "Back in Black: Remaster",
			want:   filepath.Join("/music", "AC_DC", "Back in Black_ Remaster"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AlbumDir(tt.root, tt.artist, tt.album)
			if got != tt.want {
				t.Errorf("AlbumDir(%q, %q, %q) = %q, want %q", tt.root, tt.artist, tt.album, got, tt.want)
			}
		})
	}
}

func TestTrackFilename(t *testing.T) {
	tests := []struct {
		name    string
		trackNo int
		title   string
		want    string
	}{
		{"single digit padded", 1, "Song", "01 - Song.flac"},
		{"double digit", 12, "Song", "12 - Song.flac"},
		{"title sanitized", 3, "A/B", "03 - A_B.flac"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrackFilename(tt.trackNo, tt.title)
			if got != tt.want {
				t.Errorf("TrackFilename(%d, %q) = %q, want %q", tt.trackNo, tt.title, got, tt.want)
			}
		})
	}
}

func TestTrackPath(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		artist  string
		album   string
		trackNo int
		title   string
		want    string
	}{
		{
			name:    "full path assembly",
			root:    "/music",
			artist:  "Pink Floyd",
			album:   "The Wall",
			trackNo: 5,
			title:   "Another Brick",
			want:    filepath.Join("/music", "Pink Floyd", "The Wall", "05 - Another Brick.flac"),
		},
		{
			name:    "path with sanitization",
			root:    "/music",
			artist:  "AC/DC",
			album:   "Vol: 1",
			trackNo: 10,
			title:   "Track?Name",
			want:    filepath.Join("/music", "AC_DC", "Vol_ 1", "10 - Track_Name.flac"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrackPath(tt.root, tt.artist, tt.album, tt.trackNo, tt.title)
			if got != tt.want {
				t.Errorf("TrackPath(%q, %q, %q, %d, %q) = %q, want %q",
					tt.root, tt.artist, tt.album, tt.trackNo, tt.title, got, tt.want)
			}
		})
	}
}
