export function TestLayout() {
  return (
    <div style={{
      height: '100vh',
      display: 'flex',
      fontFamily: 'system-ui',
      backgroundColor: '#fafafa'
    }}>
      {/* Sidebar */}
      <div style={{
        width: '250px',
        backgroundColor: '#ffffff',
        borderRight: '1px solid #e5e5e5',
        padding: '20px',
        display: 'flex',
        flexDirection: 'column'
      }}>
        <h2 style={{ margin: '0 0 20px 0', fontSize: '18px', fontWeight: 'bold' }}>
          🎵 Crescendo
        </h2>

        <nav style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <a href="#" style={{
            padding: '12px 16px',
            backgroundColor: '#2563eb',
            color: 'white',
            textDecoration: 'none',
            borderRadius: '6px',
            fontSize: '14px'
          }}>
            🔍 Search
          </a>
          <a href="#" style={{
            padding: '12px 16px',
            backgroundColor: 'transparent',
            color: '#6b7280',
            textDecoration: 'none',
            borderRadius: '6px',
            fontSize: '14px'
          }}>
            ⬇️ Downloads
          </a>
          <a href="#" style={{
            padding: '12px 16px',
            backgroundColor: 'transparent',
            color: '#6b7280',
            textDecoration: 'none',
            borderRadius: '6px',
            fontSize: '14px'
          }}>
            📁 Files
          </a>
        </nav>
      </div>

      {/* Main Content */}
      <div style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: '#ffffff'
      }}>
        {/* Header */}
        <header style={{
          padding: '16px 24px',
          borderBottom: '1px solid #e5e5e5',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <div></div>
          <div style={{
            padding: '4px 8px',
            backgroundColor: '#dc2626',
            color: 'white',
            borderRadius: '4px',
            fontSize: '12px'
          }}>
            📡 Disconnected
          </div>
        </header>

        {/* Page Content */}
        <main style={{
          flex: 1,
          padding: '24px',
          overflow: 'auto'
        }}>
          <h1 style={{ fontSize: '24px', fontWeight: 'bold', margin: '0 0 8px 0' }}>
            Search Music
          </h1>
          <p style={{ color: '#6b7280', margin: '0 0 24px 0' }}>
            Search for tracks, albums, and artists to download
          </p>

          <div style={{
            border: '1px solid #e5e5e5',
            borderRadius: '8px',
            padding: '24px',
            backgroundColor: '#ffffff'
          }}>
            <div style={{
              display: 'flex',
              gap: '12px',
              marginBottom: '16px'
            }}>
              <input
                type="text"
                placeholder="Search for tracks, albums, or artists..."
                style={{
                  flex: 1,
                  padding: '12px',
                  border: '1px solid #d1d5db',
                  borderRadius: '6px',
                  fontSize: '14px'
                }}
              />
              <button style={{
                padding: '12px 24px',
                backgroundColor: '#2563eb',
                color: 'white',
                border: 'none',
                borderRadius: '6px',
                fontSize: '14px',
                cursor: 'pointer'
              }}>
                🔍 Search
              </button>
            </div>

            <div style={{
              textAlign: 'center',
              color: '#6b7280',
              padding: '40px 0'
            }}>
              Enter a search term to find music
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}