
export interface ProgressUpdate {
  percentage: number
  status: string
  currentFile?: string
  error?: string
}

export class WebSocketService {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private baseUrl: string

  constructor(baseUrl: string = 'ws://localhost:8080') {
    this.baseUrl = baseUrl
  }

  connect(jobId: string, onProgress: (update: ProgressUpdate) => void): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.close()
    }

    const wsUrl = `${this.baseUrl}/ws/downloads/${jobId}`
    this.ws = new WebSocket(wsUrl)

    this.ws.onopen = () => {
      console.log(`WebSocket connected for job ${jobId}`)
      this.reconnectAttempts = 0
    }

    this.ws.onmessage = (event) => {
      try {
        const update: ProgressUpdate = JSON.parse(event.data)
        onProgress(update)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    this.ws.onclose = (event) => {
      console.log(`WebSocket closed for job ${jobId}`, event.code, event.reason)

      // Attempt to reconnect if not a clean close
      if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
        setTimeout(() => {
          this.reconnectAttempts++
          console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})`)
          this.connect(jobId, onProgress)
        }, this.reconnectDelay * Math.pow(2, this.reconnectAttempts))
      }
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}

export const websocketService = new WebSocketService()
export default websocketService