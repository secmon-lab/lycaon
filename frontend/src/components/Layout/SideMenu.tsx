import React, { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Drawer,
  Toolbar,
  Divider,
  IconButton,
  Box,
  Typography,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Menu as MenuIcon,
  ChevronLeft as ChevronLeftIcon,
  Dashboard as DashboardIcon,
  Description as IncidentIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';

const drawerWidth = 240;

interface MenuItem {
  text: string;
  path: string;
  icon: React.ReactNode;
  children?: MenuItem[];
}

const menuItems: MenuItem[] = [
  {
    text: 'Dashboard',
    path: '/',
    icon: <DashboardIcon />,
  },
  {
    text: 'Incidents',
    path: '/incidents',
    icon: <IncidentIcon />,
  },
  {
    text: 'Settings',
    path: '/settings',
    icon: <SettingsIcon />,
  },
];

interface SideMenuProps {
  open: boolean;
  onToggle: () => void;
}

const SideMenu: React.FC<SideMenuProps> = ({ open, onToggle }) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const location = useLocation();
  const [expandedItems, setExpandedItems] = useState<string[]>([]);

  const handleToggleExpand = (text: string) => {
    setExpandedItems((prev) =>
      prev.includes(text)
        ? prev.filter((item) => item !== text)
        : [...prev, text]
    );
  };

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  const renderMenuItem = (item: MenuItem, depth = 0) => {
    const hasChildren = item.children && item.children.length > 0;
    const isExpanded = expandedItems.includes(item.text);

    return (
      <React.Fragment key={item.path}>
        <ListItem disablePadding sx={{ pl: depth * 2 }}>
          <ListItemButton
            component={hasChildren ? 'div' : Link}
            to={hasChildren ? undefined : item.path}
            selected={isActive(item.path)}
            onClick={() => hasChildren && handleToggleExpand(item.text)}
          >
            {item.icon && <ListItemIcon>{item.icon}</ListItemIcon>}
            <ListItemText primary={item.text} />
            {hasChildren && (
              <IconButton size="small">
                {isExpanded ? <ChevronLeftIcon /> : <MenuIcon />}
              </IconButton>
            )}
          </ListItemButton>
        </ListItem>
        {hasChildren && isExpanded && (
          <List component="div" disablePadding>
            {item.children?.map((child) => renderMenuItem(child, depth + 1))}
          </List>
        )}
      </React.Fragment>
    );
  };

  const drawerContent = (
    <Box>
      <Toolbar>
        <Box sx={{ display: 'flex', alignItems: 'center', width: '100%' }}>
          <Box sx={{ flexGrow: 1, display: 'flex', alignItems: 'center' }}>
            <img 
              src="/logo.png" 
              alt="Lycaon" 
              style={{ height: '32px', marginRight: '12px' }} 
            />
            <Typography variant="h6" component="h2" sx={{ fontWeight: 'bold' }}>
              Lycaon
            </Typography>
          </Box>
          <IconButton onClick={onToggle}>
            <ChevronLeftIcon />
          </IconButton>
        </Box>
      </Toolbar>
      <Divider />
      <List>
        {menuItems.map((item) => renderMenuItem(item))}
      </List>
    </Box>
  );

  return (
    <Drawer
      variant={isMobile ? 'temporary' : 'persistent'}
      anchor="left"
      open={open}
      onClose={onToggle}
      sx={{
        width: open ? drawerWidth : 0,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: drawerWidth,
          boxSizing: 'border-box',
        },
      }}
    >
      {drawerContent}
    </Drawer>
  );
};

export default SideMenu;