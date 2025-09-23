package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractMetadataFromPath tests path-based metadata extraction
func TestExtractMetadataFromPath(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		expectedTitle string
		expectedArtist string
		expectedAlbum string
		expectedTrackNumber int
	}{
		{
			name:         "standard structure with track number",
			filePath:     "Artist Name/Album Name/01 - Song Title.flac",
			expectedTitle: "Song Title",
			expectedArtist: "Artist Name",
			expectedAlbum: "Album Name",
			expectedTrackNumber: 1,
		},
		{
			name:         "double digit track number",
			filePath:     "The Beatles/Abbey Road/12 - Come Together.flac",
			expectedTitle: "Come Together",
			expectedArtist: "The Beatles",
			expectedAlbum: "Abbey Road",
			expectedTrackNumber: 12,
		},
		{
			name:         "track number with dot",
			filePath:     "Artist/Album/3. Track Name.mp3",
			expectedTitle: "Track Name",
			expectedArtist: "Artist",
			expectedAlbum: "Album",
			expectedTrackNumber: 3,
		},
		{
			name:         "no track number",
			filePath:     "Artist/Album/Song Title.flac",
			expectedTitle: "Song Title",
			expectedArtist: "Artist",
			expectedAlbum: "Album",
			expectedTrackNumber: 0,
		},
		{
			name:         "complex artist name",
			filePath:     "Foo Fighters/Wasting Light/01 - Bridge Burning.flac",
			expectedTitle: "Bridge Burning",
			expectedArtist: "Foo Fighters",
			expectedAlbum: "Wasting Light",
			expectedTrackNumber: 1,
		},
		{
			name:         "single directory level",
			filePath:     "Artist/Song.mp3",
			expectedTitle: "Song",
			expectedArtist: "",
			expectedAlbum: "Artist",
			expectedTrackNumber: 0,
		},
		{
			name:         "flat file",
			filePath:     "Song.flac",
			expectedTitle: "Song",
			expectedArtist: "",
			expectedAlbum: "",
			expectedTrackNumber: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := extractMetadataFromPath(tt.filePath)

			assert.Equal(t, tt.expectedTitle, metadata.Title)
			assert.Equal(t, tt.expectedArtist, metadata.Artist)
			assert.Equal(t, tt.expectedAlbum, metadata.Album)
			assert.Equal(t, tt.expectedTrackNumber, metadata.TrackNumber)
		})
	}
}

// TestGetContentType tests MIME type detection
func TestGetContentType(t *testing.T) {
	tests := []struct {
		filePath        string
		expectedType    string
	}{
		{"test.flac", "audio/flac"},
		{"test.FLAC", "audio/flac"},
		{"test.mp3", "audio/mpeg"},
		{"test.MP3", "audio/mpeg"},
		{"test.txt", "application/octet-stream"},
		{"test", "application/octet-stream"},
		{"/path/to/file.flac", "audio/flac"},
		{"Artist/Album/Song.mp3", "audio/mpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			contentType := getContentType(tt.filePath)
			assert.Equal(t, tt.expectedType, contentType)
		})
	}
}

// TestScanAudioFiles tests the audio file scanning functionality
func TestScanAudioFiles(t *testing.T) {
	// Create temporary test directory
	testDir, err := os.MkdirTemp("", "crescendo-scan-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Save original working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Change to test directory
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create test directory structure
	testFiles := []struct {
		path    string
		content []byte
	}{
		{"Artist1/Album1/01 - Song1.flac", createMinimalFLACFile()},
		{"Artist1/Album1/02 - Song2.mp3", createMinimalMP3File()},
		{"Artist2/Album2/Track.flac", createMinimalFLACFile()},
		{"Artist2/Album2/Track.mp3", createMinimalMP3File()}, // Same name as FLAC
		{"Artist3/Album3/NoExt", []byte("not an audio file")},
		{"Artist3/Album3/document.txt", []byte("text file")},
	}

	// Create test files
	for _, tf := range testFiles {
		dir := filepath.Dir(tf.path)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(tf.path, tf.content, 0644)
		require.NoError(t, err)
	}

	// Scan files
	audioFiles, err := scanAudioFiles(".")
	require.NoError(t, err)

	// Verify results
	assert.GreaterOrEqual(t, len(audioFiles), 3) // Should have at least 3 files

	// Check that only audio files are included
	for _, file := range audioFiles {
		assert.True(t, file.Format == "flac" || file.Format == "mp3",
			"File should be FLAC or MP3: %s", file.Path)
		assert.NotEmpty(t, file.Filename)
		assert.Greater(t, file.Size, int64(0))
	}

	// Check FLAC prioritization - should only see FLAC version of Track
	trackFiles := make([]AudioFile, 0)
	for _, file := range audioFiles {
		if filepath.Base(file.Path) == "Track.flac" || filepath.Base(file.Path) == "Track.mp3" {
			trackFiles = append(trackFiles, file)
		}
	}

	// Should only have FLAC version due to prioritization
	require.Equal(t, 1, len(trackFiles))
	assert.Equal(t, "flac", trackFiles[0].Format)
}

// TestApplyFlacPrioritization tests FLAC prioritization logic
func TestApplyFlacPrioritization(t *testing.T) {
	// Create test files with overlapping names
	testFiles := []AudioFile{
		{
			Filename: "Song1.flac",
			Path:     "Artist/Album/Song1.flac",
			Format:   "flac",
			Size:     1000,
		},
		{
			Filename: "Song1.mp3",
			Path:     "Artist/Album/Song1.mp3",
			Format:   "mp3",
			Size:     800,
		},
		{
			Filename: "Song2.mp3",
			Path:     "Artist/Album/Song2.mp3",
			Format:   "mp3",
			Size:     900,
		},
		{
			Filename: "Song3.flac",
			Path:     "Artist/Album/Song3.flac",
			Format:   "flac",
			Size:     1200,
		},
	}

	result := applyFlacPrioritization(testFiles)

	// Should have 3 files total (Song1 FLAC preferred, Song2 MP3 only, Song3 FLAC only)
	assert.Equal(t, 3, len(result))

	// Check that FLAC is prioritized for Song1
	song1Found := false
	for _, file := range result {
		if filepath.Base(file.Path) == "Song1.flac" {
			song1Found = true
			assert.Equal(t, "flac", file.Format)
			break
		}
	}
	assert.True(t, song1Found, "Should have FLAC version of Song1")

	// Check that MP3 version of Song1 is not included
	for _, file := range result {
		assert.NotEqual(t, "Artist/Album/Song1.mp3", file.Path,
			"MP3 version of Song1 should not be included")
	}

	// Check that Song2 MP3 is included (no FLAC alternative)
	song2Found := false
	for _, file := range result {
		if filepath.Base(file.Path) == "Song2.mp3" {
			song2Found = true
			break
		}
	}
	assert.True(t, song2Found, "Should have MP3 version of Song2")
}

// TestMetadataExtractionWithCorruptedFiles tests handling of corrupted files
func TestMetadataExtractionWithCorruptedFiles(t *testing.T) {
	// Create temporary test directory
	testDir, err := os.MkdirTemp("", "crescendo-corrupt-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Save and change working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create corrupted files
	corruptedFiles := []struct {
		path    string
		content []byte
	}{
		{"Artist/Album/corrupted.flac", []byte("not a real flac file")},
		{"Artist/Album/empty.mp3", []byte("")},
		{"Artist/Album/01 - Good Song.flac", []byte("also not flac but has good path")},
	}

	// Create test files
	for _, cf := range corruptedFiles {
		dir := filepath.Dir(cf.path)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(cf.path, cf.content, 0644)
		require.NoError(t, err)
	}

	// Scan files - should not crash
	audioFiles, err := scanAudioFiles(".")
	require.NoError(t, err)

	// Should have all files (metadata extraction should fall back to path parsing)
	assert.Equal(t, 3, len(audioFiles))

	// Check that metadata was extracted from paths for corrupted files
	for _, file := range audioFiles {
		assert.NotNil(t, file.Metadata)
		assert.Equal(t, "Artist", file.Metadata.Artist)
		assert.Equal(t, "Album", file.Metadata.Album)

		if file.Filename == "01 - Good Song.flac" {
			assert.Equal(t, "Good Song", file.Metadata.Title)
			assert.Equal(t, 1, file.Metadata.TrackNumber)
		}
	}
}

// TestMetadataExtractionEdgeCases tests edge cases in metadata extraction
func TestMetadataExtractionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expectedResult string
	}{
		{
			name:     "special characters in path",
			filePath: "Artíst/Albüm/01 - Sóng.flac",
			expectedResult: "should handle unicode",
		},
		{
			name:     "very long path",
			filePath: "Very Long Artist Name/Very Long Album Name With Multiple Words/01 - Very Long Song Title That Goes On And On.flac",
			expectedResult: "should handle long names",
		},
		{
			name:     "path with spaces",
			filePath: " Artist / Album / 01 -  Song .flac",
			expectedResult: "should handle extra spaces",
		},
		{
			name:     "numbers in names",
			filePath: "Artist123/Album456/01 - Song789.flac",
			expectedResult: "should handle numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic or crash
			metadata := extractMetadataFromPath(tt.filePath)
			assert.NotNil(t, metadata)
			assert.NotEmpty(t, metadata.Title)
		})
	}
}

// BenchmarkMetadataExtraction benchmarks metadata extraction performance
func BenchmarkMetadataExtraction(b *testing.B) {
	testPath := "Test Artist/Test Album/01 - Test Song.flac"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractMetadataFromPath(testPath)
	}
}

// BenchmarkFlacPrioritization benchmarks FLAC prioritization performance
func BenchmarkFlacPrioritization(b *testing.B) {
	// Create test data
	testFiles := make([]AudioFile, 1000)
	for i := 0; i < 500; i++ {
		testFiles[i*2] = AudioFile{
			Path:   fmt.Sprintf("Artist/Album/Song%d.flac", i),
			Format: "flac",
		}
		testFiles[i*2+1] = AudioFile{
			Path:   fmt.Sprintf("Artist/Album/Song%d.mp3", i),
			Format: "mp3",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyFlacPrioritization(testFiles)
	}
}