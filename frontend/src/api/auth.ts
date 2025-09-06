export interface User {
  id: string
  name: string
  email: string
  slack_user_id: string
}

/**
 * Initiate Slack OAuth login
 */
export const loginWithSlack = async () => {
  window.location.href = '/api/auth/login'
}

/**
 * Logout the current user
 */
export const logout = async () => {
  try {
    const response = await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
    })
    if (!response.ok) {
      throw new Error('Logout failed')
    }
    return true
  } catch (error) {
    console.error('Logout error:', error)
    throw error
  }
}

/**
 * Get current authenticated user
 */
export const getCurrentUser = async (): Promise<User> => {
  try {
    const response = await fetch('/api/user/me', {
      credentials: 'include',
    })
    if (!response.ok) {
      throw new Error('Failed to get user')
    }
    return await response.json()
  } catch (error) {
    throw error
  }
}

/**
 * Check if user is authenticated
 */
export const isAuthenticated = async (): Promise<boolean> => {
  try {
    await getCurrentUser()
    return true
  } catch {
    return false
  }
}