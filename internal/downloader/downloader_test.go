package downloader

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MattHbrook/Crescendo/internal/hifi"
	"github.com/MattHbrook/Crescendo/internal/manifest"
)

// --- mock implementations ---

type mockPlayer struct {
	playbacks map[int64]*hifi.Playback
	err       error
}

func (m *mockPlayer) GetTrackPlayback(_ context.Context, id int64, _ string) (*hifi.Playback, error) {
	if m.err != nil {
		return nil, m.err
	}
	pb, ok := m.playbacks[id]
	if !ok {
		return nil, fmt.Errorf("no playback for track %d", id)
	}
	return pb, nil
}

type mockAlbumFetcher struct {
	albums map[int64]*hifi.AlbumDetail
	err    error
}

func (m *mockAlbumFetcher) GetAlbum(_ context.Context, id int64) (*hifi.AlbumDetail, error) {
	if m.err != nil {
		return nil, m.err
	}
	album, ok := m.albums[id]
	if !ok {
		return nil, fmt.Errorf("no album with id %d", id)
	}
	return album, nil
}

type downloadRecord struct {
	tidalAlbumID int64
	artistName   string
	albumTitle   string
	quality      string
	totalTracks  int
}

type progressUpdate struct {
	id              int64
	completedTracks int
	progress        float64
}

type failRecord struct {
	id     int64
	errMsg string
}

type mockDownloadStore struct {
	mu              sync.Mutex
	downloads       map[int64]*downloadRecord
	nextID          int64
	createErr       error
	progressUpdates []progressUpdate
	completed       []int64
	failed          []failRecord
}

func newMockDownloadStore() *mockDownloadStore {
	return &mockDownloadStore{
		downloads: make(map[int64]*downloadRecord),
		nextID:    1,
	}
}

func (s *mockDownloadStore) CreateDownload(_ context.Context, tidalAlbumID int64, artistName, albumTitle, quality string, totalTracks int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.createErr != nil {
		return 0, s.createErr
	}
	id := s.nextID
	s.nextID++
	s.downloads[id] = &downloadRecord{
		tidalAlbumID: tidalAlbumID,
		artistName:   artistName,
		albumTitle:   albumTitle,
		quality:      quality,
		totalTracks:  totalTracks,
	}
	return id, nil
}

func (s *mockDownloadStore) UpdateDownloadProgress(_ context.Context, id int64, completedTracks int, progress float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progressUpdates = append(s.progressUpdates, progressUpdate{
		id:              id,
		completedTracks: completedTracks,
		progress:        progress,
	})
	return nil
}

func (s *mockDownloadStore) CompleteDownload(_ context.Context, id int64, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completed = append(s.completed, id)
	return nil
}

func (s *mockDownloadStore) FailDownload(_ context.Context, id int64, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed = append(s.failed, failRecord{id: id, errMsg: errMsg})
	return nil
}

type mockCoverFetcher struct {
	covers map[int64]*hifi.Cover
	err    error
}

func (m *mockCoverFetcher) GetCover(_ context.Context, albumID int64) (*hifi.Cover, error) {
	if m.err != nil {
		return nil, m.err
	}
	c, ok := m.covers[albumID]
	if !ok {
		return nil, fmt.Errorf("no cover for album %d", albumID)
	}
	return c, nil
}

// noCoverFetcher returns a mock that always errors (cover art unavailable).
func noCoverFetcher() *mockCoverFetcher {
	return &mockCoverFetcher{err: fmt.Errorf("no covers")}
}

// --- helpers ---

// encodeBTSManifest builds a base64-encoded BTS manifest JSON pointing at the
// given URL with audio/flac codec.
func encodeBTSManifest(url string) string {
	m := struct {
		MimeType       string   `json:"mimeType"`
		Codecs         string   `json:"codecs"`
		EncryptionType string   `json:"encryptionType"`
		URLs           []string `json:"urls"`
	}{
		MimeType:       "audio/flac",
		Codecs:         "flac",
		EncryptionType: "NONE",
		URLs:           []string{url},
	}
	data, _ := json.Marshal(m)
	return base64.StdEncoding.EncodeToString(data)
}

// --- tests ---

func TestDownload_Success(t *testing.T) {
	fakeFlac := []byte("fake-flac-data")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "audio/flac")
		w.Write(fakeFlac)
	}))
	defer srv.Close()

	b64Manifest := encodeBTSManifest(srv.URL)

	player := &mockPlayer{
		playbacks: map[int64]*hifi.Playback{
			1: {
				TrackID:          1,
				AudioQuality:     "LOSSLESS",
				ManifestMimeType: manifest.MimeTypeBTS,
				Manifest:         b64Manifest,
			},
			2: {
				TrackID:          2,
				AudioQuality:     "LOSSLESS",
				ManifestMimeType: manifest.MimeTypeBTS,
				Manifest:         b64Manifest,
			},
		},
	}

	fetcher := &mockAlbumFetcher{
		albums: map[int64]*hifi.AlbumDetail{
			42: {
				Album: hifi.Album{
					ID:    42,
					Title: "Test Album",
					Artist: hifi.ArtistRef{
						ID:   1,
						Name: "Test Artist",
					},
				},
				Tracks: []hifi.Track{
					{ID: 1, Title: "Song One", TrackNumber: 1},
					{ID: 2, Title: "Song Two", TrackNumber: 2},
				},
			},
		},
	}

	store := newMockDownloadStore()
	tmpDir := t.TempDir()
	dl := New(tmpDir, 3, player, fetcher, noCoverFetcher(), store)

	err := dl.Download(context.Background(), Request{
		TidalAlbumID: 42,
		Quality:      "LOSSLESS",
	})
	if err != nil {
		t.Fatalf("Download returned unexpected error: %v", err)
	}

	// Verify store interactions.
	store.mu.Lock()
	defer store.mu.Unlock()

	if got := len(store.downloads); got != 1 {
		t.Errorf("expected 1 download record, got %d", got)
	}
	if got := len(store.progressUpdates); got != 2 {
		t.Errorf("expected 2 progress updates, got %d", got)
	}
	if got := len(store.completed); got != 1 {
		t.Errorf("expected 1 completion, got %d", got)
	}
	if got := len(store.failed); got != 0 {
		t.Errorf("expected 0 failures, got %d", got)
	}

	// Verify FLAC files were written to disk.
	track1 := filepath.Join(tmpDir, "Test Artist", "Test Album", "01 - Song One.flac")
	track2 := filepath.Join(tmpDir, "Test Artist", "Test Album", "02 - Song Two.flac")

	for _, path := range []string{track1, track2} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected file at %s: %v", path, err)
			continue
		}
		if string(data) != string(fakeFlac) {
			t.Errorf("file %s: got %q, want %q", path, data, fakeFlac)
		}
	}
}

func TestDownload_AlbumFetchError(t *testing.T) {
	fetcher := &mockAlbumFetcher{err: fmt.Errorf("tidal API down")}
	player := &mockPlayer{}
	store := newMockDownloadStore()

	dl := New(t.TempDir(), 3, player, fetcher, noCoverFetcher(), store)

	err := dl.Download(context.Background(), Request{
		TidalAlbumID: 99,
		Quality:      "LOSSLESS",
	})
	if err == nil {
		t.Fatal("expected an error when album fetch fails, got nil")
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	// Error occurs before CreateDownload, so no records should exist.
	if got := len(store.downloads); got != 0 {
		t.Errorf("expected 0 download records, got %d", got)
	}
}

func TestDownload_TrackPlaybackError(t *testing.T) {
	fetcher := &mockAlbumFetcher{
		albums: map[int64]*hifi.AlbumDetail{
			10: {
				Album: hifi.Album{
					ID:    10,
					Title: "Broken Album",
					Artist: hifi.ArtistRef{
						ID:   2,
						Name: "Broken Artist",
					},
				},
				Tracks: []hifi.Track{
					{ID: 100, Title: "Bad Track", TrackNumber: 1},
				},
			},
		},
	}

	player := &mockPlayer{err: fmt.Errorf("playback service unavailable")}
	store := newMockDownloadStore()

	dl := New(t.TempDir(), 3, player, fetcher, noCoverFetcher(), store)

	err := dl.Download(context.Background(), Request{
		TidalAlbumID: 10,
		Quality:      "LOSSLESS",
	})
	if err == nil {
		t.Fatal("expected an error when track playback fails, got nil")
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	// Download was created, then failed.
	if got := len(store.downloads); got != 1 {
		t.Errorf("expected 1 download record, got %d", got)
	}
	if got := len(store.failed); got != 1 {
		t.Errorf("expected 1 failure record, got %d", got)
	}
	if got := len(store.completed); got != 0 {
		t.Errorf("expected 0 completions, got %d", got)
	}
}

func TestDownload_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	b64Manifest := encodeBTSManifest(srv.URL)

	player := &mockPlayer{
		playbacks: map[int64]*hifi.Playback{
			5: {
				TrackID:          5,
				AudioQuality:     "LOSSLESS",
				ManifestMimeType: manifest.MimeTypeBTS,
				Manifest:         b64Manifest,
			},
		},
	}

	fetcher := &mockAlbumFetcher{
		albums: map[int64]*hifi.AlbumDetail{
			20: {
				Album: hifi.Album{
					ID:    20,
					Title: "Server Error Album",
					Artist: hifi.ArtistRef{
						ID:   3,
						Name: "Error Artist",
					},
				},
				Tracks: []hifi.Track{
					{ID: 5, Title: "Failing Track", TrackNumber: 1},
				},
			},
		},
	}

	store := newMockDownloadStore()
	dl := New(t.TempDir(), 3, player, fetcher, noCoverFetcher(), store)

	err := dl.Download(context.Background(), Request{
		TidalAlbumID: 20,
		Quality:      "LOSSLESS",
	})
	if err == nil {
		t.Fatal("expected an error when HTTP server returns 500, got nil")
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if got := len(store.failed); got != 1 {
		t.Errorf("expected 1 failure record, got %d", got)
	}
}

func TestDownloadAsync_Concurrency(t *testing.T) {
	fakeFlac := []byte("fake-flac-data")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "audio/flac")
		w.Write(fakeFlac)
	}))
	defer srv.Close()

	b64Manifest := encodeBTSManifest(srv.URL)

	// Build playbacks and albums for 3 separate albums, each with 1 track.
	playbacks := make(map[int64]*hifi.Playback)
	albums := make(map[int64]*hifi.AlbumDetail)

	for i := int64(1); i <= 3; i++ {
		trackID := i * 100
		playbacks[trackID] = &hifi.Playback{
			TrackID:          trackID,
			AudioQuality:     "LOSSLESS",
			ManifestMimeType: manifest.MimeTypeBTS,
			Manifest:         b64Manifest,
		}
		albums[i] = &hifi.AlbumDetail{
			Album: hifi.Album{
				ID:    i,
				Title: fmt.Sprintf("Album %d", i),
				Artist: hifi.ArtistRef{
					ID:   i,
					Name: fmt.Sprintf("Artist %d", i),
				},
			},
			Tracks: []hifi.Track{
				{ID: trackID, Title: fmt.Sprintf("Track %d", i), TrackNumber: 1},
			},
		}
	}

	player := &mockPlayer{playbacks: playbacks}
	fetcher := &mockAlbumFetcher{albums: albums}
	store := newMockDownloadStore()

	dl := New(t.TempDir(), 2, player, fetcher, noCoverFetcher(), store)

	for i := int64(1); i <= 3; i++ {
		dl.DownloadAsync(context.Background(), Request{
			TidalAlbumID: i,
			Quality:      "LOSSLESS",
		})
	}

	// Poll for completion since DownloadAsync is fire-and-forget.
	deadline := time.After(10 * time.Second)
	for {
		store.mu.Lock()
		done := len(store.completed)
		store.mu.Unlock()

		if done >= 3 {
			break
		}

		select {
		case <-deadline:
			store.mu.Lock()
			t.Fatalf("timed out waiting for 3 completions; got %d completed, %d failed",
				len(store.completed), len(store.failed))
			store.mu.Unlock()
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if got := len(store.downloads); got != 3 {
		t.Errorf("expected 3 download records, got %d", got)
	}
	if got := len(store.completed); got != 3 {
		t.Errorf("expected 3 completions, got %d", got)
	}
	if got := len(store.failed); got != 0 {
		t.Errorf("expected 0 failures, got %d", got)
	}
}
