import { useState } from 'react'
import { loginWithSlack } from '../api/auth'

function Login() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSlackLogin = async () => {
    setLoading(true)
    setError(null)
    try {
      await loginWithSlack()
    } catch (err) {
      setError('Failed to initiate Slack login. Please try again.')
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-500 to-purple-600">
      <div className="card max-w-sm w-full mx-4 text-center">
        <img 
          src="/logo.png" 
          alt="Lycaon Logo"
          className="w-30 h-30 mb-4 mx-auto"
        />
        
        <h2 className="text-3xl font-bold mb-2 text-gray-800">
          Lycaon
        </h2>
        
        <p className="text-gray-600 mb-8">
          Slack-based Incident Management Service
        </p>

        {error && (
          <div className="p-3 mb-4 text-sm text-red-700 bg-red-50 border border-red-200 rounded">
            {error}
          </div>
        )}

        <button
          onClick={handleSlackLogin}
          disabled={loading}
          className={`btn-primary w-full flex items-center justify-center gap-2 ${
            loading ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90'
          }`}
        >
          <svg 
            width="20" 
            height="20" 
            viewBox="0 0 124 124" 
            fill="none" 
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M26.4 78.4C26.4 83.9 22 88.3 16.5 88.3C11 88.3 6.6 83.9 6.6 78.4C6.6 72.9 11 68.5 16.5 68.5H26.4V78.4Z" fill="white"/>
            <path d="M31.4 78.4C31.4 72.9 35.8 68.5 41.3 68.5C46.8 68.5 51.2 72.9 51.2 78.4V107.5C51.2 113 46.8 117.4 41.3 117.4C35.8 117.4 31.4 113 31.4 107.5V78.4Z" fill="white"/>
            <path d="M41.3 26.4C35.8 26.4 31.4 22 31.4 16.5C31.4 11 35.8 6.6 41.3 6.6C46.8 6.6 51.2 11 51.2 16.5V26.4H41.3Z" fill="white"/>
            <path d="M41.3 31.4C46.8 31.4 51.2 35.8 51.2 41.3C51.2 46.8 46.8 51.2 41.3 51.2H12.2C6.7 51.2 2.3 46.8 2.3 41.3C2.3 35.8 6.7 31.4 12.2 31.4H41.3Z" fill="white"/>
            <path d="M93.3 41.3C93.3 35.8 97.7 31.4 103.2 31.4C108.7 31.4 113.1 35.8 113.1 41.3C113.1 46.8 108.7 51.2 103.2 51.2H93.3V41.3Z" fill="white"/>
            <path d="M88.3 41.3C88.3 46.8 83.9 51.2 78.4 51.2C72.9 51.2 68.5 46.8 68.5 41.3V12.2C68.5 6.7 72.9 2.3 78.4 2.3C83.9 2.3 88.3 6.7 88.3 12.2V41.3Z" fill="white"/>
            <path d="M78.4 93.3C83.9 93.3 88.3 97.7 88.3 103.2C88.3 108.7 83.9 113.1 78.4 113.1C72.9 113.1 68.5 108.7 68.5 103.2V93.3H78.4Z" fill="white"/>
            <path d="M78.4 88.3C72.9 88.3 68.5 83.9 68.5 78.4C68.5 72.9 72.9 68.5 78.4 68.5H107.5C113 68.5 117.4 72.9 117.4 78.4C117.4 83.9 113 88.3 107.5 88.3H78.4Z" fill="white"/>
          </svg>
          {loading ? 'Redirecting...' : 'Sign in with Slack'}
        </button>

        <p className="mt-8 text-sm text-gray-600">
          By signing in, you agree to our terms of service
        </p>
      </div>
    </div>
  )
}

export default Login