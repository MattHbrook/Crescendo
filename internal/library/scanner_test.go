package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattHbrook/Crescendo/internal/db"
	"github.com/MattHbrook/Crescendo/internal/hifi"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

type albumRecord struct {
	artistFolder string
	albumFolder  string
	trackCount   int
	path         string
}

type mockStore struct {
	mappings        map[string]*db.ArtistMapping
	albums          []albumRecord
	upsertArtistErr error
	getArtistErr    error
	upsertAlbumErr  error
}

func (m *mockStore) GetArtistMapping(_ context.Context, folderName string) (*db.ArtistMapping, error) {
	if m.getArtistErr != nil {
		return nil, m.getArtistErr
	}
	mapping, ok := m.mappings[folderName]
	if !ok {
		return nil, nil
	}
	return mapping, nil
}

func (m *mockStore) UpsertArtistMapping(_ context.Context, folderName string, tidalID int64, tidalName, pictureURL string) error {
	if m.upsertArtistErr != nil {
		return m.upsertArtistErr
	}
	m.mappings[folderName] = &db.ArtistMapping{
		FolderName: folderName,
		TidalID:    &tidalID,
		TidalName:  &tidalName,
		PictureURL: &pictureURL,
	}
	return nil
}

func (m *mockStore) UpsertLibraryAlbum(_ context.Context, artistFolder, albumFolder string, trackCount int, path string) error {
	if m.upsertAlbumErr != nil {
		return m.upsertAlbumErr
	}
	m.albums = append(m.albums, albumRecord{
		artistFolder: artistFolder,
		albumFolder:  albumFolder,
		trackCount:   trackCount,
		path:         path,
	})
	return nil
}

func (m *mockStore) DeleteLibraryAlbumsByArtist(_ context.Context, _ string) error {
	return nil // not used by scanner
}

type mockSearcher struct {
	results   map[string][]hifi.Artist // key is search query
	searchErr error
}

func (m *mockSearcher) SearchArtists(_ context.Context, query string, _, _ int) (*hifi.SearchResult[hifi.Artist], error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	items, ok := m.results[query]
	if !ok {
		return &hifi.SearchResult[hifi.Artist]{}, nil
	}
	return &hifi.SearchResult[hifi.Artist]{Items: items, Total: len(items)}, nil
}

// ---------------------------------------------------------------------------
// Test helper
// ---------------------------------------------------------------------------

// setupMusicDir creates a temporary directory tree that mimics a real music
// library. It returns the path to the root directory.
//
//	<root>/
//	  ArtistA/
//	    Album1/
//	      01.flac
//	      02.flac
//	      ._hidden.flac   (macOS artifact, should be skipped)
//	    Album2/
//	      track.flac
//	  ArtistB/
//	    Album3/
//	      song.flac
//	  Playlists/           (should be skipped entirely)
//	    mix.flac
//	  EmptyArtist/         (no albums)
//	  ArtistC/
//	    EmptyAlbum/        (no .flac files)
func setupMusicDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		filepath.Join(root, "ArtistA", "Album1"),
		filepath.Join(root, "ArtistA", "Album2"),
		filepath.Join(root, "ArtistB", "Album3"),
		filepath.Join(root, "Playlists"),
		filepath.Join(root, "EmptyArtist"),
		filepath.Join(root, "ArtistC", "EmptyAlbum"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("creating directory %s: %v", d, err)
		}
	}

	files := []string{
		filepath.Join(root, "ArtistA", "Album1", "01.flac"),
		filepath.Join(root, "ArtistA", "Album1", "02.flac"),
		filepath.Join(root, "ArtistA", "Album1", "._hidden.flac"),
		filepath.Join(root, "ArtistA", "Album2", "track.flac"),
		filepath.Join(root, "ArtistB", "Album3", "song.flac"),
		filepath.Join(root, "Playlists", "mix.flac"),
	}
	for _, f := range files {
		fh, err := os.Create(f)
		if err != nil {
			t.Fatalf("creating file %s: %v", f, err)
		}
		fh.Close()
	}

	return root
}

// ---------------------------------------------------------------------------
// Scanner tests
// ---------------------------------------------------------------------------

func TestScan(t *testing.T) {
	root := setupMusicDir(t)

	store := &mockStore{
		mappings: make(map[string]*db.ArtistMapping),
	}

	searcher := &mockSearcher{
		results: map[string][]hifi.Artist{
			"ArtistA": {{ID: 100, Name: "Artist A", Picture: "pic-a"}},
			"ArtistB": {{ID: 200, Name: "Artist B", Picture: "pic-b"}},
		},
	}

	scanner := NewScanner(root, store, searcher)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() returned unexpected error: %v", err)
	}

	// ArtistA, ArtistB, EmptyArtist, ArtistC (Playlists skipped)
	if result.ArtistsFound != 4 {
		t.Errorf("ArtistsFound = %d, want 4", result.ArtistsFound)
	}

	// Album1 (2 tracks), Album2 (1 track), Album3 (1 track)
	// EmptyAlbum has 0 .flac files so it is skipped
	if result.AlbumsFound != 3 {
		t.Errorf("AlbumsFound = %d, want 3", result.AlbumsFound)
	}

	// ArtistA and ArtistB matched via searcher
	if result.ArtistsMatched != 2 {
		t.Errorf("ArtistsMatched = %d, want 2", result.ArtistsMatched)
	}

	// EmptyArtist and ArtistC should produce "no Tidal match" errors
	foundEmptyArtistErr := false
	foundArtistCErr := false
	for _, e := range result.Errors {
		if contains(e, "EmptyArtist") {
			foundEmptyArtistErr = true
		}
		if contains(e, "ArtistC") {
			foundArtistCErr = true
		}
	}
	if !foundEmptyArtistErr {
		t.Error("expected error message about EmptyArtist, got none")
	}
	if !foundArtistCErr {
		t.Error("expected error message about ArtistC, got none")
	}

	// Verify album records stored
	if len(store.albums) != 3 {
		t.Fatalf("len(store.albums) = %d, want 3", len(store.albums))
	}

	albumTracks := make(map[string]int)
	for _, a := range store.albums {
		albumTracks[a.albumFolder] = a.trackCount
	}

	wantTracks := map[string]int{
		"Album1": 2,
		"Album2": 1,
		"Album3": 1,
	}
	for album, wantCount := range wantTracks {
		if gotCount, ok := albumTracks[album]; !ok {
			t.Errorf("album %q not found in store", album)
		} else if gotCount != wantCount {
			t.Errorf("album %q track count = %d, want %d", album, gotCount, wantCount)
		}
	}

	// Verify artist mappings were stored for matched artists
	if _, ok := store.mappings["ArtistA"]; !ok {
		t.Error("expected mapping for ArtistA, not found")
	}
	if _, ok := store.mappings["ArtistB"]; !ok {
		t.Error("expected mapping for ArtistB, not found")
	}
}

func TestScan_SkipsAlreadyMappedArtists(t *testing.T) {
	root := t.TempDir()

	// Create a single artist with one album and one track.
	albumDir := filepath.Join(root, "MappedArtist", "SomeAlbum")
	if err := os.MkdirAll(albumDir, 0o755); err != nil {
		t.Fatalf("creating directory: %v", err)
	}
	f, err := os.Create(filepath.Join(albumDir, "track.flac"))
	if err != nil {
		t.Fatalf("creating file: %v", err)
	}
	f.Close()

	tidalID := int64(999)
	store := &mockStore{
		mappings: map[string]*db.ArtistMapping{
			"MappedArtist": {
				FolderName: "MappedArtist",
				TidalID:    &tidalID,
			},
		},
	}

	// Set a sentinel error on the searcher. If the scanner calls
	// SearchArtists the error will surface in the scan results.
	sentinelErr := errors.New("searcher should not be called")
	searcher := &mockSearcher{
		searchErr: sentinelErr,
	}

	scanner := NewScanner(root, store, searcher)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() returned unexpected error: %v", err)
	}

	// The searcher should not have been called, so no errors should
	// reference the sentinel message.
	for _, e := range result.Errors {
		if contains(e, "searcher should not be called") {
			t.Error("searcher was called for an already-mapped artist")
		}
	}

	if result.ArtistsFound != 1 {
		t.Errorf("ArtistsFound = %d, want 1", result.ArtistsFound)
	}
}

func TestScan_InvalidMusicDir(t *testing.T) {
	store := &mockStore{
		mappings: make(map[string]*db.ArtistMapping),
	}
	searcher := &mockSearcher{}

	scanner := NewScanner("/nonexistent/path/that/does/not/exist", store, searcher)
	_, err := scanner.Scan(context.Background())
	if err == nil {
		t.Fatal("Scan() with invalid directory should return an error, got nil")
	}
}

func TestCountFLACFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []string // filenames to create in the test directory
		want  int
	}{
		{
			name:  "mixed file types",
			files: []string{"song.flac", "cover.jpg", "notes.mp3", "bonus.flac"},
			want:  2,
		},
		{
			name:  "macOS artifacts skipped",
			files: []string{"01.flac", "._02.flac", "03.flac"},
			want:  2,
		},
		{
			name:  "case insensitive flac extension",
			files: []string{"track.FLAC", "song.Flac", "other.fLaC"},
			want:  3,
		},
		{
			name:  "empty directory",
			files: nil,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tt.files {
				f, err := os.Create(filepath.Join(dir, name))
				if err != nil {
					t.Fatalf("creating file %s: %v", name, err)
				}
				f.Close()
			}

			got, err := countFLACFiles(dir)
			if err != nil {
				t.Fatalf("countFLACFiles() returned unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("countFLACFiles() = %d, want %d", got, tt.want)
			}
		})
	}
}

// contains reports whether s contains substr. Defined here to avoid importing
// strings in the test file just for this one check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
