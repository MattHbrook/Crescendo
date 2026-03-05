package manifest

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecode_BTS(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantURLs  []string
		wantMime  string
		wantCodec string
		wantErr   bool
	}{
		{
			name:      "valid BTS manifest",
			json:      `{"mimeType":"audio/flac","codecs":"flac","encryptionType":"NONE","urls":["https://example.com/track.flac"]}`,
			wantURLs:  []string{"https://example.com/track.flac"},
			wantMime:  "audio/flac",
			wantCodec: "flac",
		},
		{
			name:      "multiple URLs",
			json:      `{"mimeType":"audio/flac","codecs":"flac","encryptionType":"NONE","urls":["https://example.com/a.flac","https://example.com/b.flac"]}`,
			wantURLs:  []string{"https://example.com/a.flac", "https://example.com/b.flac"},
			wantMime:  "audio/flac",
			wantCodec: "flac",
		},
		{
			name:    "empty URLs array",
			json:    `{"mimeType":"audio/flac","codecs":"flac","encryptionType":"NONE","urls":[]}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			json:    `{not valid json at all`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b64 := base64.StdEncoding.EncodeToString([]byte(tt.json))
			got, err := Decode(MimeTypeBTS, b64)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got.URLs) != len(tt.wantURLs) {
				t.Fatalf("URLs length = %d, want %d", len(got.URLs), len(tt.wantURLs))
			}
			for i, u := range got.URLs {
				if u != tt.wantURLs[i] {
					t.Errorf("URLs[%d] = %q, want %q", i, u, tt.wantURLs[i])
				}
			}
			if got.MimeType != tt.wantMime {
				t.Errorf("MimeType = %q, want %q", got.MimeType, tt.wantMime)
			}
			if got.Codecs != tt.wantCodec {
				t.Errorf("Codecs = %q, want %q", got.Codecs, tt.wantCodec)
			}
		})
	}

	t.Run("invalid base64", func(t *testing.T) {
		_, err := Decode(MimeTypeBTS, "!!not-base64!!")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDecode_DASH(t *testing.T) {
	const validMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011">
  <Period>
    <AdaptationSet mimeType="audio/mp4" contentType="audio">
      <Representation mimeType="audio/mp4" codecs="mha1" bandwidth="2116800">
        <BaseURL>https://example.com/track.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

	const emptyBaseURLMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011">
  <Period>
    <AdaptationSet mimeType="audio/mp4" contentType="audio">
      <Representation mimeType="audio/mp4" codecs="mha1" bandwidth="2116800">
        <BaseURL></BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

	tests := []struct {
		name      string
		xml       string
		wantURL   string
		wantMime  string
		wantCodec string
		wantErr   bool
	}{
		{
			name:      "valid DASH manifest",
			xml:       validMPD,
			wantURL:   "https://example.com/track.mp4",
			wantMime:  "audio/mp4",
			wantCodec: "mha1",
		},
		{
			name:    "empty BaseURL",
			xml:     emptyBaseURLMPD,
			wantErr: true,
		},
		{
			name:    "invalid XML",
			xml:     `<<<this is not valid xml>>>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b64 := base64.StdEncoding.EncodeToString([]byte(tt.xml))
			got, err := Decode(MimeTypeDASH, b64)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got.URLs) != 1 {
				t.Fatalf("URLs length = %d, want 1", len(got.URLs))
			}
			if got.URLs[0] != tt.wantURL {
				t.Errorf("URLs[0] = %q, want %q", got.URLs[0], tt.wantURL)
			}
			if got.MimeType != tt.wantMime {
				t.Errorf("MimeType = %q, want %q", got.MimeType, tt.wantMime)
			}
			if got.Codecs != tt.wantCodec {
				t.Errorf("Codecs = %q, want %q", got.Codecs, tt.wantCodec)
			}
		})
	}

	t.Run("invalid base64", func(t *testing.T) {
		_, err := Decode(MimeTypeDASH, "!!not-base64!!")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDecode_UnknownType(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("anything"))
	_, err := Decode("application/unknown", b64)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "unsupported")
	}
}
