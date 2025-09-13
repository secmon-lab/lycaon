import { User } from '../api/auth';

/**
 * Unified user caching service
 * Combines localStorage for persistence and in-memory Map for quick access
 */

const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes
const CURRENT_USER_KEY = 'lycaon_current_user';
const USER_CACHE_KEY = 'lycaon_users_cache';

interface CacheEntry<T> {
  data: T;
  expiry: number;
}

class UserCacheService {
  // In-memory cache for quick access
  private memoryCache: Map<string, CacheEntry<User>> = new Map();
  private currentUserCache: CacheEntry<User> | null = null;

  constructor() {
    // Load existing cache from localStorage on initialization
    this.loadFromLocalStorage();
  }

  /**
   * Load cached data from localStorage into memory
   */
  private loadFromLocalStorage(): void {
    try {
      // Load current user
      const currentUserData = localStorage.getItem(CURRENT_USER_KEY);
      if (currentUserData) {
        const parsed = JSON.parse(currentUserData) as CacheEntry<User>;
        if (Date.now() < parsed.expiry) {
          this.currentUserCache = parsed;
        } else {
          localStorage.removeItem(CURRENT_USER_KEY);
        }
      }

      // Load users cache
      const usersData = localStorage.getItem(USER_CACHE_KEY);
      if (usersData) {
        const parsed = JSON.parse(usersData) as Record<string, CacheEntry<User>>;
        Object.entries(parsed).forEach(([key, entry]) => {
          if (Date.now() < entry.expiry) {
            this.memoryCache.set(key, entry);
          }
        });
        // Clean up expired entries
        this.persistUsersCache();
      }
    } catch (error) {
      console.error('Error loading cache from localStorage:', error);
      this.clearAll();
    }
  }

  /**
   * Persist users cache to localStorage
   */
  private persistUsersCache(): void {
    try {
      const cacheObject: Record<string, CacheEntry<User>> = {};
      this.memoryCache.forEach((value, key) => {
        if (Date.now() < value.expiry) {
          cacheObject[key] = value;
        }
      });
      
      if (Object.keys(cacheObject).length > 0) {
        localStorage.setItem(USER_CACHE_KEY, JSON.stringify(cacheObject));
      } else {
        localStorage.removeItem(USER_CACHE_KEY);
      }
    } catch (error) {
      console.error('Error persisting cache to localStorage:', error);
    }
  }

  /**
   * Get current authenticated user from cache
   */
  getCurrentUser(): User | null {
    if (this.currentUserCache && Date.now() < this.currentUserCache.expiry) {
      return this.currentUserCache.data;
    }
    this.currentUserCache = null;
    localStorage.removeItem(CURRENT_USER_KEY);
    return null;
  }

  /**
   * Set current authenticated user in cache
   */
  setCurrentUser(user: User): void {
    const entry: CacheEntry<User> = {
      data: user,
      expiry: Date.now() + CACHE_DURATION,
    };
    
    this.currentUserCache = entry;
    
    try {
      localStorage.setItem(CURRENT_USER_KEY, JSON.stringify(entry));
    } catch (error) {
      console.error('Error saving current user to localStorage:', error);
    }

    // Also add to users cache by slack_user_id
    if (user.slack_user_id) {
      this.setUserBySlackId(user.slack_user_id, user);
    }
  }

  /**
   * Get user by Slack user ID from cache
   */
  getUserBySlackId(slackUserId: string): User | null {
    const cached = this.memoryCache.get(slackUserId);
    if (cached && Date.now() < cached.expiry) {
      return cached.data;
    }
    
    // Remove expired entry
    if (cached) {
      this.memoryCache.delete(slackUserId);
      this.persistUsersCache();
    }
    
    return null;
  }

  /**
   * Set user by Slack user ID in cache
   */
  setUserBySlackId(slackUserId: string, user: User): void {
    const entry: CacheEntry<User> = {
      data: user,
      expiry: Date.now() + CACHE_DURATION,
    };
    
    this.memoryCache.set(slackUserId, entry);
    this.persistUsersCache();
  }

  /**
   * Clear current user cache
   */
  clearCurrentUser(): void {
    this.currentUserCache = null;
    localStorage.removeItem(CURRENT_USER_KEY);
  }

  /**
   * Clear all cached data
   */
  clearAll(): void {
    this.currentUserCache = null;
    this.memoryCache.clear();
    localStorage.removeItem(CURRENT_USER_KEY);
    localStorage.removeItem(USER_CACHE_KEY);
  }

  /**
   * Clean up expired entries
   */
  cleanup(): void {
    const now = Date.now();
    
    // Clean current user
    if (this.currentUserCache && now >= this.currentUserCache.expiry) {
      this.clearCurrentUser();
    }
    
    // Clean users cache
    let hasExpired = false;
    this.memoryCache.forEach((value, key) => {
      if (now >= value.expiry) {
        this.memoryCache.delete(key);
        hasExpired = true;
      }
    });
    
    if (hasExpired) {
      this.persistUsersCache();
    }
  }
}

// Export singleton instance
export const userCache = new UserCacheService();

// Run cleanup periodically (every minute)
setInterval(() => {
  userCache.cleanup();
}, 60 * 1000);