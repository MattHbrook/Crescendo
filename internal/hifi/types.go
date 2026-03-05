package hifi

// Artist represents a Tidal artist.
type Artist struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`    // UUID for image URL construction
	Popularity int    `json:"popularity"` // 0-100
}

// Album represents a Tidal album.
type Album struct {
	ID              int64         `json:"id"`
	Title           string        `json:"title"`
	Cover           string        `json:"cover"`       // UUID for image URL construction
	ReleaseDate     string        `json:"releaseDate"` // YYYY-MM-DD
	NumberOfTracks  int           `json:"numberOfTracks"`
	NumberOfVolumes int           `json:"numberOfVolumes"`
	AudioQuality    string        `json:"audioQuality"`
	Explicit        bool          `json:"explicit"`
	Artist          ArtistRef     `json:"artist"`
	Artists         []ArtistRef   `json:"artists"`
	MediaMetadata   MediaMetadata `json:"mediaMetadata"`
}

// ArtistRef is a lightweight artist reference embedded in albums and tracks.
type ArtistRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// MediaMetadata holds codec/quality tags from Tidal.
type MediaMetadata struct {
	Tags []string `json:"tags"` // e.g. ["LOSSLESS", "HIRES_LOSSLESS"]
}

// Track represents a Tidal track.
type Track struct {
	ID           int64       `json:"id"`
	Title        string      `json:"title"`
	Duration     int         `json:"duration"` // seconds
	TrackNumber  int         `json:"trackNumber"`
	VolumeNumber int         `json:"volumeNumber"`
	ReplayGain   float64     `json:"replayGain"`
	AudioQuality string      `json:"audioQuality"`
	Explicit     bool        `json:"explicit"`
	Artist       ArtistRef   `json:"artist"`
	Artists      []ArtistRef `json:"artists"`
}

// Playback holds track playback info including the manifest.
type Playback struct {
	TrackID          int64  `json:"trackId"`
	AudioQuality     string `json:"audioQuality"`
	ManifestMimeType string `json:"manifestMimeType"`
	Manifest         string `json:"manifest"` // base64 encoded
	BitDepth         int    `json:"bitDepth"`
	SampleRate       int    `json:"sampleRate"`
}

// Cover holds album cover art URLs at different sizes.
type Cover struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	URL1280 string `json:"1280"`
	URL640  string `json:"640"`
	URL80   string `json:"80"`
}

// SimilarArtist from the V2 API (popularity is float 0-1).
type SimilarArtist struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Picture    string  `json:"picture"`
	Popularity float64 `json:"popularity"`
}

// SimilarAlbum from the V2 API.
type SimilarAlbum struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title"`
	Cover       string      `json:"cover"`
	ReleaseDate string      `json:"releaseDate"`
	Artists     []ArtistRef `json:"artists"`
	MediaTags   []string    `json:"mediaTags"`
}

// SearchResult holds paginated search results.
type SearchResult[T any] struct {
	Items []T
	Total int
}

// AlbumDetail holds an album with its tracks.
type AlbumDetail struct {
	Album
	Tracks []Track
}

// --- JSON response envelopes (unexported, used only for unmarshaling) ---

type searchArtistsResponse struct {
	Data struct {
		Artists struct {
			Items              []Artist `json:"items"`
			TotalNumberOfItems int      `json:"totalNumberOfItems"`
		} `json:"artists"`
	} `json:"data"`
}

type searchAlbumsResponse struct {
	Data struct {
		Albums struct {
			Items              []Album `json:"items"`
			TotalNumberOfItems int     `json:"totalNumberOfItems"`
		} `json:"albums"`
	} `json:"data"`
}

type artistAlbumsResponse struct {
	Albums struct {
		Items []Album `json:"items"`
	} `json:"albums"`
}

type albumDetailResponse struct {
	Data struct {
		Album
		Items []struct {
			Item Track  `json:"item"`
			Type string `json:"type"`
		} `json:"items"`
	} `json:"data"`
}

type similarArtistsResponse struct {
	Artists []SimilarArtist `json:"artists"`
}

type similarAlbumsResponse struct {
	Albums []SimilarAlbum `json:"albums"`
}

type playbackResponse struct {
	Data Playback `json:"data"`
}

type coverResponse struct {
	Covers []Cover `json:"covers"`
}
