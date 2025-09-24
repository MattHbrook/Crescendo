// API response types for DAB Music integration

export interface SearchResult {
  type: 'track' | 'album' | 'artist'
  id: string
  title: string
  artist?: string
  album?: string
  cover?: string
  duration?: number
}

export interface Track {
  id: string
  title: string
  artist: string
  album: string
  duration: number
  quality: number
}

export interface Album {
  id: string
  title: string
  artist: string
  cover?: string
  tracks: Track[]
  year?: number
}

export interface Artist {
  id: string
  name: string
  albums: Album[]
}

export interface DownloadJob {
  id: string
  type: 'track' | 'album' | 'artist'
  title: string
  artist: string
  status: 'queued' | 'downloading' | 'completed' | 'failed' | 'cancelled'
  progress: number
  currentFile?: string
  error?: string
  createdAt: string
}

export interface FileItem {
  path: string
  name: string
  type: 'file' | 'directory'
  size?: number
  modifiedAt: string
}

export interface ConnectionStatus {
  connected: boolean
  version?: string
  error?: string
}

// Backend API response types (what the server actually returns)
export interface BackendTrack {
  id: number
  title: string
  artist: string
  albumTitle: string
  albumCover?: string
  releaseDate?: string
  duration: number
}

export interface BackendAlbum {
  id: string
  title: string
  artist: string
  cover?: string
  releaseDate?: string
  tracks?: BackendTrack[]
}

export interface BackendSearchResponse {
  query: string
  results: {
    Tracks?: {
      tracks: BackendTrack[] | null
    }
    Albums?: {
      albums: BackendAlbum[] | null
    }
    Artists?: {
      artists: Array<{ id: number, name: string }> | null
    }
  }
  type: string
}