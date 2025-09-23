import type { SearchResult, Album, Artist, DownloadJob, FileItem, ConnectionStatus } from '@/types/api'

const API_BASE_URL = 'http://localhost:8080'

class ApiService {
  private baseUrl: string

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
        },
        ...options,
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      return await response.json()
    } catch (error) {
      console.error(`API request failed for ${endpoint}:`, error)
      throw error
    }
  }

  // Health check
  async checkHealth(): Promise<ConnectionStatus> {
    try {
      const response = await fetch(`${this.baseUrl}/health`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
      })

      if (response.ok) {
        const data = await response.json()
        return { connected: true, version: data.version }
      } else {
        return { connected: false, error: `HTTP ${response.status}` }
      }
    } catch (error) {
      return {
        connected: false,
        error: error instanceof Error ? error.message : 'Unknown error'
      }
    }
  }

  // Search
  async search(query: string, type?: 'track' | 'album' | 'artist'): Promise<SearchResult[]> {
    const params = new URLSearchParams({ q: query })
    if (type) params.append('type', type)

    return this.request<SearchResult[]>(`/api/search?${params}`)
  }

  // Get album details
  async getAlbum(albumId: string): Promise<Album> {
    return this.request<Album>(`/api/album/${albumId}`)
  }

  // Get artist discography
  async getArtistDiscography(artistId: string): Promise<Artist> {
    return this.request<Artist>(`/api/artist/${artistId}/discography`)
  }

  // Download operations
  async downloadAlbum(albumId: string): Promise<{ jobId: string }> {
    return this.request<{ jobId: string }>(`/api/downloads/album/${albumId}`, {
      method: 'POST',
    })
  }

  async downloadTrack(trackId: string): Promise<{ jobId: string }> {
    return this.request<{ jobId: string }>(`/api/downloads/track/${trackId}`, {
      method: 'POST',
    })
  }

  async downloadArtist(artistId: string): Promise<{ jobId: string }> {
    return this.request<{ jobId: string }>(`/api/downloads/artist/${artistId}`, {
      method: 'POST',
    })
  }

  // Download queue management
  async getDownloads(): Promise<DownloadJob[]> {
    return this.request<DownloadJob[]>('/api/downloads')
  }

  async getDownloadStatus(jobId: string): Promise<DownloadJob> {
    return this.request<DownloadJob>(`/api/downloads/${jobId}`)
  }

  async cancelDownload(jobId: string): Promise<void> {
    await this.request(`/api/downloads/${jobId}`, {
      method: 'DELETE',
    })
  }

  // File management
  async getFiles(path?: string): Promise<FileItem[]> {
    const params = path ? `?path=${encodeURIComponent(path)}` : ''
    return this.request<FileItem[]>(`/api/files${params}`)
  }

  getFileStreamUrl(path: string): string {
    return `${this.baseUrl}/api/files/${encodeURIComponent(path)}/stream`
  }
}

export const apiService = new ApiService()
export default apiService