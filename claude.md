# Crescendo - Web Service Transformation Project

## Project Overview
Transform the existing godab CLI application into a web service backend for "Crescendo", a modern React-based audio downloader. The godab project already handles all complex DAB Music API integration, file downloads, and organization - we need to add web service capabilities while preserving existing CLI functionality.

## Current Project Analysis
The godab project is a Go CLI application with this structure:
```
godab/
├── main.go              # CLI entry point with flag parsing
├── api/                 # DAB Music API integration
│   ├── api.go          # Core API client
│   ├── album.go        # Album download logic
│   ├── track.go        # Track download logic
│   ├── artist.go       # Artist discography logic
│   └── utils.go        # Utility functions
├── config/
│   └── env.go          # Environment configuration
├── go.mod              # Dependencies (already includes required packages)
└── go.sum              # Dependency checksums
```

## Existing Functionality (Keep Intact)
- ✅ DAB Music API integration (no auth required)
- ✅ Album downloads with concurrent track processing
- ✅ Single track downloads
- ✅ Artist discography downloads
- ✅ Automatic file organization: `Artist/Album/Track.flac`
- ✅ Metadata embedding (cover art, tags)
- ✅ Progress tracking with goroutines
- ✅ FLAC quality downloads (quality=27)

## DAB API Endpoints Already Integrated
- `/api/album?albumId=X` - Get album metadata and tracks
- `/api/stream?trackId=X&quality=27` - Get download URL
- `/api/search?q=X&type=track|album|artist` - Search functionality
- `/api/discography?artistId=X` - Get artist's albums

## Transformation Requirements

### Phase 1: Add Web Server Mode
**Goal:** Keep CLI functionality, add HTTP server option

**Implementation:**
1. Modify `main.go` to support `--server` flag
2. Add Gin HTTP framework setup
3. Create basic health check endpoint
4. Preserve all existing CLI commands

**New CLI Usage:**
```bash
# Existing CLI (keep working)
go run main.go -album <album_id>
go run main.go -track <track_id>
go run main.go -artist <artist_id>

# New server mode
go run main.go --server --port 8080
```

### Phase 2: REST API Endpoints
**Create handlers that wrap existing api/ package functions:**

**Search & Discovery:**
- `GET /api/search?q=query&type=album|track|artist`
- `GET /api/album/:id`
- `GET /api/artist/:id/discography`

**Download Management:**
- `POST /api/downloads/album/:id` - Queue album download
- `POST /api/downloads/track/:id` - Queue track download
- `POST /api/downloads/artist/:id` - Queue artist discography
- `GET /api/downloads` - List active downloads
- `GET /api/downloads/:jobId` - Get download status
- `DELETE /api/downloads/:jobId` - Cancel download

**File Management:**
- `GET /api/files` - List downloaded files
- `GET /api/files/:path/stream` - Serve downloaded files
- `DELETE /api/files/:path` - Delete files

### Phase 3: Real-time Progress
**Add WebSocket support for live download progress:**
- `WebSocket /ws/downloads/:jobId` - Real-time progress updates
- Progress data: `{percentage: number, status: string, currentFile: string}`

### Technical Requirements

**Dependencies to Add:**
```go
// Add to go.mod
github.com/gin-gonic/gin
github.com/gin-contrib/cors
github.com/gorilla/websocket
```

**Key Features:**
1. **CORS Support** - Enable React frontend communication
2. **Job Queue System** - Manage concurrent downloads
3. **Progress Tracking** - Real-time updates via WebSocket
4. **Error Handling** - Proper HTTP status codes
5. **File Serving** - Static file server for downloads
6. **Request Validation** - Input sanitization

**Environment Variables:**
```bash
DAB_ENDPOINT=https://dabmusic.xyz    # Keep existing
DOWNLOAD_LOCATION=/path/to/downloads # Keep existing
SERVER_PORT=8080                     # New for web mode
CORS_ORIGIN=http://localhost:3000    # For React dev server
```

## Development Approach

### Step 1: Minimal Web Server
Add basic HTTP server to main.go with health check endpoint while keeping CLI intact.

### Step 2: Search Endpoints
Wrap existing search functionality in HTTP handlers with JSON responses.

### Step 3: Download Queue
Implement job queue system for async downloads with status tracking.

### Step 4: Progress WebSockets
Add real-time progress updates for the React frontend.

### Step 5: File Management
Add endpoints to serve and manage downloaded files.

## React Frontend Integration Plan
**Frontend will be built separately as "Crescendo" with:**
- Search interface using `/api/search`
- Download queue management via REST API
- Real-time progress via WebSocket
- File browser for downloaded content
- Settings for download preferences

## Success Criteria
1. ✅ CLI functionality preserved and working
2. ✅ Web server starts with `--server` flag
3. ✅ Search API returns JSON responses
4. ✅ Download queue accepts and processes jobs
5. ✅ WebSocket provides real-time progress
6. ✅ CORS configured for React development
7. ✅ File serving works for downloaded content

## Architecture Decisions
- **Keep existing api/ package unchanged** - just wrap with HTTP handlers
- **Use Gin framework** - fast, minimal, good middleware support
- **Channel-based job queue** - leverage Go's concurrency
- **WebSocket for progress** - real-time updates without polling
- **Preserve file structure** - maintain existing download organization
- **Environment-based config** - same pattern as current implementation

## Development Priority
Focus on creating a working web service that preserves all existing functionality while adding HTTP/WebSocket interfaces. The React frontend will be built separately once the backend API is stable.