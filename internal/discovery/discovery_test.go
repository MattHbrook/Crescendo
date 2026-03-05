package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/MattHbrook/Crescendo/internal/db"
	"github.com/MattHbrook/Crescendo/internal/hifi"
)

// --- mocks ---

type mockSeedStore struct {
	mappings        []db.ArtistMapping
	downloaded      map[int64]bool // albumID -> downloaded
	getMappingsErr  error
	isDownloadedErr error
}

func (m *mockSeedStore) GetRandomArtistMappings(_ context.Context, _ int) ([]db.ArtistMapping, error) {
	if m.getMappingsErr != nil {
		return nil, m.getMappingsErr
	}
	return m.mappings, nil
}

func (m *mockSeedStore) IsAlbumDownloaded(_ context.Context, tidalAlbumID int64) (bool, error) {
	if m.isDownloadedErr != nil {
		return false, m.isDownloadedErr
	}
	return m.downloaded[tidalAlbumID], nil
}

type mockSimilarFinder struct {
	albums  map[int64][]hifi.SimilarAlbum // artistTidalID -> similar albums
	findErr error
}

func (m *mockSimilarFinder) GetSimilarArtists(_ context.Context, _ int64) ([]hifi.SimilarArtist, error) {
	return nil, nil
}

func (m *mockSimilarFinder) GetSimilarAlbums(_ context.Context, id int64) ([]hifi.SimilarAlbum, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.albums[id], nil
}

// --- helpers ---

func ptr[T any](v T) *T { return &v }

// --- tests ---

func TestDiscover_Success(t *testing.T) {
	store := &mockSeedStore{
		mappings: []db.ArtistMapping{
			{ID: 1, FolderName: "artist_a", TidalID: ptr(int64(100)), TidalName: ptr("Artist A")},
			{ID: 2, FolderName: "artist_b", TidalID: ptr(int64(200)), TidalName: ptr("Artist B")},
		},
		downloaded: map[int64]bool{
			1001: true, // Album Y is already downloaded
		},
	}

	finder := &mockSimilarFinder{
		albums: map[int64][]hifi.SimilarAlbum{
			100: {
				{ID: 1000, Title: "Album X", Cover: "cover-x", ReleaseDate: "2024-01-01", Artists: []hifi.ArtistRef{{ID: 10, Name: "Other Artist"}}},
				{ID: 1001, Title: "Album Y", Cover: "cover-y", ReleaseDate: "2024-02-01", Artists: []hifi.ArtistRef{{ID: 11, Name: "Another"}}},
			},
			200: {
				{ID: 1002, Title: "Album Z", Cover: "cover-z", ReleaseDate: "2024-03-01", Artists: []hifi.ArtistRef{{ID: 12, Name: "Someone"}}},
				{ID: 1000, Title: "Album X", Cover: "cover-x", ReleaseDate: "2024-01-01", Artists: []hifi.ArtistRef{{ID: 10, Name: "Other Artist"}}}, // duplicate
			},
		},
	}

	eng := NewEngine(store, finder)
	recs, err := eng.Discover(context.Background(), 2, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(recs) != 2 {
		t.Fatalf("expected 2 recommendations, got %d", len(recs))
	}

	// Album X from seed Artist A
	if recs[0].AlbumID != 1000 {
		t.Errorf("recs[0].AlbumID = %d, want 1000", recs[0].AlbumID)
	}
	if recs[0].AlbumTitle != "Album X" {
		t.Errorf("recs[0].AlbumTitle = %q, want %q", recs[0].AlbumTitle, "Album X")
	}
	if recs[0].ArtistName != "Other Artist" {
		t.Errorf("recs[0].ArtistName = %q, want %q", recs[0].ArtistName, "Other Artist")
	}
	if recs[0].SeedArtist != "Artist A" {
		t.Errorf("recs[0].SeedArtist = %q, want %q", recs[0].SeedArtist, "Artist A")
	}

	// Album Z from seed Artist B
	if recs[1].AlbumID != 1002 {
		t.Errorf("recs[1].AlbumID = %d, want 1002", recs[1].AlbumID)
	}
	if recs[1].AlbumTitle != "Album Z" {
		t.Errorf("recs[1].AlbumTitle = %q, want %q", recs[1].AlbumTitle, "Album Z")
	}
	if recs[1].ArtistName != "Someone" {
		t.Errorf("recs[1].ArtistName = %q, want %q", recs[1].ArtistName, "Someone")
	}
	if recs[1].SeedArtist != "Artist B" {
		t.Errorf("recs[1].SeedArtist = %q, want %q", recs[1].SeedArtist, "Artist B")
	}
}

func TestDiscover_NoSeeds(t *testing.T) {
	store := &mockSeedStore{
		mappings:   []db.ArtistMapping{},
		downloaded: map[int64]bool{},
	}
	finder := &mockSimilarFinder{
		albums: map[int64][]hifi.SimilarAlbum{},
	}

	eng := NewEngine(store, finder)
	recs, err := eng.Discover(context.Background(), 5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 recommendations, got %d", len(recs))
	}
}

func TestDiscover_SeedStoreError(t *testing.T) {
	store := &mockSeedStore{
		getMappingsErr: errors.New("db connection lost"),
	}
	finder := &mockSimilarFinder{}

	eng := NewEngine(store, finder)
	_, err := eng.Discover(context.Background(), 5, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDiscover_SimilarAlbumsError(t *testing.T) {
	store := &mockSeedStore{
		mappings: []db.ArtistMapping{
			{ID: 1, FolderName: "artist_a", TidalID: ptr(int64(100)), TidalName: ptr("Artist A")},
		},
		downloaded: map[int64]bool{},
	}
	finder := &mockSimilarFinder{
		findErr: errors.New("tidal API timeout"),
	}

	eng := NewEngine(store, finder)
	recs, err := eng.Discover(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 recommendations, got %d", len(recs))
	}
}

func TestDiscover_MaxResultsTruncation(t *testing.T) {
	// 3 seeds, each with 5 unique albums = 15 total, maxResults = 5.
	store := &mockSeedStore{
		mappings: []db.ArtistMapping{
			{ID: 1, FolderName: "a1", TidalID: ptr(int64(100)), TidalName: ptr("Seed 1")},
			{ID: 2, FolderName: "a2", TidalID: ptr(int64(200)), TidalName: ptr("Seed 2")},
			{ID: 3, FolderName: "a3", TidalID: ptr(int64(300)), TidalName: ptr("Seed 3")},
		},
		downloaded: map[int64]bool{},
	}

	albums := map[int64][]hifi.SimilarAlbum{}
	id := int64(1000)
	for _, seedID := range []int64{100, 200, 300} {
		var batch []hifi.SimilarAlbum
		for i := 0; i < 5; i++ {
			batch = append(batch, hifi.SimilarAlbum{
				ID:      id,
				Title:   "Album",
				Artists: []hifi.ArtistRef{{ID: 1, Name: "Art"}},
			})
			id++
		}
		albums[seedID] = batch
	}

	finder := &mockSimilarFinder{albums: albums}
	eng := NewEngine(store, finder)

	recs, err := eng.Discover(context.Background(), 3, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 5 {
		t.Fatalf("expected 5 recommendations, got %d", len(recs))
	}
}

func TestDiscover_SkipsNilTidalID(t *testing.T) {
	store := &mockSeedStore{
		mappings: []db.ArtistMapping{
			{ID: 1, FolderName: "unmapped_artist", TidalID: nil},
		},
		downloaded: map[int64]bool{},
	}
	finder := &mockSimilarFinder{
		albums: map[int64][]hifi.SimilarAlbum{},
	}

	eng := NewEngine(store, finder)
	recs, err := eng.Discover(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 recommendations, got %d", len(recs))
	}
}
