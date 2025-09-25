import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { FolderOpen, Music, RefreshCw } from 'lucide-react'

export function FilesPage() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <h1 style={{
            fontSize: '24px',
            fontWeight: 'bold',
            color: '#f1f5f9',
            marginBottom: '8px'
          }}>Files</h1>
          <p style={{
            color: '#94a3b8',
            fontSize: '14px'
          }}>
            Browse and manage your downloaded music
          </p>
        </div>
        <button
          style={{
            padding: '8px 16px',
            backgroundColor: 'transparent',
            border: '1px solid #475569',
            borderRadius: '6px',
            color: '#f1f5f9',
            fontSize: '14px',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            gap: '8px'
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.backgroundColor = '#334155'
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = 'transparent'
          }}
        >
          <RefreshCw style={{ width: '16px', height: '16px' }} />
          Refresh
        </button>
      </div>

      <div style={{
        backgroundColor: '#1e293b',
        border: '1px solid #475569',
        borderRadius: '8px'
      }}>
        <div style={{
          padding: '24px 24px 0 24px',
          borderBottom: '1px solid #475569'
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            paddingBottom: '16px'
          }}>
            <FolderOpen style={{
              width: '20px',
              height: '20px',
              marginRight: '8px',
              color: '#f1f5f9'
            }} />
            <h2 style={{
              fontSize: '20px',
              fontWeight: 'bold',
              color: '#f1f5f9'
            }}>
              Music Library
            </h2>
          </div>
        </div>
        <div style={{ padding: '24px' }}>
          <div style={{
            textAlign: 'center',
            padding: '32px 0',
            color: '#94a3b8'
          }}>
            <Music style={{
              width: '48px',
              height: '48px',
              margin: '0 auto 16px',
              opacity: 0.5,
              color: '#94a3b8'
            }} />
            <p style={{
              fontSize: '16px',
              marginBottom: '8px',
              color: '#f1f5f9'
            }}>No music files found</p>
            <p style={{
              fontSize: '14px',
              color: '#94a3b8'
            }}>Download some music to see it here</p>
          </div>
        </div>
      </div>
    </div>
  )
}