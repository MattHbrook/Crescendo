import { useState, useEffect } from 'react'
import type { ConnectionStatus } from '@/types/api'
import { apiService } from '@/services/api'

export function useConnectionStatus() {
  const [status, setStatus] = useState<ConnectionStatus>({ connected: false })
  const [isChecking, setIsChecking] = useState(false)

  const checkConnection = async () => {
    setIsChecking(true)
    try {
      const result = await apiService.checkHealth()
      setStatus(result)
    } catch (error) {
      setStatus({
        connected: false,
        error: error instanceof Error ? error.message : 'Connection failed'
      })
    } finally {
      setIsChecking(false)
    }
  }

  useEffect(() => {
    // Initial check
    checkConnection()

    // Check every 30 seconds
    const interval = setInterval(checkConnection, 30000)

    return () => clearInterval(interval)
  }, [])

  return { status, isChecking, checkConnection }
}