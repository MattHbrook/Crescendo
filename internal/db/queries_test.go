package db

import (
	"context"
	"database/sql"
	"testing"
)

// newTestStore opens an isolated SQLite database, runs migrations, and returns
// a Store ready for testing. The database is automatically cleaned up when the
// test finishes.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	dir := t.TempDir()
	handle, err := Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { handle.Close() })

	if err := Migrate(handle); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewStore(handle)
}

// ---------------------------------------------------------------------------
// Artist Mapping
// ---------------------------------------------------------------------------

func TestUpsertAndGetArtistMapping(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.UpsertArtistMapping(ctx, "Pink Floyd", 1234, "Pink Floyd", "https://img.example.com/pf.jpg"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := store.GetArtistMapping(ctx, "Pink Floyd")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil mapping, got nil")
	}

	if got.FolderName != "Pink Floyd" {
		t.Errorf("FolderName = %q, want %q", got.FolderName, "Pink Floyd")
	}
	if got.TidalID == nil || *got.TidalID != 1234 {
		t.Errorf("TidalID = %v, want 1234", got.TidalID)
	}
	if got.TidalName == nil || *got.TidalName != "Pink Floyd" {
		t.Errorf("TidalName = %v, want %q", got.TidalName, "Pink Floyd")
	}
	if got.PictureURL == nil || *got.PictureURL != "https://img.example.com/pf.jpg" {
		t.Errorf("PictureURL = %v, want %q", got.PictureURL, "https://img.example.com/pf.jpg")
	}
	if got.LastUpdated == "" {
		t.Error("LastUpdated is empty")
	}
	if got.ID == 0 {
		t.Error("ID is 0, expected auto-assigned primary key")
	}
}

func TestGetArtistMapping_not_found(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	got, err := store.GetArtistMapping(ctx, "nonexistent-folder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil mapping for missing folder, got %+v", got)
	}
}

func TestListArtistMappings(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert two mappings with tidal_id set.
	if err := store.UpsertArtistMapping(ctx, "Beatles", 100, "The Beatles", "https://img.example.com/beatles.jpg"); err != nil {
		t.Fatalf("upsert Beatles: %v", err)
	}
	if err := store.UpsertArtistMapping(ctx, "Zeppelin", 200, "Led Zeppelin", "https://img.example.com/zeppelin.jpg"); err != nil {
		t.Fatalf("upsert Zeppelin: %v", err)
	}

	// Insert a mapping with NULL tidal_id directly (UpsertArtistMapping
	// always sets a non-null tidal_id, so we use raw SQL).
	if _, err := store.db.ExecContext(ctx,
		`INSERT INTO artist_mapping (folder_name) VALUES (?)`, "Unknown Artist"); err != nil {
		t.Fatalf("insert null-tidal mapping: %v", err)
	}

	mappings, err := store.ListArtistMappings(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// ListArtistMappings filters out rows with NULL tidal_id.
	if len(mappings) != 2 {
		t.Fatalf("got %d mappings, want 2", len(mappings))
	}

	// Results are ordered by folder_name ASC.
	if mappings[0].FolderName != "Beatles" {
		t.Errorf("mappings[0].FolderName = %q, want %q", mappings[0].FolderName, "Beatles")
	}
	if mappings[1].FolderName != "Zeppelin" {
		t.Errorf("mappings[1].FolderName = %q, want %q", mappings[1].FolderName, "Zeppelin")
	}
}

func TestGetRandomArtistMappings(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := int64(1); i <= 5; i++ {
		if err := store.UpsertArtistMapping(ctx, "Artist"+string(rune('A'-1+i)), i*10, "Name", "https://img.example.com/pic.jpg"); err != nil {
			t.Fatalf("upsert artist %d: %v", i, err)
		}
	}

	got, err := store.GetRandomArtistMappings(ctx, 2)
	if err != nil {
		t.Fatalf("get random: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d mappings, want 2", len(got))
	}

	// Verify each returned mapping has a valid tidal_id.
	for i, m := range got {
		if m.TidalID == nil {
			t.Errorf("mappings[%d].TidalID is nil", i)
		}
		if m.FolderName == "" {
			t.Errorf("mappings[%d].FolderName is empty", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Library Albums
// ---------------------------------------------------------------------------

func TestUpsertAndListLibraryAlbums(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.UpsertLibraryAlbum(ctx, "Pink Floyd", "The Wall", 26, "/music/Pink Floyd/The Wall"); err != nil {
		t.Fatalf("upsert The Wall: %v", err)
	}
	if err := store.UpsertLibraryAlbum(ctx, "Pink Floyd", "Animals", 5, "/music/Pink Floyd/Animals"); err != nil {
		t.Fatalf("upsert Animals: %v", err)
	}

	albums, err := store.ListAlbumsForArtist(ctx, "Pink Floyd")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("got %d albums, want 2", len(albums))
	}

	// Results are ordered by album_folder ASC.
	if albums[0].AlbumFolder != "Animals" {
		t.Errorf("albums[0].AlbumFolder = %q, want %q", albums[0].AlbumFolder, "Animals")
	}
	if albums[0].TrackCount != 5 {
		t.Errorf("albums[0].TrackCount = %d, want 5", albums[0].TrackCount)
	}
	if albums[1].AlbumFolder != "The Wall" {
		t.Errorf("albums[1].AlbumFolder = %q, want %q", albums[1].AlbumFolder, "The Wall")
	}
	if albums[1].TrackCount != 26 {
		t.Errorf("albums[1].TrackCount = %d, want 26", albums[1].TrackCount)
	}
}

func TestIsAlbumInLibrary(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Before insert, album should not be in library.
	exists, err := store.IsAlbumInLibrary(ctx, "Pink Floyd", "Wish You Were Here")
	if err != nil {
		t.Fatalf("check before insert: %v", err)
	}
	if exists {
		t.Error("expected false before insert, got true")
	}

	if err := store.UpsertLibraryAlbum(ctx, "Pink Floyd", "Wish You Were Here", 5, "/music/Pink Floyd/WYWH"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	exists, err = store.IsAlbumInLibrary(ctx, "Pink Floyd", "Wish You Were Here")
	if err != nil {
		t.Fatalf("check after insert: %v", err)
	}
	if !exists {
		t.Error("expected true after insert, got false")
	}

	// Unknown album should still return false.
	exists, err = store.IsAlbumInLibrary(ctx, "Pink Floyd", "Nonexistent")
	if err != nil {
		t.Fatalf("check unknown album: %v", err)
	}
	if exists {
		t.Error("expected false for unknown album, got true")
	}
}

func TestListLibraryArtists(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	data := []struct {
		artist, album, path string
	}{
		{"Pink Floyd", "Animals", "/music/Pink Floyd/Animals"},
		{"Pink Floyd", "The Wall", "/music/Pink Floyd/The Wall"},
		{"Led Zeppelin", "IV", "/music/Led Zeppelin/IV"},
		{"Beatles", "Abbey Road", "/music/Beatles/Abbey Road"},
	}
	for _, d := range data {
		if err := store.UpsertLibraryAlbum(ctx, d.artist, d.album, 10, d.path); err != nil {
			t.Fatalf("upsert %s/%s: %v", d.artist, d.album, err)
		}
	}

	artists, err := store.ListLibraryArtists(ctx)
	if err != nil {
		t.Fatalf("list artists: %v", err)
	}

	want := []string{"Beatles", "Led Zeppelin", "Pink Floyd"}
	if len(artists) != len(want) {
		t.Fatalf("got %d artists, want %d", len(artists), len(want))
	}
	for i, w := range want {
		if artists[i] != w {
			t.Errorf("artists[%d] = %q, want %q", i, artists[i], w)
		}
	}
}

func TestDeleteLibraryAlbumsByArtist(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.UpsertLibraryAlbum(ctx, "Pink Floyd", "Animals", 5, "/music/Pink Floyd/Animals"); err != nil {
		t.Fatalf("upsert Animals: %v", err)
	}
	if err := store.UpsertLibraryAlbum(ctx, "Pink Floyd", "The Wall", 26, "/music/Pink Floyd/The Wall"); err != nil {
		t.Fatalf("upsert The Wall: %v", err)
	}
	if err := store.UpsertLibraryAlbum(ctx, "Beatles", "Abbey Road", 17, "/music/Beatles/Abbey Road"); err != nil {
		t.Fatalf("upsert Abbey Road: %v", err)
	}

	if err := store.DeleteLibraryAlbumsByArtist(ctx, "Pink Floyd"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Pink Floyd albums should be gone.
	albums, err := store.ListAlbumsForArtist(ctx, "Pink Floyd")
	if err != nil {
		t.Fatalf("list Pink Floyd: %v", err)
	}
	if len(albums) != 0 {
		t.Errorf("expected 0 Pink Floyd albums after delete, got %d", len(albums))
	}

	// Beatles album should still exist.
	albums, err = store.ListAlbumsForArtist(ctx, "Beatles")
	if err != nil {
		t.Fatalf("list Beatles: %v", err)
	}
	if len(albums) != 1 {
		t.Errorf("expected 1 Beatles album, got %d", len(albums))
	}
}

// ---------------------------------------------------------------------------
// Downloads
// ---------------------------------------------------------------------------

func TestCreateAndGetDownloads(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.CreateDownload(ctx, 9999, "Pink Floyd", "The Wall", "LOSSLESS", 26)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero download ID")
	}

	active, err := store.GetActiveDownloads(ctx)
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("got %d active downloads, want 1", len(active))
	}

	d := active[0]
	if d.ID != id {
		t.Errorf("ID = %d, want %d", d.ID, id)
	}
	if d.TidalAlbumID != 9999 {
		t.Errorf("TidalAlbumID = %d, want 9999", d.TidalAlbumID)
	}
	if d.ArtistName != "Pink Floyd" {
		t.Errorf("ArtistName = %q, want %q", d.ArtistName, "Pink Floyd")
	}
	if d.AlbumTitle != "The Wall" {
		t.Errorf("AlbumTitle = %q, want %q", d.AlbumTitle, "The Wall")
	}
	if d.Quality != "LOSSLESS" {
		t.Errorf("Quality = %q, want %q", d.Quality, "LOSSLESS")
	}
	if d.Status != "queued" {
		t.Errorf("Status = %q, want %q", d.Status, "queued")
	}
	if d.Progress != 0 {
		t.Errorf("Progress = %f, want 0", d.Progress)
	}
	if d.TotalTracks != 26 {
		t.Errorf("TotalTracks = %d, want 26", d.TotalTracks)
	}
	if d.CompletedTracks != 0 {
		t.Errorf("CompletedTracks = %d, want 0", d.CompletedTracks)
	}
	if d.CreatedAt == "" {
		t.Error("CreatedAt is empty")
	}
}

func TestUpdateDownloadProgress(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.CreateDownload(ctx, 5555, "Beatles", "Abbey Road", "HI_RES", 17)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.UpdateDownloadProgress(ctx, id, 8, 47.06); err != nil {
		t.Fatalf("update progress: %v", err)
	}

	active, err := store.GetActiveDownloads(ctx)
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("got %d active, want 1", len(active))
	}

	d := active[0]
	if d.CompletedTracks != 8 {
		t.Errorf("CompletedTracks = %d, want 8", d.CompletedTracks)
	}
	if d.Progress != 47.06 {
		t.Errorf("Progress = %f, want 47.06", d.Progress)
	}
}

func TestCompleteDownload(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.CreateDownload(ctx, 7777, "Zeppelin", "IV", "LOSSLESS", 8)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.CompleteDownload(ctx, id, "/music/Zeppelin/IV"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Completed downloads should not appear in active list.
	active, err := store.GetActiveDownloads(ctx)
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active after complete, got %d", len(active))
	}

	// Should appear in history.
	history, err := store.GetDownloadHistory(ctx, 10)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("got %d history entries, want 1", len(history))
	}

	d := history[0]
	if d.Status != "complete" {
		t.Errorf("Status = %q, want %q", d.Status, "complete")
	}
	if d.Progress != 100 {
		t.Errorf("Progress = %f, want 100", d.Progress)
	}
	if d.OutputPath == nil || *d.OutputPath != "/music/Zeppelin/IV" {
		t.Errorf("OutputPath = %v, want %q", d.OutputPath, "/music/Zeppelin/IV")
	}
	if d.CompletedAt == nil || *d.CompletedAt == "" {
		t.Error("CompletedAt should be set after completion")
	}
}

func TestFailDownload(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.CreateDownload(ctx, 3333, "Artist", "Album", "LOSSLESS", 10)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.FailDownload(ctx, id, "network timeout"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	// Failed downloads should not appear in active list.
	active, err := store.GetActiveDownloads(ctx)
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active after failure, got %d", len(active))
	}

	// Should appear in history with error set.
	history, err := store.GetDownloadHistory(ctx, 10)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("got %d history entries, want 1", len(history))
	}

	d := history[0]
	if d.Status != "failed" {
		t.Errorf("Status = %q, want %q", d.Status, "failed")
	}
	if d.Error == nil || *d.Error != "network timeout" {
		t.Errorf("Error = %v, want %q", d.Error, "network timeout")
	}
}

func TestIsAlbumDownloaded(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	const tidalID int64 = 4444

	// Before any download.
	downloaded, err := store.IsAlbumDownloaded(ctx, tidalID)
	if err != nil {
		t.Fatalf("check before download: %v", err)
	}
	if downloaded {
		t.Error("expected false before any download, got true")
	}

	id, err := store.CreateDownload(ctx, tidalID, "Artist", "Album", "LOSSLESS", 10)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Queued download should not count as downloaded.
	downloaded, err = store.IsAlbumDownloaded(ctx, tidalID)
	if err != nil {
		t.Fatalf("check queued: %v", err)
	}
	if downloaded {
		t.Error("expected false for queued download, got true")
	}

	// Complete the download.
	if err := store.CompleteDownload(ctx, id, "/music/out"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	downloaded, err = store.IsAlbumDownloaded(ctx, tidalID)
	if err != nil {
		t.Fatalf("check after complete: %v", err)
	}
	if !downloaded {
		t.Error("expected true after complete, got false")
	}

	// Create and fail a different download; should not affect the completed one.
	id2, err := store.CreateDownload(ctx, 5555, "Other", "Other Album", "LOSSLESS", 5)
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if err := store.FailDownload(ctx, id2, "error"); err != nil {
		t.Fatalf("fail second: %v", err)
	}

	downloaded, err = store.IsAlbumDownloaded(ctx, 5555)
	if err != nil {
		t.Fatalf("check failed album: %v", err)
	}
	if downloaded {
		t.Error("expected false for failed download, got true")
	}
}

// suppress unused import warning for database/sql — the package is used by
// newTestStore via store.db field access in TestListArtistMappings.
var _ = (*sql.DB)(nil)
