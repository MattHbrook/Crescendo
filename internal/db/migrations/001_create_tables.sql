CREATE TABLE IF NOT EXISTS artist_mapping (
    id INTEGER PRIMARY KEY,
    folder_name TEXT UNIQUE NOT NULL,
    tidal_id INTEGER,
    tidal_name TEXT,
    picture_url TEXT,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS library_albums (
    id INTEGER PRIMARY KEY,
    artist_folder TEXT NOT NULL,
    album_folder TEXT NOT NULL,
    track_count INTEGER DEFAULT 0,
    path TEXT NOT NULL,
    last_scanned DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(artist_folder, album_folder)
);

CREATE TABLE IF NOT EXISTS downloads (
    id INTEGER PRIMARY KEY,
    tidal_album_id INTEGER NOT NULL,
    artist_name TEXT NOT NULL,
    album_title TEXT NOT NULL,
    quality TEXT DEFAULT 'LOSSLESS',
    status TEXT DEFAULT 'queued',
    progress REAL DEFAULT 0,
    total_tracks INTEGER DEFAULT 0,
    completed_tracks INTEGER DEFAULT 0,
    error TEXT,
    output_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);
