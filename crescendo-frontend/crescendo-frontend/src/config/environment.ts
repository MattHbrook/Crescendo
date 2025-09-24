interface AppConfig {
  API_BASE_URL: string
  WS_BASE_URL: string
  HEALTH_CHECK_INTERVAL: number
  RECONNECT_ATTEMPTS: number
  RECONNECT_DELAY: number
  SERVICE_DISCOVERY_PORTS: number[]
}

const config: AppConfig = {
  API_BASE_URL: import.meta.env.VITE_API_URL || '',
  WS_BASE_URL: import.meta.env.VITE_WS_URL || '',
  HEALTH_CHECK_INTERVAL: 30000, // 30 seconds
  RECONNECT_ATTEMPTS: 5,
  RECONNECT_DELAY: 1000, // 1 second base delay
  SERVICE_DISCOVERY_PORTS: [8080, 8081, 8082, 3000, 3001] // Common ports to try
}

export default config
export type { AppConfig }