import { Search } from 'lucide-react'

export function SearchPage() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div>
        <h1 style={{
          fontSize: '24px',
          fontWeight: 'bold',
          margin: '0 0 8px 0',
          color: '#f1f5f9'
        }}>
          Search Music
        </h1>
        <p style={{
          color: '#94a3b8',
          margin: '0',
          fontSize: '14px'
        }}>
          Search for tracks, albums, and artists to download
        </p>
      </div>

      <div style={{
        border: '1px solid #475569',
        borderRadius: '8px',
        backgroundColor: '#1e293b',
        padding: '24px'
      }}>
        <h2 style={{
          fontSize: '18px',
          fontWeight: '600',
          margin: '0 0 16px 0',
          color: '#f1f5f9'
        }}>
          Find Music
        </h2>

        <div style={{
          display: 'flex',
          gap: '12px'
        }}>
          <input
            type="text"
            placeholder="Search for tracks, albums, or artists..."
            style={{
              flex: 1,
              padding: '12px',
              border: '1px solid #64748b',
              borderRadius: '6px',
              fontSize: '14px',
              outline: 'none',
              backgroundColor: '#334155',
              color: '#f1f5f9'
            }}
            onFocus={(e) => {
              e.currentTarget.style.borderColor = '#3b82f6'
              e.currentTarget.style.boxShadow = '0 0 0 3px rgba(59, 130, 246, 0.1)'
            }}
            onBlur={(e) => {
              e.currentTarget.style.borderColor = '#64748b'
              e.currentTarget.style.boxShadow = 'none'
            }}
          />
          <button style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '12px 24px',
            backgroundColor: '#3b82f6',
            color: '#ffffff',
            border: 'none',
            borderRadius: '6px',
            fontSize: '14px',
            fontWeight: '500',
            cursor: 'pointer',
            transition: 'background-color 0.2s'
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.backgroundColor = '#2563eb'
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = '#3b82f6'
          }}
          >
            <Search style={{ width: '16px', height: '16px' }} />
            Search
          </button>
        </div>
      </div>

      <div style={{
        border: '1px solid #475569',
        borderRadius: '8px',
        backgroundColor: '#1e293b',
        padding: '48px 24px',
        textAlign: 'center' as const
      }}>
        <div style={{
          color: '#94a3b8',
          fontSize: '14px'
        }}>
          Enter a search term to find music
        </div>
      </div>
    </div>
  )
}