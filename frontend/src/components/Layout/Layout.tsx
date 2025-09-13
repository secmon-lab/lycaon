import React, { useState, useEffect } from 'react';
import { Outlet } from 'react-router-dom';
import { cn } from '../../lib/utils';
import Sidebar from './Sidebar';
import { Button } from '../ui/Button';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import * as Avatar from '@radix-ui/react-avatar';
import { getCurrentUser } from '../../api/auth';
import { LogOut, User, Menu } from 'lucide-react';

interface UserData {
  id: string;
  name: string;
  email: string;
  slack_user_id: string;
  avatar_url?: string;
}

// Cache for user data
const USER_CACHE_KEY = 'lycaon_user_cache';
const USER_CACHE_EXPIRY = 5 * 60 * 1000; // 5 minutes

const getUserFromCache = (): UserData | null => {
  try {
    const cached = localStorage.getItem(USER_CACHE_KEY);
    if (cached) {
      const { data, expiry } = JSON.parse(cached);
      if (Date.now() < expiry) {
        return data;
      }
      localStorage.removeItem(USER_CACHE_KEY);
    }
  } catch (error) {
    console.error('Error reading user cache:', error);
  }
  return null;
};

const setUserCache = (user: UserData) => {
  try {
    const cacheData = {
      data: user,
      expiry: Date.now() + USER_CACHE_EXPIRY,
    };
    localStorage.setItem(USER_CACHE_KEY, JSON.stringify(cacheData));
  } catch (error) {
    console.error('Error setting user cache:', error);
  }
};

const Layout: React.FC = () => {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [user, setUser] = useState<UserData | null>(null);
  const [userLoading, setUserLoading] = useState(true);

  useEffect(() => {
    loadUser();
  }, []);

  const loadUser = async () => {
    // Check cache first
    const cachedUser = getUserFromCache();
    if (cachedUser) {
      setUser(cachedUser);
      setUserLoading(false);
      return;
    }

    // Fetch from API
    try {
      const userData = await getCurrentUser();
      setUser(userData);
      setUserCache(userData);
    } catch (error) {
      console.error('Failed to load user:', error);
    } finally {
      setUserLoading(false);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem(USER_CACHE_KEY);
    window.location.href = '/api/auth/logout';
  };

  return (
    <div className="flex h-screen bg-slate-50">
      <Sidebar isOpen={sidebarOpen} onToggle={() => setSidebarOpen(!sidebarOpen)} />
      
      {/* Main content area */}
      <div className={cn(
        "flex-1 flex flex-col transition-all duration-300",
        sidebarOpen ? "lg:ml-64" : "lg:ml-16"
      )}>
        {/* Top bar */}
        <header className="sticky top-0 z-30 flex h-16 items-center gap-4 border-b border-slate-200 bg-white px-6">
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden"
            onClick={() => setSidebarOpen(!sidebarOpen)}
          >
            <Menu className="h-5 w-5" />
          </Button>

          <h1 className="flex-1 text-xl font-semibold text-slate-900">
            Incident Management
          </h1>

          {/* User menu */}
          <div className="flex items-center gap-4">
            {userLoading ? (
              <div className="h-8 w-8 animate-pulse rounded-full bg-slate-200" />
            ) : user ? (
              <DropdownMenu.Root>
                <DropdownMenu.Trigger asChild>
                  <button className="outline-none">
                    <Avatar.Root className="inline-flex h-9 w-9 select-none items-center justify-center overflow-hidden rounded-full align-middle">
                      {user.avatar_url ? (
                        <Avatar.Image
                          className="h-full w-full object-cover"
                          src={user.avatar_url}
                          alt={user.name}
                        />
                      ) : null}
                      <Avatar.Fallback className="flex h-full w-full items-center justify-center bg-gradient-to-br from-blue-500 to-purple-600 text-sm font-medium text-white">
                        {user.name.charAt(0).toUpperCase()}
                      </Avatar.Fallback>
                    </Avatar.Root>
                  </button>
                </DropdownMenu.Trigger>

                <DropdownMenu.Portal>
                  <DropdownMenu.Content
                    className="z-50 min-w-[220px] overflow-hidden rounded-lg bg-white p-1.5 shadow-lg border border-slate-200"
                    sideOffset={5}
                    align="end"
                  >
                    <div className="px-2 py-1.5">
                      <p className="text-sm font-medium text-slate-900">{user.name}</p>
                      <p className="text-xs text-slate-500">{user.email}</p>
                    </div>
                    
                    <DropdownMenu.Separator className="my-1 h-px bg-slate-200" />
                    
                    <DropdownMenu.Item
                      className="flex cursor-pointer select-none items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none hover:bg-slate-100"
                      disabled
                    >
                      <User className="h-4 w-4" />
                      <span>Profile</span>
                    </DropdownMenu.Item>
                    
                    <DropdownMenu.Item
                      className="flex cursor-pointer select-none items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none hover:bg-slate-100 text-red-600"
                      onSelect={handleLogout}
                    >
                      <LogOut className="h-4 w-4" />
                      <span>Logout</span>
                    </DropdownMenu.Item>
                  </DropdownMenu.Content>
                </DropdownMenu.Portal>
              </DropdownMenu.Root>
            ) : (
              <div className="h-9 w-9 rounded-full bg-slate-200" />
            )}
          </div>
        </header>

        {/* Main content */}
        <main className="flex-1 overflow-auto p-6">
          <div className="mx-auto max-w-7xl">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
};

export default Layout;