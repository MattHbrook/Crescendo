import { useState, useEffect } from 'react'
import { apiService } from '@/services/api'

export function SettingsPage() {
  const [downloadLocation, setDownloadLocation] = useState('')
  const [isLoading, setIsLoading] = useState(true)

  // Load current settings on mount
  useEffect(() => {
    loadCurrentSettings()
  }, [])

  const loadCurrentSettings = async () => {
    try {
      const settings = await apiService.getSettings()
      setDownloadLocation(settings.downloadLocation)
    } catch (error) {
      console.error('Failed to load settings:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleOpenFolder = () => {
    // Open the downloads folder in Finder/Explorer
    if (downloadLocation) {
      window.open(`file://${downloadLocation}`, '_blank')
    }
  }

  if (isLoading) {
    return (
      <div style={{
        padding: '24px',
        textAlign: 'center'
      }}>
        <div style={{ color: '#94a3b8' }}>Loading settings...</div>
      </div>
    )
  }

  return (
    <div style={{
      padding: '24px',
      maxWidth: '800px',
      margin: '0 auto'
    }}>
      <div style={{
        marginBottom: '32px'
      }}>
        <h1 style={{
          fontSize: '24px',
          fontWeight: 'bold',
          color: '#f1f5f9',
          marginBottom: '8px'
        }}>
          Settings
        </h1>
        <p style={{
          color: '#94a3b8',
          fontSize: '14px'
        }}>
          View your Crescendo configuration
        </p>
      </div>

      <div style={{
        backgroundColor: '#1e293b',
        borderRadius: '8px',
        padding: '24px',
        border: '1px solid #334155'
      }}>
        <h2 style={{
          fontSize: '18px',
          fontWeight: '600',
          color: '#f1f5f9',
          marginBottom: '16px',
          display: 'flex',
          alignItems: 'center',
          gap: '8px'
        }}>
          📁 Download Location
        </h2>

        <div style={{
          marginBottom: '16px'
        }}>
          <label style={{
            display: 'block',
            fontSize: '14px',
            fontWeight: '500',
            color: '#e2e8f0',
            marginBottom: '8px'
          }}>
            All downloads are saved to:
          </label>
          <div style={{
            padding: '12px',
            backgroundColor: '#0f172a',
            borderRadius: '6px',
            border: '1px solid #334155',
            fontSize: '14px',
            color: '#94a3b8',
            fontFamily: 'monospace',
            marginBottom: '16px'
          }}>
            {downloadLocation}
          </div>
          <button
            onClick={handleOpenFolder}
            style={{
              padding: '12px 20px',
              backgroundColor: '#3b82f6',
              border: 'none',
              borderRadius: '6px',
              color: '#ffffff',
              fontSize: '14px',
              fontWeight: '500',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '8px'
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = '#2563eb'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = '#3b82f6'
            }}
          >
            🗂️ Open Downloads Folder
          </button>
        </div>

        <div style={{
          padding: '16px',
          backgroundColor: '#0f172a',
          borderRadius: '6px',
          border: '1px solid #334155'
        }}>
          <h3 style={{
            fontSize: '14px',
            fontWeight: '600',
            color: '#e2e8f0',
            marginBottom: '8px'
          }}>
            ℹ️ About Download Location
          </h3>
          <p style={{
            fontSize: '13px',
            color: '#94a3b8',
            lineHeight: '1.5',
            marginBottom: '8px'
          }}>
            Crescendo uses a fixed download location to avoid confusion. All your music downloads are organized in the folder structure:
          </p>
          <div style={{
            fontSize: '12px',
            color: '#64748b',
            fontFamily: 'monospace',
            fontStyle: 'italic'
          }}>
            Artist/Album/Track.flac
          </div>
        </div>
      </div>
    </div>
  )
}