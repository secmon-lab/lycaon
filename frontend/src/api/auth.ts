import { userCache } from '../services/userCache';

export interface User {
  id: string
  name: string
  email: string
  slack_user_id: string
  avatar_url?: string
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
    // Clear cache before logout
    userCache.clearAll();
    
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
 * Get current authenticated user with caching
 */
export const getCurrentUser = async (): Promise<User> => {
  // Check cache first
  const cached = userCache.getCurrentUser();
  if (cached) {
    return cached;
  }

  try {
    const response = await fetch('/api/user/me', {
      credentials: 'include',
    })
    if (!response.ok) {
      throw new Error('Failed to get user')
    }
    const user = await response.json();
    
    // Cache the result
    userCache.setCurrentUser(user);
    
    return user;
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

/**
 * Get user by Slack user ID with caching
 */
export const getUserBySlackId = async (slackUserId: string): Promise<User | null> => {
  // Check cache first
  const cached = userCache.getUserBySlackId(slackUserId);
  if (cached) {
    return cached;
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
    userCache.setUserBySlackId(slackUserId, user);
    
    return user
  } catch (error) {
    console.error('Error fetching user by Slack ID:', error)
    return null
  }
}