import type { SearchResult, Album, Artist, DownloadJob, FileItem, ConnectionStatus, BackendSearchResponse, BackendTrack, BackendAlbum } from '@/types/api'
import config from '@/config/environment'

class ApiService {
  private baseUrl: string = ''
  private healthCheckInterval?: NodeJS.Timeout
  private connectionStatus: 'connected' | 'disconnected' | 'error' = 'disconnected'
  private statusListeners: ((status: 'connected' | 'disconnected' | 'error') => void)[] = []

  constructor() {
    this.initializeConnection()
  }

  private async initializeConnection() {
    if (config.API_BASE_URL) {
      // Use configured URL if available
      this.baseUrl = config.API_BASE_URL
      await this.testConnection(this.baseUrl)
    } else {
      // Auto-discover backend
      await this.discoverBackend()
    }
    this.startHealthChecking()
  }

  async discoverBackend(): Promise<string> {
    console.log('🔍 Discovering backend server...')

    for (const port of config.SERVICE_DISCOVERY_PORTS) {
      const testUrl = `http://localhost:${port}`
      try {
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), 2000)

        const response = await fetch(`${testUrl}/health`, {
          method: 'GET',
          mode: 'cors',
          headers: {
            'Accept': 'application/json',
          },
          signal: controller.signal
        })

        clearTimeout(timeoutId)

        if (response.ok) {
          console.log(`✅ Found backend at ${testUrl}`)
          this.baseUrl = testUrl
          this.setConnectionStatus('connected')
          return testUrl
        }
      } catch (error) {
        console.log(`❌ No backend at ${testUrl}:`, error instanceof Error ? error.message : 'Unknown error')
      }
    }

    this.setConnectionStatus('error')
    throw new Error('No backend server found on any port')
  }

  private async testConnection(url: string): Promise<boolean> {
    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 2000)

      const response = await fetch(`${url}/health`, {
        method: 'GET',
        mode: 'cors',
        headers: {
          'Accept': 'application/json',
        },
        signal: controller.signal
      })

      clearTimeout(timeoutId)

      if (response.ok) {
        this.setConnectionStatus('connected')
        return true
      }
    } catch (error) {
      this.setConnectionStatus('error')
    }
    return false
  }

  private setConnectionStatus(status: 'connected' | 'disconnected' | 'error') {
    if (this.connectionStatus !== status) {
      this.connectionStatus = status
      console.log(`🔗 Connection status changed to: ${status}`)
      this.statusListeners.forEach(listener => listener(status))
    }
  }

  onConnectionStatusChange(listener: (status: 'connected' | 'disconnected' | 'error') => void) {
    this.statusListeners.push(listener)
    // Immediately call with current status
    listener(this.connectionStatus)

    // Return unsubscribe function
    return () => {
      const index = this.statusListeners.indexOf(listener)
      if (index > -1) {
        this.statusListeners.splice(index, 1)
      }
    }
  }

  private startHealthChecking() {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval)
    }

    this.healthCheckInterval = setInterval(async () => {
      if (this.baseUrl) {
        const isHealthy = await this.testConnection(this.baseUrl)
        if (!isHealthy && this.connectionStatus === 'connected') {
          console.log('🔄 Backend became unhealthy, attempting rediscovery...')
          try {
            await this.discoverBackend()
          } catch {
            this.setConnectionStatus('error')
          }
        }
      }
    }, config.HEALTH_CHECK_INTERVAL)
  }

  getConnectionStatus() {
    return this.connectionStatus
  }

  getBaseUrl() {
    return this.baseUrl
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    if (!this.baseUrl) {
      throw new Error('Backend not available - check connection')
    }

    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        mode: 'cors',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          ...options?.headers,
        },
        ...options,
      })

      if (!response.ok) {
        if (response.status >= 500) {
          this.setConnectionStatus('error')
        }
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      // Ensure we're connected if request succeeded
      if (this.connectionStatus !== 'connected') {
        this.setConnectionStatus('connected')
      }

      return await response.json()
    } catch (error) {
      console.error(`API request failed for ${endpoint}:`, error)

      // Network errors suggest connection issues
      if (error instanceof TypeError && error.message.includes('fetch')) {
        this.setConnectionStatus('error')
        // Try to rediscover backend
        try {
          await this.discoverBackend()
          // Retry the request once with new URL
          return this.request(endpoint, options)
        } catch {
          // Rediscovery failed, propagate original error
        }
      }

      throw error
    }
  }

  // Health check
  async checkHealth(): Promise<ConnectionStatus> {
    const status = this.getConnectionStatus()
    try {
      if (status === 'connected' && this.baseUrl) {
        const response = await this.request('/health')
        return { connected: true, version: response.version }
      }
    } catch (error) {
      // Fall through to return disconnected status
    }

    return {
      connected: false,
      error: status === 'error' ? 'Backend server not reachable' : 'Connecting to backend...'
    }
  }

  async checkConnection(): Promise<ConnectionStatus> {
    const status = this.getConnectionStatus()
    return {
      status: status === 'connected' ? 'connected' : 'disconnected',
      message: status === 'connected' ? 'Connection healthy' :
               status === 'error' ? 'Backend server not reachable' :
               'Connecting to backend...'
    }
  }

  // Search functionality
  async search(query: string, type: 'track' | 'album' | 'artist' = 'track'): Promise<SearchResult[]> {
    const response = await this.request<BackendSearchResponse>(`/api/search?q=${encodeURIComponent(query)}&type=${type}`)

    // Handle the actual backend response structure
    if (type === 'track' && response.results.Tracks?.tracks) {
      return response.results.Tracks.tracks.map(track => ({
        id: track.id.toString(),
        type: 'track' as const,
        title: track.title,
        artist: track.artist || 'Unknown Artist',
        album: track.albumTitle || 'Unknown Album',
        duration: track.duration,
        cover: track.albumCover
      }))
    } else if (type === 'album' && response.results.Albums?.albums) {
      return response.results.Albums.albums.map(album => ({
        id: album.id,
        type: 'album' as const,
        title: album.title,
        artist: album.artist || 'Unknown Artist',
        cover: album.cover
      }))
    } else if (type === 'artist' && response.results.Artists?.artists) {
      return response.results.Artists.artists.map(artist => ({
        id: artist.id.toString(),
        type: 'artist' as const,
        title: artist.name || 'Unknown Artist',
        artist: artist.name || 'Unknown Artist'
      }))
    }

    // Return empty array if no results
    return []
  }

  // Download management
  async queueTrackDownload(trackId: string): Promise<DownloadJob> {
    const response = await this.request<{ job: DownloadJob }>(`/api/downloads/track/${trackId}`, {
      method: 'POST'
    })
    return response.job
  }

  async queueAlbumDownload(albumId: string): Promise<DownloadJob> {
    const response = await this.request<{ job: DownloadJob }>(`/api/downloads/album/${albumId}`, {
      method: 'POST'
    })
    return response.job
  }

  async queueArtistDownload(artistId: string): Promise<DownloadJob> {
    const response = await this.request<{ job: DownloadJob }>(`/api/downloads/artist/${artistId}`, {
      method: 'POST'
    })
    return response.job
  }

  async getDownloads(): Promise<DownloadJob[]> {
    const response = await this.request<{ jobs: DownloadJob[] }>('/api/downloads')
    return response.jobs
  }

  async getDownloadStatus(jobId: string): Promise<DownloadJob> {
    return await this.request<DownloadJob>(`/api/downloads/${jobId}`)
  }

  async cancelDownload(jobId: string): Promise<void> {
    await this.request(`/api/downloads/${jobId}`, {
      method: 'DELETE'
    })
  }

  // Convenience methods for download operations (matching SearchPage expectations)
  async downloadTrack(trackId: string): Promise<{ jobId: string }> {
    const job = await this.queueTrackDownload(trackId)
    return { jobId: job.id }
  }

  async downloadAlbum(albumId: string): Promise<{ jobId: string }> {
    const job = await this.queueAlbumDownload(albumId)
    return { jobId: job.id }
  }

  // File management
  async getFiles(): Promise<FileItem[]> {
    const response = await this.request<{ files: FileItem[] }>('/api/files')
    return response.files
  }

  getFileStreamUrl(filePath: string): string {
    return `${this.baseUrl}/api/files/stream/${encodeURIComponent(filePath)}`
  }

  // Settings management
  async getSettings(): Promise<{ downloadLocation: string }> {
    return await this.request<{ downloadLocation: string }>('/api/settings')
  }

  async updateSettings(settings: { downloadLocation: string }): Promise<void> {
    await this.request('/api/settings', {
      method: 'POST',
      body: JSON.stringify(settings)
    })
  }

  // Cleanup method
  destroy() {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval)
    }
    this.statusListeners.length = 0
  }
}

export const apiService = new ApiService()
export default apiService