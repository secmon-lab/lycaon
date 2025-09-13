import React, { useState, useEffect } from 'react';
import { Outlet } from 'react-router-dom';
import {
  Box,
  AppBar,
  Toolbar,
  IconButton,
  Typography,
  CssBaseline,
  Avatar,
  Menu,
  MenuItem,
  CircularProgress,
} from '@mui/material';
import { Menu as MenuIcon, AccountCircle } from '@mui/icons-material';
import SideMenu from './SideMenu';
import { getCurrentUser } from '../../api/auth';

const drawerWidth = 240;

interface User {
  id: string;
  name: string;
  email: string;
  slack_user_id: string;
  avatar_url?: string;
}

// Cache for user data with expiry
const USER_CACHE_KEY = 'lycaon_user_cache';
const USER_CACHE_EXPIRY = 5 * 60 * 1000; // 5 minutes

const getUserFromCache = (): User | null => {
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

const setUserCache = (user: User) => {
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

const MainLayout: React.FC = () => {
  const [sideMenuOpen, setSideMenuOpen] = useState(true);
  const [user, setUser] = useState<User | null>(null);
  const [userLoading, setUserLoading] = useState(true);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

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

  const handleToggleSideMenu = () => {
    setSideMenuOpen(!sideMenuOpen);
  };

  const handleUserMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleUserMenuClose = () => {
    setAnchorEl(null);
  };

  const handleLogout = () => {
    localStorage.removeItem(USER_CACHE_KEY);
    window.location.href = '/api/auth/logout';
  };

  return (
    <Box sx={{ display: 'flex' }}>
      <CssBaseline />
      <AppBar
        position="fixed"
        sx={{
          width: sideMenuOpen ? `calc(100% - ${drawerWidth}px)` : '100%',
          ml: sideMenuOpen ? `${drawerWidth}px` : 0,
          transition: (theme) =>
            theme.transitions.create(['margin', 'width'], {
              easing: theme.transitions.easing.sharp,
              duration: theme.transitions.duration.leavingScreen,
            }),
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            aria-label="open drawer"
            onClick={handleToggleSideMenu}
            edge="start"
            sx={{ mr: 2, ...(sideMenuOpen && { display: 'none' }) }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
            Incident Management
          </Typography>
          
          {/* User Avatar Section */}
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            {userLoading ? (
              <CircularProgress size={24} color="inherit" />
            ) : user ? (
              <>
                <IconButton
                  onClick={handleUserMenuOpen}
                  sx={{ p: 0 }}
                  aria-label="user menu"
                >
                  {user.avatar_url ? (
                    <Avatar 
                      alt={user.name} 
                      src={user.avatar_url}
                      sx={{ width: 32, height: 32 }}
                    />
                  ) : (
                    <Avatar sx={{ width: 32, height: 32 }}>
                      {user.name.charAt(0).toUpperCase()}
                    </Avatar>
                  )}
                </IconButton>
                <Menu
                  anchorEl={anchorEl}
                  open={Boolean(anchorEl)}
                  onClose={handleUserMenuClose}
                  anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'right',
                  }}
                  transformOrigin={{
                    vertical: 'top',
                    horizontal: 'right',
                  }}
                >
                  <MenuItem disabled>
                    <Box>
                      <Typography variant="body2" fontWeight="bold">
                        {user.name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {user.email}
                      </Typography>
                    </Box>
                  </MenuItem>
                  <MenuItem onClick={handleLogout}>Logout</MenuItem>
                </Menu>
              </>
            ) : (
              <IconButton color="inherit">
                <AccountCircle />
              </IconButton>
            )}
          </Box>
        </Toolbar>
      </AppBar>
      <SideMenu open={sideMenuOpen} onToggle={handleToggleSideMenu} />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          bgcolor: 'background.default',
          p: 3,
          width: sideMenuOpen ? `calc(100% - ${drawerWidth}px)` : '100%',
          ml: sideMenuOpen ? `${drawerWidth}px` : 0,
          transition: (theme) =>
            theme.transitions.create(['margin', 'width'], {
              easing: theme.transitions.easing.sharp,
              duration: theme.transitions.duration.leavingScreen,
            }),
        }}
      >
        <Toolbar />
        <Outlet />
      </Box>
    </Box>
  );
};

export default MainLayout;