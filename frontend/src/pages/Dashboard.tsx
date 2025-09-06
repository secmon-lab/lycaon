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
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 py-4">
        <div className="container flex justify-between items-center">
          <div className="flex items-center gap-4">
            <h1 className="text-2xl flex items-center gap-2">
              <img 
                src="/logo.png" 
                alt="Lycaon Logo"
                className="w-8 h-8"
              />
              Lycaon
            </h1>
          </div>
          
          <div className="flex items-center gap-4">
            <span className="text-gray-600">
              {user.name}
            </span>
            <button
              onClick={handleLogout}
              disabled={loggingOut}
              className={`btn-secondary px-4 py-2 text-sm ${
                loggingOut ? 'opacity-50' : ''
              }`}
            >
              {loggingOut ? 'Logging out...' : 'Logout'}
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 py-8">
        <div className="container">
          {/* Welcome Card */}
          <div className="card mb-8">
            <h2 className="mb-4">Welcome back, {user.name}!</h2>
            <p className="text-gray-600 mb-4">
              You're successfully connected to Lycaon incident management system.
            </p>
            <div className="p-4 bg-gray-50 rounded-lg text-sm">
              <p><strong>User ID:</strong> {user.id}</p>
              <p><strong>Email:</strong> {user.email}</p>
              <p><strong>Slack User ID:</strong> {user.slack_user_id}</p>
            </div>
          </div>

          {/* Placeholder Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div className="card">
              <h3 className="mb-2">📊 Incidents Overview</h3>
              <p className="text-gray-600">
                View and manage active incidents
              </p>
              <div className="mt-8 p-8 bg-gray-50 rounded-lg text-center text-gray-600">
                Coming soon...
              </div>
            </div>

            <div className="card">
              <h3 className="mb-2">💬 Recent Messages</h3>
              <p className="text-gray-600">
                Latest messages from Slack channels
              </p>
              <div className="mt-8 p-8 bg-gray-50 rounded-lg text-center text-gray-600">
                Coming soon...
              </div>
            </div>

            <div className="card">
              <h3 className="mb-2">⚡ Quick Actions</h3>
              <p className="text-gray-600">
                Create and manage incidents quickly
              </p>
              <div className="mt-8 p-8 bg-gray-50 rounded-lg text-center text-gray-600">
                Coming soon...
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="bg-white border-t border-gray-200 py-4 mt-auto">
        <div className="container text-center text-gray-600 text-sm">
          Lycaon Incident Management © {new Date().getFullYear()}
        </div>
      </footer>
    </div>
  )
}

export default Dashboard