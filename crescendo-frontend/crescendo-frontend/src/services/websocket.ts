
import config from '@/config/environment'
import { apiService } from './api'

export interface ProgressUpdate {
  percentage: number
  status: string
  currentFile?: string
  error?: string
  speed?: string
  eta?: string
}

export class WebSocketService {
  private connections = new Map<string, WebSocket>()
  private reconnectAttempts = new Map<string, number>()
  private reconnectTimeouts = new Map<string, NodeJS.Timeout>()
  private maxReconnectAttempts = config.RECONNECT_ATTEMPTS
  private reconnectDelay = config.RECONNECT_DELAY
  private baseUrl: string = ''

  constructor() {
    // Will be set dynamically based on discovered backend
  }

  async connect(jobId: string, onProgress: (update: ProgressUpdate) => void): Promise<void> {
    // Close existing connection for this job
    this.disconnect(jobId)

    try {
      // Get the backend URL from API service
      const backendUrl = apiService.getBaseUrl()
      if (!backendUrl) {
        throw new Error('Backend URL not available')
      }

      this.baseUrl = backendUrl.replace('http', 'ws')
      const wsUrl = `${this.baseUrl}/api/ws/downloads/${jobId}`

      console.log(`🔌 Connecting WebSocket for job ${jobId} to ${wsUrl}`)

      const ws = new WebSocket(wsUrl)
      this.connections.set(jobId, ws)

      ws.onopen = () => {
        console.log(`✅ WebSocket connected for job ${jobId}`)
        this.reconnectAttempts.set(jobId, 0)
      }

      ws.onmessage = (event) => {
        try {
          const update: ProgressUpdate = JSON.parse(event.data)
          onProgress(update)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      ws.onclose = (event) => {
        console.log(`🔌 WebSocket closed for job ${jobId}`, event.code, event.reason)
        this.connections.delete(jobId)

        // Attempt to reconnect if not a clean close
        if (event.code !== 1000) {
          this.scheduleReconnect(jobId, onProgress)
        }
      }

      ws.onerror = (error) => {
        console.error(`❌ WebSocket error for job ${jobId}:`, error)
        this.scheduleReconnect(jobId, onProgress)
      }

    } catch (error) {
      console.error(`Failed to connect WebSocket for job ${jobId}:`, error)
      throw error
    }
  }

  private scheduleReconnect(jobId: string, onProgress: (update: ProgressUpdate) => void): void {
    const attempts = this.reconnectAttempts.get(jobId) || 0

    if (attempts >= this.maxReconnectAttempts) {
      console.log(`❌ Max reconnection attempts reached for job ${jobId}`)
      return
    }

    const delay = this.reconnectDelay * Math.pow(2, attempts)
    console.log(`🔄 Scheduling reconnect for job ${jobId} in ${delay}ms (attempt ${attempts + 1}/${this.maxReconnectAttempts})`)

    const timeoutId = setTimeout(async () => {
      this.reconnectTimeouts.delete(jobId)
      this.reconnectAttempts.set(jobId, attempts + 1)
      try {
        await this.connect(jobId, onProgress)
      } catch (error) {
        console.error(`Reconnection failed for job ${jobId}:`, error)
      }
    }, delay)

    this.reconnectTimeouts.set(jobId, timeoutId)
  }

  disconnect(jobId?: string): void {
    if (jobId) {
      // Disconnect specific job
      const ws = this.connections.get(jobId)
      if (ws) {
        ws.close(1000, 'Client disconnect')
        this.connections.delete(jobId)
      }

      // Clear reconnection attempts and timeouts
      this.reconnectAttempts.delete(jobId)
      const timeoutId = this.reconnectTimeouts.get(jobId)
      if (timeoutId) {
        clearTimeout(timeoutId)
        this.reconnectTimeouts.delete(jobId)
      }

      console.log(`🔌 Disconnected WebSocket for job ${jobId}`)
    } else {
      // Disconnect all connections
      console.log('🔌 Disconnecting all WebSocket connections')

      this.connections.forEach((ws, jobId) => {
        ws.close(1000, 'Client disconnect')
        console.log(`🔌 Disconnected WebSocket for job ${jobId}`)
      })

      this.connections.clear()
      this.reconnectAttempts.clear()

      // Clear all timeouts
      this.reconnectTimeouts.forEach(timeoutId => clearTimeout(timeoutId))
      this.reconnectTimeouts.clear()
    }
  }

  isConnected(jobId?: string): boolean {
    if (jobId) {
      const ws = this.connections.get(jobId)
      return ws?.readyState === WebSocket.OPEN
    }
    // Check if any connection is open
    for (const ws of this.connections.values()) {
      if (ws.readyState === WebSocket.OPEN) {
        return true
      }
    }
    return false
  }

  getActiveConnections(): string[] {
    const active: string[] = []
    this.connections.forEach((ws, jobId) => {
      if (ws.readyState === WebSocket.OPEN) {
        active.push(jobId)
      }
    })
    return active
  }
}

export const websocketService = new WebSocketService()
export default websocketService