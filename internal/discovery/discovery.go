package discovery

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/MattHbrook/Crescendo/internal/db"
	"github.com/MattHbrook/Crescendo/internal/hifi"
)

// SeedStore provides random artist seeds from the library.
type SeedStore interface {
	GetRandomArtistMappings(ctx context.Context, limit int) ([]db.ArtistMapping, error)
	IsAlbumDownloaded(ctx context.Context, tidalAlbumID int64) (bool, error)
}

// SimilarFinder fetches similar artists and albums from Tidal.
type SimilarFinder interface {
	GetSimilarArtists(ctx context.Context, id int64) ([]hifi.SimilarArtist, error)
	GetSimilarAlbums(ctx context.Context, id int64) ([]hifi.SimilarAlbum, error)
}

// Recommendation is a suggested album from Tidal that the user doesn't have yet.
type Recommendation struct {
	AlbumID     int64
	AlbumTitle  string
	ArtistName  string
	Cover       string // UUID
	ReleaseDate string
	SeedArtist  string // which library artist led to this recommendation
}

// Engine generates music recommendations from the user's library.
type Engine struct {
	store  SeedStore
	finder SimilarFinder
	logger *log.Logger
}

// NewEngine creates a discovery engine backed by the given store and finder.
func NewEngine(store SeedStore, finder SimilarFinder) *Engine {
	return &Engine{
		store:  store,
		finder: finder,
		logger: log.New(os.Stderr, "[discovery] ", log.LstdFlags),
	}
}

// Discover picks random seed artists from the library, fetches similar albums
// from Tidal, and returns up to maxResults recommendations the user hasn't
// downloaded yet.
func (e *Engine) Discover(ctx context.Context, seedCount, maxResults int) ([]Recommendation, error) {
	seeds, err := e.store.GetRandomArtistMappings(ctx, seedCount)
	if err != nil {
		return nil, fmt.Errorf("discovery: fetching seed artists: %w", err)
	}

	seen := make(map[int64]bool)
	var recs []Recommendation

	for _, seed := range seeds {
		if seed.TidalID == nil {
			continue
		}

		seedName := seed.FolderName
		if seed.TidalName != nil {
			seedName = *seed.TidalName
		}

		albums, err := e.finder.GetSimilarAlbums(ctx, *seed.TidalID)
		if err != nil {
			e.logger.Printf("similar albums for %s: %v", seedName, err)
			continue
		}

		for _, album := range albums {
			if seen[album.ID] {
				continue
			}
			seen[album.ID] = true

			downloaded, err := e.store.IsAlbumDownloaded(ctx, album.ID)
			if err != nil {
				e.logger.Printf("checking download status for album %d: %v", album.ID, err)
				continue
			}
			if downloaded {
				continue
			}

			artistName := ""
			if len(album.Artists) > 0 {
				artistName = album.Artists[0].Name
			}

			recs = append(recs, Recommendation{
				AlbumID:     album.ID,
				AlbumTitle:  album.Title,
				ArtistName:  artistName,
				Cover:       album.Cover,
				ReleaseDate: album.ReleaseDate,
				SeedArtist:  seedName,
			})
		}
	}

	if len(recs) > maxResults {
		recs = recs[:maxResults]
	}

	return recs, nil
}
