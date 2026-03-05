package manifest

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
)

const (
	// MimeTypeBTS is the manifest MIME type for BTS (lossless) streams.
	MimeTypeBTS = "application/vnd.tidal.bts"

	// MimeTypeDASH is the manifest MIME type for DASH (hi-res lossless) streams.
	MimeTypeDASH = "application/dash+xml"
)

// Result holds the decoded manifest data needed by the downloader.
type Result struct {
	URLs     []string // direct download URLs (BTS always has these; DASH has the BaseURL)
	MimeType string   // e.g. "audio/flac", "audio/mp4"
	Codecs   string   // e.g. "flac", "mqa", "mha1"
}

type btsManifest struct {
	MimeType       string   `json:"mimeType"`
	Codecs         string   `json:"codecs"`
	EncryptionType string   `json:"encryptionType"`
	URLs           []string `json:"urls"`
}

type mpd struct {
	XMLName xml.Name  `xml:"MPD"`
	Period  mpdPeriod `xml:"Period"`
}

type mpdPeriod struct {
	AdaptationSet mpdAdaptationSet `xml:"AdaptationSet"`
}

type mpdAdaptationSet struct {
	Representation mpdRepresentation `xml:"Representation"`
}

type mpdRepresentation struct {
	MimeType string `xml:"mimeType,attr"`
	Codecs   string `xml:"codecs,attr"`
	BaseURL  string `xml:"BaseURL"`
}

// Decode base64-decodes the manifest and parses it according to the given MIME type.
func Decode(manifestMimeType, manifestB64 string) (*Result, error) {
	data, err := base64.StdEncoding.DecodeString(manifestB64)
	if err != nil {
		return nil, fmt.Errorf("manifest: base64 decode: %w", err)
	}

	switch manifestMimeType {
	case MimeTypeBTS:
		return decodeBTS(data)
	case MimeTypeDASH:
		return decodeDASH(data)
	default:
		return nil, fmt.Errorf("manifest: unsupported manifest type: %s", manifestMimeType)
	}
}

func decodeBTS(data []byte) (*Result, error) {
	var bts btsManifest
	if err := json.Unmarshal(data, &bts); err != nil {
		return nil, fmt.Errorf("manifest: unmarshal BTS JSON: %w", err)
	}

	if len(bts.URLs) == 0 {
		return nil, fmt.Errorf("manifest: BTS manifest contains no URLs")
	}

	return &Result{
		URLs:     bts.URLs,
		MimeType: bts.MimeType,
		Codecs:   bts.Codecs,
	}, nil
}

func decodeDASH(data []byte) (*Result, error) {
	var m mpd
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: unmarshal DASH MPD: %w", err)
	}

	rep := m.Period.AdaptationSet.Representation
	if rep.BaseURL == "" {
		return nil, fmt.Errorf("manifest: DASH manifest contains no BaseURL")
	}

	return &Result{
		URLs:     []string{rep.BaseURL},
		MimeType: rep.MimeType,
		Codecs:   rep.Codecs,
	}, nil
}
