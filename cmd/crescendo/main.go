package main

import (
	"io/fs"
	"log"
	"net/http"
	"time"

	crescendo "github.com/MattHbrook/Crescendo"
	"github.com/MattHbrook/Crescendo/internal/config"
	"github.com/MattHbrook/Crescendo/internal/db"
	"github.com/MattHbrook/Crescendo/internal/discovery"
	"github.com/MattHbrook/Crescendo/internal/downloader"
	"github.com/MattHbrook/Crescendo/internal/handlers"
	"github.com/MattHbrook/Crescendo/internal/hifi"
	"github.com/MattHbrook/Crescendo/internal/library"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Open(cfg.DataPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer func() { _ = database.Close() }()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	store := db.NewStore(database)
	hifiClient := hifi.NewClient(cfg.HiFiAPIURL)
	scanner := library.NewScanner(cfg.MusicPath, store, hifiClient)
	dl := downloader.New(cfg.MusicPath, cfg.MaxConcurrentDownloads, hifiClient, hifiClient, hifiClient, store)
	disc := discovery.NewEngine(store, hifiClient)

	templatesFS, err := fs.Sub(crescendo.Content, "templates")
	if err != nil {
		log.Fatalf("embedded templates: %v", err)
	}

	h, err := handlers.New(templatesFS, store, hifiClient, scanner, dl, disc, cfg.DefaultQuality)
	if err != nil {
		log.Fatalf("handlers: %v", err)
	}

	staticFS, err := fs.Sub(crescendo.Content, "static")
	if err != nil {
		log.Fatalf("embedded static: %v", err)
	}

	r := newRouter(h, staticFS)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Println("crescendo listening on", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func newRouter(h *handlers.Handler, staticFS fs.FS) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handleHealth)

	// Serve embedded static assets.
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	// Register all page and action routes.
	h.Routes(r)

	return r
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
