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

// Cache for user data by ID
const USER_CACHE_BY_ID: Map<string, { data: User; expiry: number }> = new Map()
const USER_CACHE_DURATION = 5 * 60 * 1000 // 5 minutes

/**
 * Get user by Slack user ID with caching
 */
export const getUserBySlackId = async (slackUserId: string): Promise<User | null> => {
  // Check cache first
  const cached = USER_CACHE_BY_ID.get(slackUserId)
  if (cached && Date.now() < cached.expiry) {
    return cached.data
  }

  try {
    const response = await fetch(`/api/user/slack/${slackUserId}`, {
      credentials: 'include',
    })
    if (!response.ok) {
      if (response.status === 404) {
        return null
      }
      throw new Error('Failed to get user')
    }
    const user = await response.json()
    
    // Cache the result
    USER_CACHE_BY_ID.set(slackUserId, {
      data: user,
      expiry: Date.now() + USER_CACHE_DURATION
    })
    
    return user
  } catch (error) {
    console.error('Error fetching user by Slack ID:', error)
    return null
  }
}