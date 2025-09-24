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

## Code Quality & Cleanup (CRITICAL BEFORE UI)

**Current Status:** ⚠️ **Functional but requires cleanup before UI development**

### Analysis Summary
Comprehensive codebase analysis completed (see `docs/ANALYSIS.md`). While the web service transformation is functionally complete, critical architectural issues must be addressed before frontend development:

**Critical Issues:**
- 🔴 **main.go too large** (1,306 lines) - violates maintainability principles
- 🔴 **Security warnings** - debug mode, proxy trust issues in production
- 🔴 **Missing modular architecture** - all code in single file
- 🟡 **Limited test coverage** - missing API integration tests
- 🟡 **No static analysis** - quality checks needed

### Cleanup Plan Overview

**📋 PHASE 1: Architecture Refactoring (REQUIRED)**
- **Priority:** CRITICAL for UI development
- **Timeline:** 1 development session
- **Goal:** Modular, maintainable, secure codebase

Key tasks:
- Split main.go into packages (`handlers/`, `services/`, `websocket/`, `middleware/`)
- Fix security warnings (production mode, CORS configuration)
- Standardize error handling and logging
- See: `docs/PHASE_1_ARCHITECTURE.md`

**📋 PHASE 2: Testing & Quality (RECOMMENDED)**
- **Priority:** Production readiness
- **Timeline:** 1 development session
- **Goal:** Comprehensive testing and performance optimization

Key tasks:
- Add static analysis (golangci-lint)
- Expand test coverage (API integration, security)
- Performance optimization (connection pooling, rate limiting)
- See: `docs/PHASE_2_TESTING.md`

**📋 PHASE 3: Documentation & DX (NICE TO HAVE)**
- **Priority:** Team collaboration and polish
- **Timeline:** 0.5 development session
- **Goal:** Professional documentation and developer experience

Key tasks:
- Update README (Godab → Crescendo)
- Comprehensive API documentation
- Docker support and deployment guides
- See: `docs/PHASE_3_DOCUMENTATION.md`

### Documentation Reference

**📚 Available Documentation:**
- `docs/ANALYSIS.md` - Complete codebase analysis with specific findings
- `docs/CLEANUP_PLAN.md` - Detailed strategy and rationale
- `docs/PHASE_1_ARCHITECTURE.md` - Step-by-step refactoring tasks
- `docs/PHASE_2_TESTING.md` - Testing and quality improvements
- `docs/PHASE_3_DOCUMENTATION.md` - Documentation and developer experience

**🎯 Next Steps:**
1. **MUST DO:** Complete Phase 1 before any UI development
2. **SHOULD DO:** Complete Phase 2 for production readiness
3. **NICE TO HAVE:** Complete Phase 3 for team collaboration

### Why Cleanup is Critical for UI

**Without cleanup:**
- Adding UI complexity to 1,306-line main.go will make codebase unmaintainable
- Security issues will affect web-facing application
- Testing new features becomes difficult
- Team collaboration suffers

**After cleanup:**
- ✅ Modular architecture enables parallel UI development
- ✅ Clean API layer easy to integrate with frontend
- ✅ Comprehensive testing provides confidence
- ✅ Production-ready security and performance

## Development Priority
**PHASE 1 CLEANUP REQUIRED** before UI development. The modular architecture and security fixes are essential for maintainable frontend integration. UI development can begin safely after Phase 1 completion.

## React Frontend Development (Phase 4)

**Status:** Ready to begin (Backend transformation complete ✅)

### Overview
With the Crescendo backend fully functional, we now build the React frontend that will provide a modern web interface for music discovery and downloading. The backend runs on `localhost:8080` with complete REST API and WebSocket support.

### Technology Stack
- **React** with **TypeScript** for type safety
- **shadcn/ui** for modern, accessible UI components
- **Vite** for fast development and building
- **Tailwind CSS** for styling (included with shadcn/ui)

### Available Backend APIs
The following endpoints are ready for frontend integration:
- `GET /api/search?q=query&type=album|track|artist` - Search functionality
- `POST /api/downloads/album/:id` - Queue album download
- `POST /api/downloads/track/:id` - Queue track download
- `POST /api/downloads/artist/:id` - Queue artist discography
- `GET /api/downloads` - List active downloads
- `GET /api/downloads/:jobId` - Get download status
- `DELETE /api/downloads/:jobId` - Cancel download
- `GET /api/files` - List downloaded files
- `GET /api/files/:path/stream` - Serve downloaded files
- `WebSocket /ws/downloads/:jobId` - Real-time progress updates

### Development Approach: Thin Vertical Slices

Each slice delivers working functionality and builds incrementally toward the complete application.

#### **Slice 1: Foundation & Basic UI**
**Goal:** Working React app with layout and backend connection

**Tasks:**
- Set up React + TypeScript project with Vite
- Install and configure shadcn/ui component library
- Create basic application layout (header, sidebar, main content area)
- Add simple navigation structure
- Test backend connectivity with health check endpoint
- Configure CORS for development (backend already supports this)

**Deliverable:** Clean, responsive layout that successfully connects to backend

#### **Slice 2: Search & Discovery**
**Goal:** Users can search and view results

**Tasks:**
- Build search interface with input field and type selector
- Connect to `/api/search` endpoint with proper error handling
- Display search results in organized cards/lists
- Add loading states and error messages
- Implement result filtering by type (track/album/artist)
- Style results with shadcn/ui components

**Deliverable:** Fully functional search that displays DAB Music catalog

#### **Slice 3: Download Queue Management**
**Goal:** Users can queue downloads and see active jobs

**Tasks:**
- Add "Download" buttons to search result items
- Implement download queue interface using shadcn/ui components
- Connect to `/api/downloads` POST endpoints for queuing
- Display active downloads list with basic status
- Add cancel download functionality via DELETE endpoint
- Handle queue state management and updates

**Deliverable:** Complete download queuing system with status tracking

#### **Slice 4: Real-time Progress**
**Goal:** Live progress updates during downloads

**Tasks:**
- Implement WebSocket connection to `/ws/downloads/:jobId`
- Add progress bars showing download percentage
- Display real-time status updates and current file being downloaded
- Handle WebSocket connection lifecycle (connect, disconnect, reconnect)
- Update UI dynamically as progress changes
- Show completion notifications

**Deliverable:** Real-time download progress with live updates

#### **Slice 5: File Browser & Audio Player**
**Goal:** Browse and play downloaded content

**Tasks:**
- Connect to `/api/files` endpoint to display downloaded content
- Create file browser with folder navigation (Artist/Album structure)
- Implement basic audio player for track previewing
- Add file streaming via `/api/files/:path/stream` endpoint
- Create breadcrumb navigation for folder hierarchy
- Add file management features (delete, organize)

**Deliverable:** Complete file browser with audio playback

### Project Structure
```
crescendo-frontend/
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── ui/             # shadcn/ui components
│   │   ├── layout/         # Layout components
│   │   ├── search/         # Search-related components
│   │   ├── downloads/      # Download queue components
│   │   └── files/          # File browser components
│   ├── services/           # API integration
│   │   ├── api.ts          # REST API client
│   │   └── websocket.ts    # WebSocket client
│   ├── hooks/              # Custom React hooks
│   ├── types/              # TypeScript type definitions
│   └── utils/              # Helper functions
├── public/                 # Static assets
└── package.json           # Dependencies
```

### Success Criteria
1. ✅ Modern, responsive UI using shadcn/ui components
2. ✅ Complete search functionality with DAB Music integration
3. ✅ Download queue management with real-time progress
4. ✅ File browser with audio playback capabilities
5. ✅ Proper error handling and loading states
6. ✅ Type-safe TypeScript implementation
7. ✅ Cross-browser compatibility and accessibility

### Next Steps
Begin with **Slice 1** to establish the foundation, then proceed through each slice to build the complete Crescendo music downloader frontend.

## ✅ SLICE 4 COMPLETED: Real-time WebSocket Progress

### Current Implementation Status:
- **✅ WebSocket Integration**: Implemented real-time progress tracking
- **✅ Custom Hook**: `useDownloadProgress.ts` manages connections
- **✅ Connection Status**: Visual indicators for WebSocket state
- **✅ Progress Display**: Live progress bars with speed/ETA
- **✅ Completion Notifications**: Toast notifications for success/failure
- **⚠️ CONNECTION MANAGEMENT**: Needs improvement for production

### 🔧 CONNECTION MANAGEMENT IMPROVEMENT PLAN

#### Issue Identified:
Current hardcoded ports and connection handling can cause:
- Connection state confusion (showing disconnected when actually connected)
- Port conflicts when multiple instances run
- Poor error recovery when services restart
- Hardcoded URLs that don't adapt to environment changes

#### Proposed Solution:

**1. Environment-Based Configuration**
```typescript
// Create src/config/environment.ts
interface AppConfig {
  API_BASE_URL: string
  WS_BASE_URL: string
  HEALTH_CHECK_INTERVAL: number
  RECONNECT_ATTEMPTS: number
}

const config: AppConfig = {
  API_BASE_URL: process.env.VITE_API_URL || 'http://localhost:8080',
  WS_BASE_URL: process.env.VITE_WS_URL || 'ws://localhost:8080',
  HEALTH_CHECK_INTERVAL: 30000,
  RECONNECT_ATTEMPTS: 5
}
```

**2. Service Discovery & Health Checking**
```typescript
// Add to api.ts
async discoverBackend(): Promise<string> {
  const ports = [8080, 8081, 8082]
  for (const port of ports) {
    try {
      const response = await fetch(`http://localhost:${port}/health`)
      if (response.ok) return `http://localhost:${port}`
    } catch {}
  }
  throw new Error('No backend server found')
}
```

**3. Improved WebSocket Connection Management**
```typescript
// Update websocket.ts
async connectWithDiscovery(jobId: string, onProgress: ProgressCallback) {
  const baseUrl = await this.discoverWebSocketEndpoint()
  this.connect(jobId, onProgress, baseUrl)
}
```

**4. Production Deployment Script**
```bash
#!/bin/bash
# Kill existing processes
pkill -f "crescendo --server" || true
pkill -f "npm run dev" || true

# Find available port for backend
BACKEND_PORT=$(python3 -c "import socket; s=socket.socket(); s.bind(('', 0)); print(s.getsockname()[1]); s.close()")

# Start services
./crescendo --server --port $BACKEND_PORT &
cd crescendo-frontend/crescendo-frontend
VITE_API_URL="http://localhost:$BACKEND_PORT" npm run dev &
```

### 🎯 Next Steps for Connection Management:
1. Implement environment-based configuration
2. Add service discovery for flexible port handling
3. Improve WebSocket connection state tracking
4. Add proper health checking and error recovery
5. Create deployment scripts for clean startup/shutdown

### 🚀 Quick Start (Current Working State):
```bash
# Terminal 1: Start backend
./crescendo --server --port 8080

# Terminal 2: Start frontend
cd crescendo-frontend/crescendo-frontend
npm run dev

# Visit: http://localhost:5173
```

**The WebSocket real-time progress is fully functional** but would benefit from the connection management improvements above for production robustness.