package library

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/MattHbrook/Crescendo/internal/db"
	"github.com/MattHbrook/Crescendo/internal/hifi"
)

// ArtistStore is the subset of db.Store needed by the scanner.
type ArtistStore interface {
	GetArtistMapping(ctx context.Context, folderName string) (*db.ArtistMapping, error)
	UpsertArtistMapping(ctx context.Context, folderName string, tidalID int64, tidalName, pictureURL string) error
	UpsertLibraryAlbum(ctx context.Context, artistFolder, albumFolder string, trackCount int, path string) error
	DeleteLibraryAlbumsByArtist(ctx context.Context, artistFolder string) error
}

// ArtistSearcher is the subset of hifi.Client needed by the scanner.
type ArtistSearcher interface {
	SearchArtists(ctx context.Context, query string, limit, offset int) (*hifi.SearchResult[hifi.Artist], error)
}

// Scanner walks a music directory and populates the database with artist and
// album information, resolving artist identities via the HiFi search API.
type Scanner struct {
	musicPath string
	store     ArtistStore
	searcher  ArtistSearcher
	logger    *log.Logger
}

// NewScanner creates a Scanner that will walk musicPath and use the provided
// store and searcher for persistence and artist resolution.
func NewScanner(musicPath string, store ArtistStore, searcher ArtistSearcher) *Scanner {
	return &Scanner{
		musicPath: musicPath,
		store:     store,
		searcher:  searcher,
		logger:    log.New(os.Stderr, "[library] ", log.LstdFlags),
	}
}

// ScanResult holds aggregate statistics from a library scan.
type ScanResult struct {
	ArtistsFound   int
	AlbumsFound    int
	ArtistsMatched int // successfully resolved to Tidal ID
	Errors         []string
}

// Scan reads the top-level music directory and processes every artist folder
// it finds. It returns an error only if the music directory itself cannot be
// read; all per-artist and per-album errors are collected in ScanResult.Errors.
func (s *Scanner) Scan(ctx context.Context) (*ScanResult, error) {
	entries, err := os.ReadDir(s.musicPath)
	if err != nil {
		return nil, fmt.Errorf("reading music directory %s: %w", s.musicPath, err)
	}

	result := &ScanResult{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "Playlists" {
			continue
		}
		s.scanArtist(ctx, entry.Name(), result)
	}

	return result, nil
}

// scanArtist processes a single artist folder: it discovers albums, counts
// FLAC tracks in each, persists album records, and attempts to resolve the
// artist to a Tidal ID.
func (s *Scanner) scanArtist(ctx context.Context, artistFolder string, result *ScanResult) {
	result.ArtistsFound++

	artistPath := filepath.Join(s.musicPath, artistFolder)
	entries, err := os.ReadDir(artistPath)
	if err != nil {
		msg := fmt.Sprintf("reading artist directory %s: %v", artistPath, err)
		s.logger.Println(msg)
		result.Errors = append(result.Errors, msg)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		albumFolder := entry.Name()
		albumPath := filepath.Join(artistPath, albumFolder)

		trackCount, err := countFLACFiles(albumPath)
		if err != nil {
			msg := fmt.Sprintf("counting tracks in %s: %v", albumPath, err)
			s.logger.Println(msg)
			result.Errors = append(result.Errors, msg)
			continue
		}

		if trackCount > 0 {
			if err := s.store.UpsertLibraryAlbum(ctx, artistFolder, albumFolder, trackCount, albumPath); err != nil {
				msg := fmt.Sprintf("upserting album %s/%s: %v", artistFolder, albumFolder, err)
				s.logger.Println(msg)
				result.Errors = append(result.Errors, msg)
				continue
			}
			result.AlbumsFound++
		}
	}

	// Attempt to resolve the artist's Tidal ID if not already mapped.
	mapping, err := s.store.GetArtistMapping(ctx, artistFolder)
	if err != nil {
		msg := fmt.Sprintf("getting artist mapping for %s: %v", artistFolder, err)
		s.logger.Println(msg)
		result.Errors = append(result.Errors, msg)
		return
	}

	if mapping != nil && mapping.TidalID != nil {
		return // already resolved
	}

	s.resolveArtist(ctx, artistFolder, result)
}

// resolveArtist searches the HiFi API for the given artist folder name and,
// if a match is found, persists the mapping. Errors are logged and collected
// rather than propagated so the scan can continue.
func (s *Scanner) resolveArtist(ctx context.Context, artistFolder string, result *ScanResult) {
	searchResult, err := s.searcher.SearchArtists(ctx, artistFolder, 1, 0)
	if err != nil {
		msg := fmt.Sprintf("searching for artist %s: %v", artistFolder, err)
		s.logger.Println(msg)
		result.Errors = append(result.Errors, msg)
		return
	}

	if len(searchResult.Items) == 0 {
		msg := fmt.Sprintf("no Tidal match found for artist %s", artistFolder)
		s.logger.Println(msg)
		result.Errors = append(result.Errors, msg)
		return
	}

	artist := searchResult.Items[0]
	if err := s.store.UpsertArtistMapping(ctx, artistFolder, artist.ID, artist.Name, artist.Picture); err != nil {
		msg := fmt.Sprintf("upserting artist mapping for %s: %v", artistFolder, err)
		s.logger.Println(msg)
		result.Errors = append(result.Errors, msg)
		return
	}

	result.ArtistsMatched++
}

// countFLACFiles returns the number of .flac files (case-insensitive) in the
// given directory, ignoring macOS resource-fork artifacts (files starting
// with "._").
func countFLACFiles(dirPath string) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "._") {
			continue
		}
		if strings.EqualFold(filepath.Ext(name), ".flac") {
			count++
		}
	}

	return count, nil
}
