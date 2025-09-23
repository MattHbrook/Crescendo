import { useConnectionStatus } from '@/hooks/useConnectionStatus'
import { Wifi, WifiOff, RefreshCw } from 'lucide-react'

export function Header() {
  const { status, isChecking, checkConnection } = useConnectionStatus()

  return (
    <header style={{
      padding: '16px 24px',
      borderBottom: '1px solid #334155',
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      backgroundColor: '#1e293b'
    }}>
      <div style={{ flex: 1 }}>
        {/* Search will go here later */}
      </div>

      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px'
      }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: '6px',
          padding: '4px 8px',
          borderRadius: '4px',
          fontSize: '12px',
          fontWeight: '500',
          backgroundColor: status.connected ? '#10b981' : '#dc2626',
          color: '#ffffff'
        }}>
          {status.connected ? (
            <Wifi style={{ width: '12px', height: '12px' }} />
          ) : (
            <WifiOff style={{ width: '12px', height: '12px' }} />
          )}
          <span>
            {status.connected ? 'Connected' : 'Disconnected'}
          </span>
        </div>

        <button
          onClick={checkConnection}
          disabled={isChecking}
          style={{
            width: '32px',
            height: '32px',
            borderRadius: '4px',
            border: 'none',
            backgroundColor: 'transparent',
            cursor: isChecking ? 'default' : 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            opacity: isChecking ? 0.5 : 1
          }}
          onMouseEnter={(e) => {
            if (!isChecking) {
              e.currentTarget.style.backgroundColor = '#475569'
            }
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = 'transparent'
          }}
        >
          <RefreshCw
            style={{
              width: '16px',
              height: '16px',
              color: '#94a3b8',
              transform: isChecking ? 'rotate(360deg)' : 'none',
              transition: isChecking ? 'transform 1s linear infinite' : 'none'
            }}
          />
        </button>
      </div>
    </header>
  )
}