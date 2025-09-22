import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { ApolloProvider } from '@apollo/client/react'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import IncidentList from './pages/IncidentList'
import IncidentDetail from './pages/IncidentDetail'
import Layout from './components/Layout/Layout'
import { getCurrentUser } from './api/auth'
import client from './apollo'

interface User {
  id: string
  name: string
  email: string
  slack_user_id: string
}


function App() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    checkAuth()
  }, [])

  const checkAuth = async () => {
    try {
      const userData = await getCurrentUser()
      setUser(userData)
    } catch (error) {
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen bg-gradient-to-br from-indigo-500 to-purple-600">
        <div className="text-white text-2xl">Loading...</div>
      </div>
    )
  }

  return (
    <ApolloProvider client={client}>
      <Router>
        <Routes>
          <Route 
            path="/login" 
            element={user ? <Navigate to="/" /> : <Login />} 
          />
          {user ? (
            <Route path="/" element={<Layout />}>
              <Route index element={<Dashboard user={user} setUser={setUser} />} />
              <Route path="incidents" element={<IncidentList />} />
              <Route path="incidents/:id" element={<IncidentDetail />} />
              <Route path="*" element={<Navigate to="/" />} />
            </Route>
          ) : (
            <Route path="*" element={<Navigate to="/login" />} />
          )}
        </Routes>
      </Router>
    </ApolloProvider>
  )
}

export default App