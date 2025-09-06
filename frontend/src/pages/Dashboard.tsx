import { useState } from 'react'
import { logout } from '../api/auth'

interface User {
  id: string
  name: string
  email: string
  slack_user_id: string
}

interface DashboardProps {
  user: User
  setUser: (user: User | null) => void
}

function Dashboard({ user, setUser }: DashboardProps) {
  const [loggingOut, setLoggingOut] = useState(false)

  const handleLogout = async () => {
    setLoggingOut(true)
    try {
      await logout()
      setUser(null)
      window.location.href = '/login'
    } catch (error) {
      console.error('Logout failed:', error)
      setLoggingOut(false)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <header style={{
        background: 'white',
        borderBottom: '1px solid #e2e8f0',
        padding: '1rem 0',
      }}>
        <div className="container" style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
            <h1 style={{ fontSize: '1.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <img 
                src="/logo.png" 
                alt="Lycaon Logo"
                style={{
                  width: '32px',
                  height: '32px',
                }}
              />
              Lycaon
            </h1>
          </div>
          
          <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
            <span style={{ color: 'var(--text-secondary)' }}>
              {user.name}
            </span>
            <button
              onClick={handleLogout}
              disabled={loggingOut}
              className="btn-secondary"
              style={{
                padding: '0.5rem 1rem',
                fontSize: '0.875rem',
                opacity: loggingOut ? 0.5 : 1,
              }}
            >
              {loggingOut ? 'Logging out...' : 'Logout'}
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main style={{ flex: 1, padding: '2rem 0' }}>
        <div className="container">
          {/* Welcome Card */}
          <div className="card" style={{ marginBottom: '2rem' }}>
            <h2 style={{ marginBottom: '1rem' }}>Welcome back, {user.name}!</h2>
            <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
              You're successfully connected to Lycaon incident management system.
            </p>
            <div style={{
              padding: '1rem',
              background: 'var(--background)',
              borderRadius: '0.5rem',
              fontSize: '0.875rem',
            }}>
              <p><strong>User ID:</strong> {user.id}</p>
              <p><strong>Email:</strong> {user.email}</p>
              <p><strong>Slack User ID:</strong> {user.slack_user_id}</p>
            </div>
          </div>

          {/* Placeholder Cards */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
            gap: '1.5rem',
          }}>
            <div className="card">
              <h3 style={{ marginBottom: '0.5rem' }}>📊 Incidents Overview</h3>
              <p style={{ color: 'var(--text-secondary)' }}>
                View and manage active incidents
              </p>
              <div style={{
                marginTop: '2rem',
                padding: '2rem',
                background: 'var(--background)',
                borderRadius: '0.5rem',
                textAlign: 'center',
                color: 'var(--text-secondary)',
              }}>
                Coming soon...
              </div>
            </div>

            <div className="card">
              <h3 style={{ marginBottom: '0.5rem' }}>💬 Recent Messages</h3>
              <p style={{ color: 'var(--text-secondary)' }}>
                Latest messages from Slack channels
              </p>
              <div style={{
                marginTop: '2rem',
                padding: '2rem',
                background: 'var(--background)',
                borderRadius: '0.5rem',
                textAlign: 'center',
                color: 'var(--text-secondary)',
              }}>
                Coming soon...
              </div>
            </div>

            <div className="card">
              <h3 style={{ marginBottom: '0.5rem' }}>⚡ Quick Actions</h3>
              <p style={{ color: 'var(--text-secondary)' }}>
                Create and manage incidents quickly
              </p>
              <div style={{
                marginTop: '2rem',
                padding: '2rem',
                background: 'var(--background)',
                borderRadius: '0.5rem',
                textAlign: 'center',
                color: 'var(--text-secondary)',
              }}>
                Coming soon...
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer style={{
        background: 'white',
        borderTop: '1px solid #e2e8f0',
        padding: '1rem 0',
        marginTop: 'auto',
      }}>
        <div className="container" style={{
          textAlign: 'center',
          color: 'var(--text-secondary)',
          fontSize: '0.875rem',
        }}>
          Lycaon Incident Management © {new Date().getFullYear()}
        </div>
      </footer>
    </div>
  )
}

export default Dashboard