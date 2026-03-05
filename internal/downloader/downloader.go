package downloader

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/MattHbrook/Crescendo/internal/hifi"
	"github.com/MattHbrook/Crescendo/internal/library"
	"github.com/MattHbrook/Crescendo/internal/manifest"
)

// TrackPlayer fetches playback info for a track.
type TrackPlayer interface {
	GetTrackPlayback(ctx context.Context, id int64, quality string) (*hifi.Playback, error)
}

// AlbumFetcher fetches album details including track list.
type AlbumFetcher interface {
	GetAlbum(ctx context.Context, id int64) (*hifi.AlbumDetail, error)
}

// DownloadStore persists download state.
type DownloadStore interface {
	CreateDownload(ctx context.Context, tidalAlbumID int64, artistName, albumTitle, quality string, totalTracks int) (int64, error)
	UpdateDownloadProgress(ctx context.Context, id int64, completedTracks int, progress float64) error
	CompleteDownload(ctx context.Context, id int64, outputPath string) error
	FailDownload(ctx context.Context, id int64, errMsg string) error
}

// Request is a request to download an album.
type Request struct {
	TidalAlbumID int64
	Quality      string // "LOSSLESS" or "HI_RES_LOSSLESS"
}

// Downloader manages concurrent album downloads.
type Downloader struct {
	musicPath string
	player    TrackPlayer
	albums    AlbumFetcher
	store     DownloadStore
	sem       chan struct{} // limits concurrency
	logger    *log.Logger
}

// New creates a Downloader with the given concurrency limit and dependencies.
func New(musicPath string, maxConcurrent int, player TrackPlayer, albums AlbumFetcher, store DownloadStore) *Downloader {
	return &Downloader{
		musicPath: musicPath,
		player:    player,
		albums:    albums,
		store:     store,
		sem:       make(chan struct{}, maxConcurrent),
		logger:    log.New(os.Stderr, "[downloader] ", log.LstdFlags),
	}
}

// Download orchestrates the full download of an album's tracks.
func (d *Downloader) Download(ctx context.Context, req Request) error {
	album, err := d.albums.GetAlbum(ctx, req.TidalAlbumID)
	if err != nil {
		return fmt.Errorf("downloader: fetching album %d: %w", req.TidalAlbumID, err)
	}

	outputDir := library.AlbumDir(d.musicPath, album.Artist.Name, album.Title)

	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("downloader: creating album directory: %w", err)
	}

	downloadID, err := d.store.CreateDownload(ctx, req.TidalAlbumID, album.Artist.Name, album.Title, req.Quality, len(album.Tracks))
	if err != nil {
		return fmt.Errorf("downloader: creating download record: %w", err)
	}

	for i, track := range album.Tracks {
		playback, err := d.player.GetTrackPlayback(ctx, track.ID, req.Quality)
		if err != nil {
			_ = d.store.FailDownload(ctx, downloadID, err.Error())
			return fmt.Errorf("downloader: getting playback for track %d: %w", track.ID, err)
		}

		manifestResult, err := manifest.Decode(playback.ManifestMimeType, playback.Manifest)
		if err != nil {
			_ = d.store.FailDownload(ctx, downloadID, err.Error())
			return fmt.Errorf("downloader: decoding manifest for track %d: %w", track.ID, err)
		}

		trackPath := library.TrackPath(d.musicPath, album.Artist.Name, album.Title, track.TrackNumber, track.Title)

		if err := d.downloadTrack(ctx, manifestResult, trackPath); err != nil {
			_ = d.store.FailDownload(ctx, downloadID, err.Error())
			return fmt.Errorf("downloader: downloading track %d (%s): %w", track.ID, track.Title, err)
		}

		progress := float64(i+1) / float64(len(album.Tracks)) * 100
		if err := d.store.UpdateDownloadProgress(ctx, downloadID, i+1, progress); err != nil {
			_ = d.store.FailDownload(ctx, downloadID, err.Error())
			return fmt.Errorf("downloader: updating progress: %w", err)
		}
	}

	if err := d.store.CompleteDownload(ctx, downloadID, outputDir); err != nil {
		return fmt.Errorf("downloader: completing download record: %w", err)
	}

	return nil
}

// DownloadAsync acquires a semaphore slot and runs Download in a goroutine.
// Errors are logged but not returned (fire-and-forget).
func (d *Downloader) DownloadAsync(ctx context.Context, req Request) {
	d.sem <- struct{}{}
	go func() {
		defer func() { <-d.sem }()
		if err := d.Download(ctx, req); err != nil {
			d.logger.Printf("download failed for album %d: %v", req.TidalAlbumID, err)
		}
	}()
}

// downloadTrack downloads a single track, choosing the strategy based on codec.
func (d *Downloader) downloadTrack(ctx context.Context, m *manifest.Result, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return fmt.Errorf("creating track directory: %w", err)
	}

	if m.Codecs == "flac" {
		return d.downloadDirect(ctx, m.URLs[0], outputPath)
	}
	return d.downloadAndRemux(ctx, m.URLs[0], outputPath)
}

// downloadDirect streams the audio file directly to disk (used for FLAC/BTS).
func (d *Downloader) downloadDirect(ctx context.Context, url, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating HTTP request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // URLs come from Tidal API, not user input
	if err != nil {
		return fmt.Errorf("downloading track: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(outputPath) //nolint:gosec // path built by library.TrackPath, not user input
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing track data: %w", err)
	}

	return nil
}

// downloadAndRemux downloads a DASH stream to a temp file and remuxes it to
// FLAC with ffmpeg.
func (d *Downloader) downloadAndRemux(ctx context.Context, url, outputPath string) error {
	tmpPath := outputPath + ".mp4"

	// Download the raw stream to a temp file.
	if err := d.downloadDirect(ctx, url, tmpPath); err != nil {
		return fmt.Errorf("downloading DASH stream: %w", err)
	}

	// Remux with ffmpeg.
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", tmpPath, "-c:a", "flac", "-y", outputPath) //nolint:gosec // args built from sanitized paths
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg remux failed: %w\noutput: %s", err, string(output))
	}

	// Best-effort cleanup of the temp file.
	_ = os.Remove(tmpPath)

	return nil
}
