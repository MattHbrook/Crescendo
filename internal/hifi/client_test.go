package hifi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer creates an httptest.Server that returns the given JSON body with
// status 200 for every request. The caller must call srv.Close() when done.
func newTestServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
}

// newErrorServer creates an httptest.Server that always returns the given HTTP
// status code with an empty body.
func newErrorServer(t *testing.T, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	}))
}

// newInvalidJSONServer creates an httptest.Server that returns malformed JSON.
func newInvalidJSONServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{not valid json`)); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
}

func TestSearchArtists(t *testing.T) {
	const fixture = `{
		"version": "2.6",
		"data": {
			"artists": {
				"limit": 10,
				"offset": 0,
				"totalNumberOfItems": 50,
				"items": [
					{"id": 8812, "name": "Coldplay", "picture": "b4579672-5b91-4679-a27a-288f097a4da5", "popularity": 92},
					{"id": 1234, "name": "Cold War Kids", "picture": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "popularity": 65}
				]
			}
		}
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.SearchArtists(context.Background(), "cold", 10, 0)
	if err != nil {
		t.Fatalf("SearchArtists returned error: %v", err)
	}

	if got, want := len(result.Items), 2; got != want {
		t.Fatalf("Items count = %d, want %d", got, want)
	}
	if got, want := result.Total, 50; got != want {
		t.Errorf("Total = %d, want %d", got, want)
	}

	first := result.Items[0]
	if first.ID != 8812 {
		t.Errorf("first.ID = %d, want 8812", first.ID)
	}
	if first.Name != "Coldplay" {
		t.Errorf("first.Name = %q, want %q", first.Name, "Coldplay")
	}
	if first.Picture != "b4579672-5b91-4679-a27a-288f097a4da5" {
		t.Errorf("first.Picture = %q, want %q", first.Picture, "b4579672-5b91-4679-a27a-288f097a4da5")
	}
}

func TestSearchAlbums(t *testing.T) {
	const fixture = `{
		"version": "2.6",
		"data": {
			"albums": {
				"limit": 10,
				"offset": 0,
				"totalNumberOfItems": 25,
				"items": [
					{
						"id": 77640617,
						"title": "A Rush of Blood to the Head",
						"cover": "deadbeef-1234-5678-9abc-def012345678",
						"releaseDate": "2002-08-12",
						"numberOfTracks": 11,
						"numberOfVolumes": 1,
						"audioQuality": "HI_RES_LOSSLESS",
						"explicit": false,
						"artist": {"id": 8812, "name": "Coldplay"},
						"artists": [{"id": 8812, "name": "Coldplay"}],
						"mediaMetadata": {"tags": ["LOSSLESS", "HIRES_LOSSLESS"]}
					}
				]
			}
		}
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.SearchAlbums(context.Background(), "rush of blood", 10, 0)
	if err != nil {
		t.Fatalf("SearchAlbums returned error: %v", err)
	}

	if got, want := len(result.Items), 1; got != want {
		t.Fatalf("Items count = %d, want %d", got, want)
	}
	if got, want := result.Total, 25; got != want {
		t.Errorf("Total = %d, want %d", got, want)
	}

	album := result.Items[0]
	if album.ID != 77640617 {
		t.Errorf("album.ID = %d, want 77640617", album.ID)
	}
	if album.Title != "A Rush of Blood to the Head" {
		t.Errorf("album.Title = %q, want %q", album.Title, "A Rush of Blood to the Head")
	}
	if album.NumberOfTracks != 11 {
		t.Errorf("album.NumberOfTracks = %d, want 11", album.NumberOfTracks)
	}
	if album.Artist.Name != "Coldplay" {
		t.Errorf("album.Artist.Name = %q, want %q", album.Artist.Name, "Coldplay")
	}
}

func TestGetArtistAlbums(t *testing.T) {
	const fixture = `{
		"albums": {
			"items": [
				{"id": 100, "title": "Album One", "artist": {"id": 8812, "name": "Coldplay"}},
				{"id": 200, "title": "Album Two", "artist": {"id": 8812, "name": "Coldplay"}},
				{"id": 300, "title": "Album Three", "artist": {"id": 8812, "name": "Coldplay"}}
			]
		}
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	albums, err := c.GetArtistAlbums(context.Background(), 8812)
	if err != nil {
		t.Fatalf("GetArtistAlbums returned error: %v", err)
	}

	if got, want := len(albums), 3; got != want {
		t.Fatalf("album count = %d, want %d", got, want)
	}
	if albums[0].ID != 100 {
		t.Errorf("albums[0].ID = %d, want 100", albums[0].ID)
	}
	if albums[0].Title != "Album One" {
		t.Errorf("albums[0].Title = %q, want %q", albums[0].Title, "Album One")
	}
}

func TestGetAlbum(t *testing.T) {
	const fixture = `{
		"data": {
			"id": 77640617,
			"title": "A Rush of Blood to the Head",
			"cover": "deadbeef-1234-5678-9abc-def012345678",
			"releaseDate": "2002-08-12",
			"numberOfTracks": 2,
			"numberOfVolumes": 1,
			"audioQuality": "HI_RES_LOSSLESS",
			"explicit": false,
			"artist": {"id": 8812, "name": "Coldplay"},
			"artists": [{"id": 8812, "name": "Coldplay"}],
			"mediaMetadata": {"tags": ["LOSSLESS"]},
			"items": [
				{
					"item": {
						"id": 10001,
						"title": "Politik",
						"duration": 317,
						"trackNumber": 1,
						"volumeNumber": 1,
						"audioQuality": "HI_RES_LOSSLESS",
						"artist": {"id": 8812, "name": "Coldplay"},
						"artists": [{"id": 8812, "name": "Coldplay"}]
					},
					"type": "track"
				},
				{
					"item": {
						"id": 10002,
						"title": "In My Place",
						"duration": 228,
						"trackNumber": 2,
						"volumeNumber": 1,
						"audioQuality": "HI_RES_LOSSLESS",
						"artist": {"id": 8812, "name": "Coldplay"},
						"artists": [{"id": 8812, "name": "Coldplay"}]
					},
					"type": "track"
				}
			]
		}
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	detail, err := c.GetAlbum(context.Background(), 77640617)
	if err != nil {
		t.Fatalf("GetAlbum returned error: %v", err)
	}

	if detail.ID != 77640617 {
		t.Errorf("album.ID = %d, want 77640617", detail.ID)
	}
	if detail.Title != "A Rush of Blood to the Head" {
		t.Errorf("album.Title = %q, want %q", detail.Title, "A Rush of Blood to the Head")
	}
	if got, want := len(detail.Tracks), 2; got != want {
		t.Fatalf("track count = %d, want %d", got, want)
	}
	if detail.Tracks[0].TrackNumber != 1 {
		t.Errorf("Tracks[0].TrackNumber = %d, want 1", detail.Tracks[0].TrackNumber)
	}
	if detail.Tracks[0].Title != "Politik" {
		t.Errorf("Tracks[0].Title = %q, want %q", detail.Tracks[0].Title, "Politik")
	}
	if detail.Tracks[1].TrackNumber != 2 {
		t.Errorf("Tracks[1].TrackNumber = %d, want 2", detail.Tracks[1].TrackNumber)
	}
	if detail.Tracks[1].Title != "In My Place" {
		t.Errorf("Tracks[1].Title = %q, want %q", detail.Tracks[1].Title, "In My Place")
	}
}

func TestGetSimilarArtists(t *testing.T) {
	const fixture = `{
		"artists": [
			{"id": 3520813, "name": "Radiohead", "picture": "11111111-2222-3333-4444-555555555555", "popularity": 0.89},
			{"id": 14670,   "name": "Muse",      "picture": "66666666-7777-8888-9999-aaaaaaaaaaaa", "popularity": 0.82}
		]
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	artists, err := c.GetSimilarArtists(context.Background(), 8812)
	if err != nil {
		t.Fatalf("GetSimilarArtists returned error: %v", err)
	}

	if got, want := len(artists), 2; got != want {
		t.Fatalf("artist count = %d, want %d", got, want)
	}
	if artists[0].ID != 3520813 {
		t.Errorf("artists[0].ID = %d, want 3520813", artists[0].ID)
	}
	if artists[0].Name != "Radiohead" {
		t.Errorf("artists[0].Name = %q, want %q", artists[0].Name, "Radiohead")
	}
}

func TestGetSimilarAlbums(t *testing.T) {
	const fixture = `{
		"albums": [
			{
				"id": 50001,
				"title": "OK Computer",
				"cover": "abababab-cdcd-efef-1212-343434343434",
				"releaseDate": "1997-06-16",
				"artists": [{"id": 3520813, "name": "Radiohead"}],
				"mediaTags": ["LOSSLESS"]
			},
			{
				"id": 50002,
				"title": "The Bends",
				"cover": "fefefefe-dcdc-baba-9898-767676767676",
				"releaseDate": "1995-03-13",
				"artists": [{"id": 3520813, "name": "Radiohead"}],
				"mediaTags": ["LOSSLESS"]
			}
		]
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	albums, err := c.GetSimilarAlbums(context.Background(), 77640617)
	if err != nil {
		t.Fatalf("GetSimilarAlbums returned error: %v", err)
	}

	if got, want := len(albums), 2; got != want {
		t.Fatalf("album count = %d, want %d", got, want)
	}
	if albums[0].ID != 50001 {
		t.Errorf("albums[0].ID = %d, want 50001", albums[0].ID)
	}
	if albums[0].Title != "OK Computer" {
		t.Errorf("albums[0].Title = %q, want %q", albums[0].Title, "OK Computer")
	}
}

func TestGetTrackPlayback(t *testing.T) {
	const fixture = `{
		"data": {
			"trackId": 10001,
			"audioQuality": "HI_RES_LOSSLESS",
			"manifestMimeType": "application/vnd.tidal.bts",
			"manifest": "eyJtaW1lVHlwZSI6ICJhdWRpby9mbGFjIn0=",
			"bitDepth": 24,
			"sampleRate": 96000
		}
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	pb, err := c.GetTrackPlayback(context.Background(), 10001, "HI_RES_LOSSLESS")
	if err != nil {
		t.Fatalf("GetTrackPlayback returned error: %v", err)
	}

	if pb.TrackID != 10001 {
		t.Errorf("TrackID = %d, want 10001", pb.TrackID)
	}
	if pb.ManifestMimeType != "application/vnd.tidal.bts" {
		t.Errorf("ManifestMimeType = %q, want %q", pb.ManifestMimeType, "application/vnd.tidal.bts")
	}
	if pb.Manifest != "eyJtaW1lVHlwZSI6ICJhdWRpby9mbGFjIn0=" {
		t.Errorf("Manifest = %q, want %q", pb.Manifest, "eyJtaW1lVHlwZSI6ICJhdWRpby9mbGFjIn0=")
	}
	if pb.BitDepth != 24 {
		t.Errorf("BitDepth = %d, want 24", pb.BitDepth)
	}
	if pb.SampleRate != 96000 {
		t.Errorf("SampleRate = %d, want 96000", pb.SampleRate)
	}
}

func TestGetCover(t *testing.T) {
	const fixture = `{
		"covers": [
			{
				"id": 77640617,
				"name": "album_cover",
				"1280": "https://resources.tidal.com/images/deadbeef/1280x1280.jpg",
				"640":  "https://resources.tidal.com/images/deadbeef/640x640.jpg",
				"80":   "https://resources.tidal.com/images/deadbeef/80x80.jpg"
			}
		]
	}`

	srv := newTestServer(t, fixture)
	defer srv.Close()

	c := NewClient(srv.URL)
	cover, err := c.GetCover(context.Background(), 77640617)
	if err != nil {
		t.Fatalf("GetCover returned error: %v", err)
	}

	if cover.URL1280 != "https://resources.tidal.com/images/deadbeef/1280x1280.jpg" {
		t.Errorf("URL1280 = %q, want %q", cover.URL1280, "https://resources.tidal.com/images/deadbeef/1280x1280.jpg")
	}
	if cover.URL640 != "https://resources.tidal.com/images/deadbeef/640x640.jpg" {
		t.Errorf("URL640 = %q, want %q", cover.URL640, "https://resources.tidal.com/images/deadbeef/640x640.jpg")
	}
	if cover.URL80 != "https://resources.tidal.com/images/deadbeef/80x80.jpg" {
		t.Errorf("URL80 = %q, want %q", cover.URL80, "https://resources.tidal.com/images/deadbeef/80x80.jpg")
	}
}

func TestClient_HTTP_error(t *testing.T) {
	tests := []struct {
		name string
		call func(c *Client) error
	}{
		{
			name: "SearchArtists",
			call: func(c *Client) error {
				_, err := c.SearchArtists(context.Background(), "test", 10, 0)
				return err
			},
		},
		{
			name: "SearchAlbums",
			call: func(c *Client) error {
				_, err := c.SearchAlbums(context.Background(), "test", 10, 0)
				return err
			},
		},
		{
			name: "GetArtistAlbums",
			call: func(c *Client) error {
				_, err := c.GetArtistAlbums(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetAlbum",
			call: func(c *Client) error {
				_, err := c.GetAlbum(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetSimilarArtists",
			call: func(c *Client) error {
				_, err := c.GetSimilarArtists(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetSimilarAlbums",
			call: func(c *Client) error {
				_, err := c.GetSimilarAlbums(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetTrackPlayback",
			call: func(c *Client) error {
				_, err := c.GetTrackPlayback(context.Background(), 1, "HI_RES_LOSSLESS")
				return err
			},
		},
		{
			name: "GetCover",
			call: func(c *Client) error {
				_, err := c.GetCover(context.Background(), 1)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newErrorServer(t, http.StatusInternalServerError)
			defer srv.Close()

			c := NewClient(srv.URL)
			if err := tt.call(c); err == nil {
				t.Error("expected error for 500 response, got nil")
			}
		})
	}
}

func TestClient_invalid_JSON(t *testing.T) {
	tests := []struct {
		name string
		call func(c *Client) error
	}{
		{
			name: "SearchArtists",
			call: func(c *Client) error {
				_, err := c.SearchArtists(context.Background(), "test", 10, 0)
				return err
			},
		},
		{
			name: "SearchAlbums",
			call: func(c *Client) error {
				_, err := c.SearchAlbums(context.Background(), "test", 10, 0)
				return err
			},
		},
		{
			name: "GetArtistAlbums",
			call: func(c *Client) error {
				_, err := c.GetArtistAlbums(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetAlbum",
			call: func(c *Client) error {
				_, err := c.GetAlbum(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetSimilarArtists",
			call: func(c *Client) error {
				_, err := c.GetSimilarArtists(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetSimilarAlbums",
			call: func(c *Client) error {
				_, err := c.GetSimilarAlbums(context.Background(), 1)
				return err
			},
		},
		{
			name: "GetTrackPlayback",
			call: func(c *Client) error {
				_, err := c.GetTrackPlayback(context.Background(), 1, "HI_RES_LOSSLESS")
				return err
			},
		},
		{
			name: "GetCover",
			call: func(c *Client) error {
				_, err := c.GetCover(context.Background(), 1)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newInvalidJSONServer(t)
			defer srv.Close()

			c := NewClient(srv.URL)
			if err := tt.call(c); err == nil {
				t.Error("expected error for invalid JSON, got nil")
			}
		})
	}
}
