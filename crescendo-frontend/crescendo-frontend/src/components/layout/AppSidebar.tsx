import { Search, Download, FolderOpen, Music, Settings } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'

const items = [
  {
    title: 'Search',
    url: '/search',
    icon: Search,
    description: 'Find music to download'
  },
  {
    title: 'Downloads',
    url: '/downloads',
    icon: Download,
    description: 'Manage download queue'
  },
  {
    title: 'Files',
    url: '/files',
    icon: FolderOpen,
    description: 'Browse downloaded music'
  },
  {
    title: 'Settings',
    url: '/settings',
    icon: Settings,
    description: 'Configure app preferences'
  },
]

export function AppSidebar() {
  const location = useLocation()

  return (
    <div style={{
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      padding: '20px'
    }}>
      {/* Logo */}
      <div style={{
        marginBottom: '24px',
        paddingBottom: '16px',
        borderBottom: '1px solid #475569'
      }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px'
        }}>
          <Music style={{ width: '24px', height: '24px', color: '#3b82f6' }} />
          <span style={{
            fontSize: '18px',
            fontWeight: 'bold',
            color: '#f1f5f9'
          }}>
            Crescendo
          </span>
        </div>
      </div>

      {/* Navigation */}
      <nav style={{ flex: 1 }}>
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '8px'
        }}>
          {items.map((item) => {
            const isActive = location.pathname === item.url
            const Icon = item.icon

            return (
              <Link
                key={item.title}
                to={item.url}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  padding: '12px 16px',
                  borderRadius: '6px',
                  textDecoration: 'none',
                  fontSize: '14px',
                  fontWeight: '500',
                  backgroundColor: isActive ? '#3b82f6' : 'transparent',
                  color: isActive ? '#ffffff' : '#94a3b8',
                  transition: 'all 0.2s'
                }}
                onMouseEnter={(e) => {
                  if (!isActive) {
                    e.currentTarget.style.backgroundColor = '#475569'
                    e.currentTarget.style.color = '#e2e8f0'
                  }
                }}
                onMouseLeave={(e) => {
                  if (!isActive) {
                    e.currentTarget.style.backgroundColor = 'transparent'
                    e.currentTarget.style.color = '#94a3b8'
                  }
                }}
              >
                <Icon style={{ width: '20px', height: '20px' }} />
                <span>{item.title}</span>
              </Link>
            )
          })}
        </div>
      </nav>
    </div>
  )
}