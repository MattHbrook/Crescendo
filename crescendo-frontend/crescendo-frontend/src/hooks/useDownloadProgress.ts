import { useState, useEffect, useRef, useCallback } from 'react'
import { websocketService, type ProgressUpdate } from '@/services/websocket'
import { apiService } from '@/services/api'
import type { DownloadJob } from '@/types/api'

interface DownloadProgressState {
  [jobId: string]: {
    progress: number
    status: string
    currentFile?: string
    error?: string
    speed?: string
    eta?: string
  }
}

interface UseDownloadProgressReturn {
  progressData: DownloadProgressState
  connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error'
  connectToDownload: (jobId: string) => void
  disconnectFromDownload: (jobId: string) => void
  disconnectAll: () => void
}

export function useDownloadProgress(): UseDownloadProgressReturn {
  const [progressData, setProgressData] = useState<DownloadProgressState>({})
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected')
  const connectionsRef = useRef<Map<string, boolean>>(new Map())
  const apiStatusUnsubscribeRef = useRef<(() => void) | null>(null)

  const handleProgressUpdate = useCallback((jobId: string, update: ProgressUpdate) => {
    setProgressData(prev => ({
      ...prev,
      [jobId]: {
        progress: update.percentage,
        status: update.status,
        currentFile: update.currentFile,
        error: update.error,
        speed: update.speed,
        eta: update.eta
      }
    }))

    // Update connection status to connected when we receive data
    setConnectionStatus('connected')
  }, [])

  const connectToDownload = useCallback(async (jobId: string) => {
    if (connectionsRef.current.has(jobId)) {
      return // Already connected
    }

    // Check if API is connected first
    if (apiService.getConnectionStatus() !== 'connected') {
      console.log(`⏳ Waiting for API connection before connecting WebSocket for job ${jobId}`)
      setConnectionStatus('connecting')
      return
    }

    connectionsRef.current.set(jobId, true)
    setConnectionStatus('connecting')

    try {
      await websocketService.connect(jobId, (update) => handleProgressUpdate(jobId, update))
      setConnectionStatus('connected')
    } catch (error) {
      console.error(`Failed to connect to WebSocket for job ${jobId}:`, error)
      setConnectionStatus('error')
      connectionsRef.current.delete(jobId)
    }
  }, [handleProgressUpdate])

  const disconnectFromDownload = useCallback((jobId: string) => {
    connectionsRef.current.delete(jobId)
    websocketService.disconnect(jobId)

    // Remove progress data for this job
    setProgressData(prev => {
      const newData = { ...prev }
      delete newData[jobId]
      return newData
    })

    // Update connection status
    if (connectionsRef.current.size === 0) {
      setConnectionStatus('disconnected')
    }
  }, [])

  const disconnectAll = useCallback(() => {
    // Clear all connections
    connectionsRef.current.clear()

    // Disconnect all WebSockets
    websocketService.disconnect()

    // Reset state
    setProgressData({})
    setConnectionStatus('disconnected')
  }, [])

  // Monitor API connection status and auto-connect pending WebSockets
  useEffect(() => {
    const unsubscribe = apiService.onConnectionStatusChange((status) => {
      console.log(`🔗 API status changed to: ${status}`)

      if (status === 'connected') {
        // Try to connect any pending WebSocket connections
        connectionsRef.current.forEach((_, jobId) => {
          if (!websocketService.isConnected(jobId)) {
            connectToDownload(jobId)
          }
        })
      } else if (status === 'error') {
        setConnectionStatus('error')
      }
    })

    apiStatusUnsubscribeRef.current = unsubscribe

    return () => {
      if (apiStatusUnsubscribeRef.current) {
        apiStatusUnsubscribeRef.current()
      }
    }
  }, [connectToDownload])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnectAll()
    }
  }, [disconnectAll])

  return {
    progressData,
    connectionStatus,
    connectToDownload,
    disconnectFromDownload,
    disconnectAll
  }
}