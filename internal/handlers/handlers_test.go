package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattHbrook/Crescendo/internal/db"
	"github.com/MattHbrook/Crescendo/internal/discovery"
	"github.com/MattHbrook/Crescendo/internal/downloader"
	"github.com/MattHbrook/Crescendo/internal/hifi"
	"github.com/MattHbrook/Crescendo/internal/library"
	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockStore struct {
	artists   []db.ArtistMapping
	active    []db.Download
	history   []db.Download
	errList   error
	errActive error
	errHist   error
}

func (m *mockStore) ListArtistMappings(_ context.Context) ([]db.ArtistMapping, error) {
	return m.artists, m.errList
}

func (m *mockStore) GetActiveDownloads(_ context.Context) ([]db.Download, error) {
	return m.active, m.errActive
}

func (m *mockStore) GetDownloadHistory(_ context.Context, _ int) ([]db.Download, error) {
	return m.history, m.errHist
}

type mockHiFi struct {
	artists      *hifi.SearchResult[hifi.Artist]
	albums       *hifi.SearchResult[hifi.Album]
	artistAlbums []hifi.Album
	albumDetail  *hifi.AlbumDetail
	errArtists   error
	errAlbums    error
	errArtAlb    error
	errAlbDet    error
}

func (m *mockHiFi) SearchArtists(_ context.Context, _ string, _, _ int) (*hifi.SearchResult[hifi.Artist], error) {
	return m.artists, m.errArtists
}

func (m *mockHiFi) SearchAlbums(_ context.Context, _ string, _, _ int) (*hifi.SearchResult[hifi.Album], error) {
	return m.albums, m.errAlbums
}

func (m *mockHiFi) GetArtistAlbums(_ context.Context, _ int64) ([]hifi.Album, error) {
	return m.artistAlbums, m.errArtAlb
}

func (m *mockHiFi) GetAlbum(_ context.Context, _ int64) (*hifi.AlbumDetail, error) {
	return m.albumDetail, m.errAlbDet
}

type mockScanner struct {
	result *library.ScanResult
	err    error
}

func (m *mockScanner) Scan(_ context.Context) (*library.ScanResult, error) {
	return m.result, m.err
}

type mockDownloader struct {
	lastReq downloader.Request
	called  bool
}

func (m *mockDownloader) DownloadAsync(_ context.Context, req downloader.Request) {
	m.lastReq = req
	m.called = true
}

type mockDiscovery struct {
	recs []discovery.Recommendation
	err  error
}

func (m *mockDiscovery) Discover(_ context.Context, _, _ int) ([]discovery.Recommendation, error) {
	return m.recs, m.err
}

// ---------------------------------------------------------------------------
// Template setup helper
// ---------------------------------------------------------------------------

// writeTemplates creates a minimal set of template files in a temp directory
// and returns its path. The templates are just enough for the handlers to
// execute without error.
func writeTemplates(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	files := map[string]string{
		"layout.html": `{{define "layout"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`,
		"home.html":   `{{define "content"}}ok{{end}}`,
		"search.html": `{{define "content"}}ok{{end}}`,
		"search_results.html": `{{define "search_results"}}results{{end}}
{{define "content"}}search results{{end}}`,
		"artist.html":    `{{define "content"}}ok{{end}}`,
		"album.html":     `{{define "content"}}ok{{end}}`,
		"downloads.html": `{{define "content"}}ok{{end}}`,
		"discover.html":  `{{define "content"}}ok{{end}}`,
		"library.html":   `{{define "content"}}ok{{end}}`,
		"error.html":     `{{define "content"}}ok{{end}}`,
		"download_status.html": `{{define "download_status"}}queued{{end}}
{{define "content"}}download status{{end}}`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("writing template %s: %v", name, err)
		}
	}

	return dir
}

// newTestHandler creates a Handler wired up with the given mocks and a valid
// template directory. It fails the test if construction fails.
func newTestHandler(t *testing.T, store HandlerStore, hf HandlerHiFi, scanner HandlerScanner, dl HandlerDownloader, disc HandlerDiscovery) *Handler {
	t.Helper()

	dir := writeTemplates(t)

	h, err := New(dir, store, hf, scanner, dl, disc, "LOSSLESS")
	if err != nil {
		t.Fatalf("creating handler: %v", err)
	}
	return h
}

// chiContext injects a chi URL parameter into the request context so that
// chi.URLParam works in tests without a full router.
func chiContext(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	t.Run("fails with bad template dir", func(t *testing.T) {
		_, err := New("/no/such/dir", &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{}, "LOSSLESS")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("succeeds with valid template dir", func(t *testing.T) {
		dir := writeTemplates(t)

		h, err := New(dir, &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{}, "LOSSLESS")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if h == nil {
			t.Fatal("expected non-nil handler")
		}
	})
}

func TestHome(t *testing.T) {
	t.Run("returns 200", func(t *testing.T) {
		h := newTestHandler(t, &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.Home(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("expected text/html content type, got %q", ct)
		}
	})
}

func TestSearchResults(t *testing.T) {
	t.Run("artists search returns 200", func(t *testing.T) {
		hf := &mockHiFi{
			artists: &hifi.SearchResult[hifi.Artist]{
				Items: []hifi.Artist{
					{ID: 1, Name: "Radiohead"},
					{ID: 2, Name: "Radiohead Tribute"},
				},
				Total: 2,
			},
		}
		h := newTestHandler(t, &mockStore{}, hf, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/search/results?q=radiohead&type=artists", nil)
		rec := httptest.NewRecorder()

		h.SearchResults(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("albums search returns 200", func(t *testing.T) {
		hf := &mockHiFi{
			albums: &hifi.SearchResult[hifi.Album]{
				Items: []hifi.Album{
					{ID: 100, Title: "OK Computer", Artist: hifi.ArtistRef{ID: 1, Name: "Radiohead"}},
				},
				Total: 1,
			},
		}
		h := newTestHandler(t, &mockStore{}, hf, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/search/results?q=ok+computer&type=albums", nil)
		rec := httptest.NewRecorder()

		h.SearchResults(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("missing query returns 400", func(t *testing.T) {
		h := newTestHandler(t, &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/search/results", nil)
		rec := httptest.NewRecorder()

		h.SearchResults(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestArtist(t *testing.T) {
	t.Run("valid ID returns 200", func(t *testing.T) {
		hf := &mockHiFi{
			artistAlbums: []hifi.Album{
				{
					ID:    100,
					Title: "OK Computer",
					Artist: hifi.ArtistRef{
						ID:   1,
						Name: "Radiohead",
					},
				},
			},
		}
		h := newTestHandler(t, &mockStore{}, hf, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/artist/1", nil)
		req = chiContext(req, "id", "1")
		rec := httptest.NewRecorder()

		h.Artist(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("invalid ID returns 400", func(t *testing.T) {
		h := newTestHandler(t, &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/artist/abc", nil)
		req = chiContext(req, "id", "abc")
		rec := httptest.NewRecorder()

		h.Artist(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestAlbum(t *testing.T) {
	t.Run("valid ID returns 200", func(t *testing.T) {
		hf := &mockHiFi{
			albumDetail: &hifi.AlbumDetail{
				Album: hifi.Album{
					ID:    100,
					Title: "OK Computer",
					Artist: hifi.ArtistRef{
						ID:   1,
						Name: "Radiohead",
					},
				},
				Tracks: []hifi.Track{
					{ID: 1, Title: "Airbag", Duration: 284, TrackNumber: 1},
					{ID: 2, Title: "Paranoid Android", Duration: 383, TrackNumber: 2},
				},
			},
		}
		h := newTestHandler(t, &mockStore{}, hf, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/album/100", nil)
		req = chiContext(req, "id", "100")
		rec := httptest.NewRecorder()

		h.Album(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("invalid ID returns 400", func(t *testing.T) {
		h := newTestHandler(t, &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/album/xyz", nil)
		req = chiContext(req, "id", "xyz")
		rec := httptest.NewRecorder()

		h.Album(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestDownloads(t *testing.T) {
	t.Run("returns 200", func(t *testing.T) {
		store := &mockStore{
			active: []db.Download{
				{ID: 1, TidalAlbumID: 100, ArtistName: "Radiohead", AlbumTitle: "OK Computer", Quality: "LOSSLESS", Status: "downloading", Progress: 50.0, TotalTracks: 12, CompletedTracks: 6},
			},
			history: []db.Download{
				{ID: 2, TidalAlbumID: 200, ArtistName: "Pink Floyd", AlbumTitle: "Wish You Were Here", Quality: "LOSSLESS", Status: "complete", Progress: 100.0, TotalTracks: 5, CompletedTracks: 5},
			},
		}
		h := newTestHandler(t, store, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/downloads", nil)
		rec := httptest.NewRecorder()

		h.Downloads(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestDiscover(t *testing.T) {
	t.Run("returns 200", func(t *testing.T) {
		disc := &mockDiscovery{
			recs: []discovery.Recommendation{
				{
					AlbumID:     300,
					AlbumTitle:  "In Rainbows",
					ArtistName:  "Radiohead",
					Cover:       "cover-uuid",
					ReleaseDate: "2007-10-10",
					SeedArtist:  "Muse",
				},
			},
		}
		h := newTestHandler(t, &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, disc)

		req := httptest.NewRequest(http.MethodGet, "/discover", nil)
		rec := httptest.NewRecorder()

		h.Discover(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestLibrary(t *testing.T) {
	t.Run("returns 200", func(t *testing.T) {
		tidalID := int64(1)
		tidalName := "Radiohead"
		store := &mockStore{
			artists: []db.ArtistMapping{
				{ID: 1, FolderName: "Radiohead", TidalID: &tidalID, TidalName: &tidalName},
			},
		}
		h := newTestHandler(t, store, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodGet, "/library", nil)
		rec := httptest.NewRecorder()

		h.Library(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestStartDownload(t *testing.T) {
	t.Run("valid album_id returns 200", func(t *testing.T) {
		dl := &mockDownloader{}
		hf := &mockHiFi{
			albumDetail: &hifi.AlbumDetail{
				Album: hifi.Album{
					ID:    100,
					Title: "OK Computer",
					Artist: hifi.ArtistRef{
						ID:   1,
						Name: "Radiohead",
					},
				},
				Tracks: []hifi.Track{
					{ID: 1, Title: "Airbag", Duration: 284, TrackNumber: 1},
				},
			},
		}
		h := newTestHandler(t, &mockStore{}, hf, &mockScanner{}, dl, &mockDiscovery{})

		form := url.Values{}
		form.Set("album_id", "100")
		form.Set("quality", "LOSSLESS")

		req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		h.StartDownload(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if !dl.called {
			t.Fatal("expected DownloadAsync to be called")
		}
		if dl.lastReq.TidalAlbumID != 100 {
			t.Fatalf("expected TidalAlbumID 100, got %d", dl.lastReq.TidalAlbumID)
		}
		if dl.lastReq.Quality != "LOSSLESS" {
			t.Fatalf("expected quality LOSSLESS, got %q", dl.lastReq.Quality)
		}
	})

	t.Run("uses default quality when not specified", func(t *testing.T) {
		dl := &mockDownloader{}
		hf := &mockHiFi{
			albumDetail: &hifi.AlbumDetail{
				Album: hifi.Album{
					ID:    100,
					Title: "OK Computer",
					Artist: hifi.ArtistRef{
						ID:   1,
						Name: "Radiohead",
					},
				},
				Tracks: []hifi.Track{
					{ID: 1, Title: "Airbag", Duration: 284, TrackNumber: 1},
				},
			},
		}
		h := newTestHandler(t, &mockStore{}, hf, &mockScanner{}, dl, &mockDiscovery{})

		form := url.Values{}
		form.Set("album_id", "100")

		req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		h.StartDownload(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if dl.lastReq.Quality != "LOSSLESS" {
			t.Fatalf("expected default quality LOSSLESS, got %q", dl.lastReq.Quality)
		}
	})

	t.Run("invalid album_id returns 400", func(t *testing.T) {
		h := newTestHandler(t, &mockStore{}, &mockHiFi{}, &mockScanner{}, &mockDownloader{}, &mockDiscovery{})

		form := url.Values{}
		form.Set("album_id", "notanumber")

		req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		h.StartDownload(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestStartScan(t *testing.T) {
	t.Run("returns 200 with JSON", func(t *testing.T) {
		scanner := &mockScanner{
			result: &library.ScanResult{
				ArtistsFound:   10,
				AlbumsFound:    25,
				ArtistsMatched: 8,
				Errors:         []string{"some error"},
			},
		}
		h := newTestHandler(t, &mockStore{}, &mockHiFi{}, scanner, &mockDownloader{}, &mockDiscovery{})

		req := httptest.NewRequest(http.MethodPost, "/scan", nil)
		rec := httptest.NewRecorder()

		h.StartScan(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected application/json content type, got %q", ct)
		}

		body := rec.Body.String()
		want := `{"artists_found":10,"albums_found":25,"artists_matched":8,"errors":1}`
		if body != want {
			t.Fatalf("unexpected body:\n got: %s\nwant: %s", body, want)
		}
	})
}
