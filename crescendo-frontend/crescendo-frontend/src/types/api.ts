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