package db

import (
	"context"
	"database/sql"
	"fmt"
)

// ArtistMapping represents a row in the artist_mapping table.
type ArtistMapping struct {
	ID          int64
	FolderName  string
	TidalID     *int64
	TidalName   *string
	PictureURL  *string
	LastUpdated string
}

// LibraryAlbum represents a row in the library_albums table.
type LibraryAlbum struct {
	ID           int64
	ArtistFolder string
	AlbumFolder  string
	TrackCount   int
	Path         string
	LastScanned  string
}

// Download represents a row in the downloads table.
type Download struct {
	ID              int64
	TidalAlbumID    int64
	ArtistName      string
	AlbumTitle      string
	Quality         string
	Status          string
	Progress        float64
	TotalTracks     int
	CompletedTracks int
	Error           *string
	OutputPath      *string
	CreatedAt       string
	CompletedAt     *string
}

// Store provides query methods over the database.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ---------------------------------------------------------------------------
// Artist Mapping
// ---------------------------------------------------------------------------

// UpsertArtistMapping inserts or replaces an artist mapping entry.
func (s *Store) UpsertArtistMapping(ctx context.Context, folderName string, tidalID int64, tidalName, pictureURL string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO artist_mapping (folder_name, tidal_id, tidal_name, picture_url, last_updated)
		VALUES (?, ?, ?, ?, datetime('now'))`,
		folderName, tidalID, tidalName, pictureURL,
	)
	if err != nil {
		return fmt.Errorf("store: upsert artist mapping %q: %w", folderName, err)
	}
	return nil
}

// GetArtistMapping returns the mapping for the given folder name, or nil if
// no row exists.
func (s *Store) GetArtistMapping(ctx context.Context, folderName string) (*ArtistMapping, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, folder_name, tidal_id, tidal_name, picture_url, last_updated
		FROM artist_mapping
		WHERE folder_name = ?`,
		folderName,
	)

	var m ArtistMapping
	var tidalID sql.NullInt64
	var tidalName, pictureURL sql.NullString

	err := row.Scan(&m.ID, &m.FolderName, &tidalID, &tidalName, &pictureURL, &m.LastUpdated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get artist mapping %q: %w", folderName, err)
	}

	if tidalID.Valid {
		m.TidalID = &tidalID.Int64
	}
	if tidalName.Valid {
		m.TidalName = &tidalName.String
	}
	if pictureURL.Valid {
		m.PictureURL = &pictureURL.String
	}

	return &m, nil
}

// ListArtistMappings returns all artist mappings that have a non-null tidal_id,
// ordered by folder_name.
func (s *Store) ListArtistMappings(ctx context.Context) ([]ArtistMapping, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, folder_name, tidal_id, tidal_name, picture_url, last_updated
		FROM artist_mapping
		WHERE tidal_id IS NOT NULL
		ORDER BY folder_name`)
	if err != nil {
		return nil, fmt.Errorf("store: list artist mappings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var mappings []ArtistMapping
	for rows.Next() {
		var m ArtistMapping
		var tidalID sql.NullInt64
		var tidalName, pictureURL sql.NullString

		if err := rows.Scan(&m.ID, &m.FolderName, &tidalID, &tidalName, &pictureURL, &m.LastUpdated); err != nil {
			return nil, fmt.Errorf("store: list artist mappings scan: %w", err)
		}

		if tidalID.Valid {
			m.TidalID = &tidalID.Int64
		}
		if tidalName.Valid {
			m.TidalName = &tidalName.String
		}
		if pictureURL.Valid {
			m.PictureURL = &pictureURL.String
		}

		mappings = append(mappings, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list artist mappings rows: %w", err)
	}

	return mappings, nil
}

// GetRandomArtistMappings returns up to limit random artist mappings that have
// a non-null tidal_id.
func (s *Store) GetRandomArtistMappings(ctx context.Context, limit int) ([]ArtistMapping, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, folder_name, tidal_id, tidal_name, picture_url, last_updated
		FROM artist_mapping
		WHERE tidal_id IS NOT NULL
		ORDER BY RANDOM()
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get random artist mappings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var mappings []ArtistMapping
	for rows.Next() {
		var m ArtistMapping
		var tidalID sql.NullInt64
		var tidalName, pictureURL sql.NullString

		if err := rows.Scan(&m.ID, &m.FolderName, &tidalID, &tidalName, &pictureURL, &m.LastUpdated); err != nil {
			return nil, fmt.Errorf("store: get random artist mappings scan: %w", err)
		}

		if tidalID.Valid {
			m.TidalID = &tidalID.Int64
		}
		if tidalName.Valid {
			m.TidalName = &tidalName.String
		}
		if pictureURL.Valid {
			m.PictureURL = &pictureURL.String
		}

		mappings = append(mappings, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: get random artist mappings rows: %w", err)
	}

	return mappings, nil
}

// ---------------------------------------------------------------------------
// Library Albums
// ---------------------------------------------------------------------------

// UpsertLibraryAlbum inserts or replaces a library album entry.
func (s *Store) UpsertLibraryAlbum(ctx context.Context, artistFolder, albumFolder string, trackCount int, path string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO library_albums (artist_folder, album_folder, track_count, path, last_scanned)
		VALUES (?, ?, ?, ?, datetime('now'))`,
		artistFolder, albumFolder, trackCount, path,
	)
	if err != nil {
		return fmt.Errorf("store: upsert library album %q/%q: %w", artistFolder, albumFolder, err)
	}
	return nil
}

// IsAlbumInLibrary returns true if an album with the given artist and album
// folder names exists in the library.
func (s *Store) IsAlbumInLibrary(ctx context.Context, artistFolder, albumFolder string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM library_albums
			WHERE artist_folder = ? AND album_folder = ?
		)`,
		artistFolder, albumFolder,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: check album in library %q/%q: %w", artistFolder, albumFolder, err)
	}
	return exists, nil
}

// ListLibraryArtists returns all distinct artist folder names, ordered
// alphabetically.
func (s *Store) ListLibraryArtists(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT artist_folder
		FROM library_albums
		ORDER BY artist_folder`)
	if err != nil {
		return nil, fmt.Errorf("store: list library artists: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var artists []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store: list library artists scan: %w", err)
		}
		artists = append(artists, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list library artists rows: %w", err)
	}

	return artists, nil
}

// ListAlbumsForArtist returns all library albums for the given artist folder,
// ordered by album folder name.
func (s *Store) ListAlbumsForArtist(ctx context.Context, artistFolder string) ([]LibraryAlbum, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, artist_folder, album_folder, track_count, path, last_scanned
		FROM library_albums
		WHERE artist_folder = ?
		ORDER BY album_folder`,
		artistFolder,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list albums for artist %q: %w", artistFolder, err)
	}
	defer func() { _ = rows.Close() }()

	var albums []LibraryAlbum
	for rows.Next() {
		var a LibraryAlbum
		if err := rows.Scan(&a.ID, &a.ArtistFolder, &a.AlbumFolder, &a.TrackCount, &a.Path, &a.LastScanned); err != nil {
			return nil, fmt.Errorf("store: list albums for artist %q scan: %w", artistFolder, err)
		}
		albums = append(albums, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list albums for artist %q rows: %w", artistFolder, err)
	}

	return albums, nil
}

// DeleteLibraryAlbumsByArtist removes all library album entries for the given
// artist folder, supporting a full rescan.
func (s *Store) DeleteLibraryAlbumsByArtist(ctx context.Context, artistFolder string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM library_albums
		WHERE artist_folder = ?`,
		artistFolder,
	)
	if err != nil {
		return fmt.Errorf("store: delete library albums for artist %q: %w", artistFolder, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Downloads
// ---------------------------------------------------------------------------

// CreateDownload inserts a new download record and returns its ID.
func (s *Store) CreateDownload(ctx context.Context, tidalAlbumID int64, artistName, albumTitle, quality string, totalTracks int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO downloads (tidal_album_id, artist_name, album_title, quality, total_tracks, status, progress, completed_tracks, created_at)
		VALUES (?, ?, ?, ?, ?, 'queued', 0, 0, datetime('now'))`,
		tidalAlbumID, artistName, albumTitle, quality, totalTracks,
	)
	if err != nil {
		return 0, fmt.Errorf("store: create download for album %d: %w", tidalAlbumID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("store: create download last insert id: %w", err)
	}

	return id, nil
}

// UpdateDownloadProgress updates the completed track count and progress
// percentage of an in-flight download.
func (s *Store) UpdateDownloadProgress(ctx context.Context, id int64, completedTracks int, progress float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE downloads
		SET completed_tracks = ?, progress = ?
		WHERE id = ?`,
		completedTracks, progress, id,
	)
	if err != nil {
		return fmt.Errorf("store: update download progress %d: %w", id, err)
	}
	return nil
}

// CompleteDownload marks a download as successfully complete.
func (s *Store) CompleteDownload(ctx context.Context, id int64, outputPath string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE downloads
		SET status = 'complete', output_path = ?, completed_at = datetime('now'), progress = 100
		WHERE id = ?`,
		outputPath, id,
	)
	if err != nil {
		return fmt.Errorf("store: complete download %d: %w", id, err)
	}
	return nil
}

// FailDownload marks a download as failed with the given error message.
func (s *Store) FailDownload(ctx context.Context, id int64, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE downloads
		SET status = 'failed', error = ?
		WHERE id = ?`,
		errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("store: fail download %d: %w", id, err)
	}
	return nil
}

// GetActiveDownloads returns all downloads with a queued or downloading status,
// ordered by creation time.
func (s *Store) GetActiveDownloads(ctx context.Context) ([]Download, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tidal_album_id, artist_name, album_title, quality, status,
		       progress, total_tracks, completed_tracks, error, output_path,
		       created_at, completed_at
		FROM downloads
		WHERE status IN ('queued', 'downloading')
		ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("store: get active downloads: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDownloads(rows)
}

// GetDownloadHistory returns the most recent downloads up to the given limit,
// ordered by creation time descending.
func (s *Store) GetDownloadHistory(ctx context.Context, limit int) ([]Download, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tidal_album_id, artist_name, album_title, quality, status,
		       progress, total_tracks, completed_tracks, error, output_path,
		       created_at, completed_at
		FROM downloads
		ORDER BY created_at DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get download history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDownloads(rows)
}

// IsAlbumDownloaded returns true if a completed download exists for the given
// Tidal album ID.
func (s *Store) IsAlbumDownloaded(ctx context.Context, tidalAlbumID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM downloads
			WHERE tidal_album_id = ? AND status = 'complete'
		)`,
		tidalAlbumID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: check album downloaded %d: %w", tidalAlbumID, err)
	}
	return exists, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// scanDownloads scans all rows into a slice of Download values.
func scanDownloads(rows *sql.Rows) ([]Download, error) {
	var downloads []Download
	for rows.Next() {
		var d Download
		var errMsg, outputPath, completedAt sql.NullString

		if err := rows.Scan(
			&d.ID, &d.TidalAlbumID, &d.ArtistName, &d.AlbumTitle,
			&d.Quality, &d.Status, &d.Progress, &d.TotalTracks,
			&d.CompletedTracks, &errMsg, &outputPath,
			&d.CreatedAt, &completedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan download row: %w", err)
		}

		if errMsg.Valid {
			d.Error = &errMsg.String
		}
		if outputPath.Valid {
			d.OutputPath = &outputPath.String
		}
		if completedAt.Valid {
			d.CompletedAt = &completedAt.String
		}

		downloads = append(downloads, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: scan download rows: %w", err)
	}

	return downloads, nil
}
