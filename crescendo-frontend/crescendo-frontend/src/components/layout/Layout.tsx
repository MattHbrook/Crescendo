import { Outlet } from 'react-router-dom'
import { Header } from './Header'
import { AppSidebar } from './AppSidebar'

export function Layout() {
  return (
    <div style={{
      height: '100vh',
      display: 'flex',
      fontFamily: 'system-ui, -apple-system, sans-serif',
      backgroundColor: '#0f172a'
    }}>
      {/* Sidebar */}
      <div style={{
        width: '250px',
        backgroundColor: '#1e293b',
        borderRight: '1px solid #334155',
        display: 'flex',
        flexDirection: 'column'
      }}>
        <AppSidebar />
      </div>

      {/* Main content */}
      <div style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: '#0f172a'
      }}>
        <Header />
        <main style={{
          flex: 1,
          overflow: 'auto',
          padding: '24px'
        }}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}