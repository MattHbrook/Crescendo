package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MattHbrook/Crescendo/internal/db"
	"github.com/MattHbrook/Crescendo/internal/discovery"
	"github.com/MattHbrook/Crescendo/internal/downloader"
	"github.com/MattHbrook/Crescendo/internal/hifi"
	"github.com/MattHbrook/Crescendo/internal/library"
	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Dependency interfaces
// ---------------------------------------------------------------------------

// HandlerStore is the subset of db.Store used by HTTP handlers.
type HandlerStore interface {
	ListArtistMappings(ctx context.Context) ([]db.ArtistMapping, error)
	GetActiveDownloads(ctx context.Context) ([]db.Download, error)
	GetDownloadHistory(ctx context.Context, limit int) ([]db.Download, error)
}

// HandlerHiFi is the subset of hifi.Client used by HTTP handlers.
type HandlerHiFi interface {
	SearchArtists(ctx context.Context, query string, limit, offset int) (*hifi.SearchResult[hifi.Artist], error)
	SearchAlbums(ctx context.Context, query string, limit, offset int) (*hifi.SearchResult[hifi.Album], error)
	GetArtistAlbums(ctx context.Context, id int64) ([]hifi.Album, error)
	GetAlbum(ctx context.Context, id int64) (*hifi.AlbumDetail, error)
}

// HandlerScanner is the subset of library.Scanner used by HTTP handlers.
type HandlerScanner interface {
	Scan(ctx context.Context) (*library.ScanResult, error)
}

// HandlerDownloader is the subset of downloader.Downloader used by HTTP handlers.
type HandlerDownloader interface {
	DownloadAsync(ctx context.Context, req downloader.Request)
}

// HandlerDiscovery is the subset of discovery.Engine used by HTTP handlers.
type HandlerDiscovery interface {
	Discover(ctx context.Context, seedCount, maxResults int) ([]discovery.Recommendation, error)
}

// ---------------------------------------------------------------------------
// Template functions
// ---------------------------------------------------------------------------

var funcMap = template.FuncMap{
	"replace": strings.ReplaceAll,
	"formatDuration": func(seconds int) string {
		m := seconds / 60
		s := seconds % 60
		return fmt.Sprintf("%d:%02d", m, s)
	},
	"deref": func(v any) any {
		switch val := v.(type) {
		case *string:
			if val != nil {
				return *val
			}
			return ""
		case *int64:
			if val != nil {
				return *val
			}
			return int64(0)
		default:
			return v
		}
	},
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler holds all dependencies needed by HTTP handlers.
type Handler struct {
	templates  map[string]*template.Template
	store      HandlerStore
	hifi       HandlerHiFi
	scanner    HandlerScanner
	downloader HandlerDownloader
	discovery  HandlerDiscovery
	quality    string // default download quality from config
}

// New creates a Handler, parsing all HTML templates from templateDir.
func New(
	templateDir string,
	store HandlerStore,
	hifi HandlerHiFi,
	scanner HandlerScanner,
	dl HandlerDownloader,
	disc HandlerDiscovery,
	quality string,
) (*Handler, error) {
	// Parse layout as the base template that every page clones.
	base, err := template.New("layout").Funcs(funcMap).ParseFiles(filepath.Join(templateDir, "layout.html"))
	if err != nil {
		return nil, fmt.Errorf("handlers: parsing layout template: %w", err)
	}

	// Each page template defines {{define "content"}}...{{end}} which is
	// rendered inside the layout's {{block "content" .}} placeholder.
	// We clone the base for each page so the "content" definitions don't
	// collide across pages.
	pages := map[string]string{
		"home":            "home.html",
		"search":          "search.html",
		"search_results":  "search_results.html",
		"artist":          "artist.html",
		"album":           "album.html",
		"downloads":       "downloads.html",
		"discover":        "discover.html",
		"library":         "library.html",
		"download_status": "download_status.html",
		"error":           "error.html",
	}

	tmpl := make(map[string]*template.Template, len(pages))
	for name, file := range pages {
		t, err := template.Must(base.Clone()).ParseFiles(filepath.Join(templateDir, file))
		if err != nil {
			return nil, fmt.Errorf("handlers: parsing template %s: %w", file, err)
		}
		tmpl[name] = t
	}

	return &Handler{
		templates:  tmpl,
		store:      store,
		hifi:       hifi,
		scanner:    scanner,
		downloader: dl,
		discovery:  disc,
		quality:    quality,
	}, nil
}

// ---------------------------------------------------------------------------
// Template rendering helpers
// ---------------------------------------------------------------------------

// render executes a full-page template (layout + content block).
func (h *Handler) render(w http.ResponseWriter, page string, data any) {
	t, ok := h.templates[page]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// renderPartial executes a named block within a page template, without the
// layout wrapper. Used for HTMX partial responses.
func (h *Handler) renderPartial(w http.ResponseWriter, page, block string, data any) {
	t, ok := h.templates[page]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, block, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// renderError renders a full error page with the given HTTP status and message.
func (h *Handler) renderError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	h.render(w, "error", map[string]any{
		"Title": "Error",
		"Error": msg,
	})
}

// ---------------------------------------------------------------------------
// Route registration
// ---------------------------------------------------------------------------

// Routes registers all HTTP routes on the given chi router.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.Home)
	r.Get("/search", h.Search)
	r.Get("/search/results", h.SearchResults)
	r.Get("/artist/{id}", h.Artist)
	r.Get("/album/{id}", h.Album)
	r.Get("/downloads", h.Downloads)
	r.Get("/discover", h.Discover)
	r.Get("/library", h.Library)
	r.Post("/download", h.StartDownload)
	r.Post("/scan", h.StartScan)
}

// ---------------------------------------------------------------------------
// Page handlers
// ---------------------------------------------------------------------------

// Home renders the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.render(w, "home", map[string]any{"Title": "Home"})
}

// Search renders the search form page.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	h.render(w, "search", map[string]any{"Title": "Search"})
}

// SearchResults returns an HTMX partial with search results.
func (h *Handler) SearchResults(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("type")
	if q == "" {
		http.Error(w, "query required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	data := map[string]any{"Type": searchType}

	switch searchType {
	case "albums":
		result, err := h.hifi.SearchAlbums(ctx, q, 20, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data["Albums"] = result.Items
	default: // "artists" or anything else
		result, err := h.hifi.SearchArtists(ctx, q, 20, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data["Artists"] = result.Items
		data["Type"] = "artists"
	}

	h.renderPartial(w, "search_results", "search_results", data)
}

// Artist renders the artist page with their albums.
func (h *Handler) Artist(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid artist ID")
		return
	}

	albums, err := h.hifi.GetArtistAlbums(r.Context(), id)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to load artist")
		return
	}

	// We don't have a separate "get artist" endpoint, so use the first
	// album's artist info when available.
	artistName := idStr
	if len(albums) > 0 {
		artistName = albums[0].Artist.Name
	}

	h.render(w, "artist", map[string]any{
		"Title":  artistName,
		"Artist": map[string]any{"Name": artistName, "ID": id},
		"Albums": albums,
	})
}

// Album renders the album detail page with track list.
func (h *Handler) Album(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid album ID")
		return
	}

	detail, err := h.hifi.GetAlbum(r.Context(), id)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to load album")
		return
	}

	h.render(w, "album", map[string]any{
		"Title":   detail.Title,
		"Album":   detail.Album,
		"Tracks":  detail.Tracks,
		"Quality": h.quality,
	})
}

// Downloads renders the downloads page with active and historical downloads.
func (h *Handler) Downloads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	active, err := h.store.GetActiveDownloads(ctx)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to load downloads")
		return
	}

	history, err := h.store.GetDownloadHistory(ctx, 50)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to load download history")
		return
	}

	h.render(w, "downloads", map[string]any{
		"Title":   "Downloads",
		"Active":  active,
		"History": history,
	})
}

// Discover renders the discovery page with album recommendations.
func (h *Handler) Discover(w http.ResponseWriter, r *http.Request) {
	recs, err := h.discovery.Discover(r.Context(), 5, 20)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to generate recommendations")
		return
	}

	h.render(w, "discover", map[string]any{
		"Title":           "Discover",
		"Recommendations": recs,
	})
}

// Library renders the library page with all mapped artists.
func (h *Handler) Library(w http.ResponseWriter, r *http.Request) {
	artists, err := h.store.ListArtistMappings(r.Context())
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to load library")
		return
	}

	h.render(w, "library", map[string]any{
		"Title":   "Library",
		"Artists": artists,
	})
}

// ---------------------------------------------------------------------------
// Action handlers
// ---------------------------------------------------------------------------

// StartDownload kicks off an async album download and returns an HTMX partial
// confirming the request.
func (h *Handler) StartDownload(w http.ResponseWriter, r *http.Request) {
	albumIDStr := r.FormValue("album_id")
	quality := r.FormValue("quality")
	if quality == "" {
		quality = h.quality
	}

	albumID, err := strconv.ParseInt(albumIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid album ID", http.StatusBadRequest)
		return
	}

	// Fetch album info so we can show a meaningful status message.
	detail, err := h.hifi.GetAlbum(r.Context(), albumID)
	if err != nil {
		http.Error(w, "Failed to fetch album", http.StatusInternalServerError)
		return
	}

	h.downloader.DownloadAsync(r.Context(), downloader.Request{
		TidalAlbumID: albumID,
		Quality:      quality,
	})

	h.renderPartial(w, "download_status", "download_status", map[string]any{
		"ArtistName": detail.Artist.Name,
		"AlbumTitle": detail.Title,
	})
}

// StartScan triggers a library scan and returns the result as JSON.
func (h *Handler) StartScan(w http.ResponseWriter, r *http.Request) {
	result, err := h.scanner.Scan(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{
		"artists_found":   result.ArtistsFound,
		"albums_found":    result.AlbumsFound,
		"artists_matched": result.ArtistsMatched,
		"errors":          len(result.Errors),
	})
}
