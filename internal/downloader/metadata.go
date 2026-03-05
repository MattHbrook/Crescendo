package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	flac "github.com/go-flac/go-flac/v2"
	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
)

// trackMeta holds the metadata needed to tag a single FLAC file.
type trackMeta struct {
	Artist      string
	Album       string
	Title       string
	TrackNumber int
	DiscNumber  int
	TotalDiscs  int
	Date        string // YYYY or YYYY-MM-DD
	CoverJPEG   []byte // raw JPEG bytes (nil to skip picture embedding)
}

// tagFLAC writes Vorbis comments and optionally embeds cover art into a FLAC file.
func tagFLAC(path string, meta trackMeta) error {
	f, err := flac.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse flac: %w", err)
	}
	defer f.Close()

	// Build a fresh Vorbis comment block.
	cmts := flacvorbis.New()
	cmts.Add(flacvorbis.FIELD_ARTIST, meta.Artist)
	cmts.Add(flacvorbis.FIELD_ALBUM, meta.Album)
	cmts.Add(flacvorbis.FIELD_TITLE, meta.Title)
	cmts.Add(flacvorbis.FIELD_TRACKNUMBER, strconv.Itoa(meta.TrackNumber))
	cmts.Add(flacvorbis.FIELD_DATE, meta.Date)
	if meta.TotalDiscs > 1 {
		cmts.Add("DISCNUMBER", strconv.Itoa(meta.DiscNumber))
	}

	cmtBlock := cmts.Marshal()

	// Replace existing VorbisComment block or append a new one.
	replaced := false
	for i, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			f.Meta[i] = &cmtBlock
			replaced = true
			break
		}
	}
	if !replaced {
		f.Meta = append(f.Meta, &cmtBlock)
	}

	// Embed cover art if provided.
	if len(meta.CoverJPEG) > 0 {
		// Strip any existing picture blocks to avoid duplicates.
		filtered := f.Meta[:0]
		for _, block := range f.Meta {
			if block.Type != flac.Picture {
				filtered = append(filtered, block)
			}
		}
		f.Meta = filtered

		pic, err := flacpicture.NewFromImageData(
			flacpicture.PictureTypeFrontCover,
			"Front cover",
			meta.CoverJPEG,
			"image/jpeg",
		)
		if err != nil {
			return fmt.Errorf("create picture block: %w", err)
		}
		picBlock := pic.Marshal()
		f.Meta = append(f.Meta, &picBlock)
	}

	return f.Save(path)
}

// downloadCover fetches cover art from the given URL and returns the raw bytes.
// It also saves the image as cover.jpg in albumDir.
func downloadCover(ctx context.Context, coverURL, albumDir string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating cover request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // URL comes from Tidal API
	if err != nil {
		return nil, fmt.Errorf("downloading cover: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading cover data: %w", err)
	}

	// Save cover.jpg to album directory.
	coverPath := filepath.Join(albumDir, "cover.jpg")
	if err := os.WriteFile(coverPath, data, 0o644); err != nil { //nolint:gosec // non-sensitive file
		return nil, fmt.Errorf("saving cover.jpg: %w", err)
	}

	return data, nil
}
