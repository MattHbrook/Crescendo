package hifi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is an HTTP client for the hifi-api (Tidal proxy).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new hifi-api client pointed at the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// get performs a GET request, checks the status code, and decodes the JSON
// response body into dest.
func (c *Client) get(ctx context.Context, path string, params url.Values, dest any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("hifi: parse url: %w", err)
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("hifi: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) //nolint:gosec // baseURL is set by trusted config, not user input
	if err != nil {
		return fmt.Errorf("hifi: do request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("hifi: close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hifi: unexpected status %d for %s", resp.StatusCode, path)
	}

	if err = json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("hifi: decode response: %w", err)
	}

	return nil
}

// SearchArtists searches for artists by name.
func (c *Client) SearchArtists(ctx context.Context, query string, limit, offset int) (*SearchResult[Artist], error) {
	params := url.Values{
		"a":      {query},
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}

	var resp searchArtistsResponse
	if err := c.get(ctx, "/search/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: search artists: %w", err)
	}

	return &SearchResult[Artist]{
		Items: resp.Data.Artists.Items,
		Total: resp.Data.Artists.TotalNumberOfItems,
	}, nil
}

// SearchAlbums searches for albums by name.
func (c *Client) SearchAlbums(ctx context.Context, query string, limit, offset int) (*SearchResult[Album], error) {
	params := url.Values{
		"al":     {query},
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}

	var resp searchAlbumsResponse
	if err := c.get(ctx, "/search/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: search albums: %w", err)
	}

	return &SearchResult[Album]{
		Items: resp.Data.Albums.Items,
		Total: resp.Data.Albums.TotalNumberOfItems,
	}, nil
}

// GetArtistAlbums returns all albums for the given artist.
func (c *Client) GetArtistAlbums(ctx context.Context, id int64) ([]Album, error) {
	params := url.Values{
		"f":           {strconv.FormatInt(id, 10)},
		"skip_tracks": {"true"},
	}

	var resp artistAlbumsResponse
	if err := c.get(ctx, "/artist/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: get artist albums: %w", err)
	}

	return resp.Albums.Items, nil
}

// GetAlbum returns album details including tracks.
func (c *Client) GetAlbum(ctx context.Context, id int64) (*AlbumDetail, error) {
	params := url.Values{
		"id": {strconv.FormatInt(id, 10)},
	}

	var resp albumDetailResponse
	if err := c.get(ctx, "/album/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: get album: %w", err)
	}

	tracks := make([]Track, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		tracks = append(tracks, item.Item)
	}

	return &AlbumDetail{
		Album:  resp.Data.Album,
		Tracks: tracks,
	}, nil
}

// GetSimilarArtists returns artists similar to the given artist.
func (c *Client) GetSimilarArtists(ctx context.Context, id int64) ([]SimilarArtist, error) {
	params := url.Values{
		"id": {strconv.FormatInt(id, 10)},
	}

	var resp similarArtistsResponse
	if err := c.get(ctx, "/artist/similar/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: get similar artists: %w", err)
	}

	return resp.Artists, nil
}

// GetSimilarAlbums returns albums similar to the given album.
func (c *Client) GetSimilarAlbums(ctx context.Context, id int64) ([]SimilarAlbum, error) {
	params := url.Values{
		"id": {strconv.FormatInt(id, 10)},
	}

	var resp similarAlbumsResponse
	if err := c.get(ctx, "/album/similar/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: get similar albums: %w", err)
	}

	return resp.Albums, nil
}

// GetTrackPlayback returns playback info (including the manifest) for a track.
func (c *Client) GetTrackPlayback(ctx context.Context, id int64, quality string) (*Playback, error) {
	params := url.Values{
		"id":      {strconv.FormatInt(id, 10)},
		"quality": {quality},
	}

	var resp playbackResponse
	if err := c.get(ctx, "/track/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: get track playback: %w", err)
	}

	return &resp.Data, nil
}

// GetCover returns cover art URLs for the given album. It returns the first
// cover entry or an error if none are available.
func (c *Client) GetCover(ctx context.Context, albumID int64) (*Cover, error) {
	params := url.Values{
		"id": {strconv.FormatInt(albumID, 10)},
	}

	var resp coverResponse
	if err := c.get(ctx, "/cover/", params, &resp); err != nil {
		return nil, fmt.Errorf("hifi: get cover: %w", err)
	}

	if len(resp.Covers) == 0 {
		return nil, fmt.Errorf("hifi: get cover: no covers returned for album %d", albumID)
	}

	return &resp.Covers[0], nil
}
