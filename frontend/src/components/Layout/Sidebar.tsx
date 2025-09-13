import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../lib/utils';
import {
  LayoutDashboard,
  AlertCircle,
  Settings,
  ChevronLeft,
  Menu,
} from 'lucide-react';
import { Button } from '../ui/Button';

interface SidebarProps {
  isOpen: boolean;
  onToggle: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ isOpen, onToggle }) => {
  const location = useLocation();

  const menuItems = [
    {
      text: 'Dashboard',
      path: '/',
      icon: <LayoutDashboard className="h-5 w-5" />,
    },
    {
      text: 'Incidents',
      path: '/incidents',
      icon: <AlertCircle className="h-5 w-5" />,
    },
    {
      text: 'Settings',
      path: '/settings',
      icon: <Settings className="h-5 w-5" />,
    },
  ];

  const isActive = (path: string) => {
    return location.pathname === path || 
           (path !== '/' && location.pathname.startsWith(path));
  };

  return (
    <>
      {/* Mobile backdrop */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={onToggle}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed left-0 top-0 z-50 h-full bg-white border-r border-slate-200 transition-all duration-300",
          isOpen ? "w-64" : "w-0 lg:w-16"
        )}
      >
        <div className="flex h-full flex-col">
          {/* Header */}
          <div className="flex h-16 items-center justify-between px-4 border-b border-slate-200">
            <div className={cn(
              "flex items-center gap-3 transition-opacity duration-200",
              !isOpen && "lg:opacity-0"
            )}>
              <img 
                src="/logo.png" 
                alt="Lycaon" 
                className="h-8 w-8"
              />
              <span className="text-lg font-semibold text-slate-900">
                Lycaon
              </span>
            </div>
            <Button
              variant="ghost"
              size="icon"
              onClick={onToggle}
              className={cn(
                "transition-transform duration-200",
                !isOpen && "lg:rotate-180"
              )}
            >
              {isOpen ? (
                <ChevronLeft className="h-4 w-4" />
              ) : (
                <Menu className="h-4 w-4" />
              )}
            </Button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 overflow-y-auto p-4">
            <ul className="space-y-2">
              {menuItems.map((item) => (
                <li key={item.path}>
                  <Link
                    to={item.path}
                    className={cn(
                      "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all",
                      "hover:bg-slate-100",
                      isActive(item.path)
                        ? "bg-slate-100 text-slate-900"
                        : "text-slate-600 hover:text-slate-900",
                      !isOpen && "lg:justify-center"
                    )}
                  >
                    <span className={cn(
                      "transition-colors",
                      isActive(item.path) && "text-blue-600"
                    )}>
                      {item.icon}
                    </span>
                    <span className={cn(
                      "transition-opacity duration-200",
                      !isOpen && "lg:hidden"
                    )}>
                      {item.text}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </aside>
    </>
  );
};

export default Sidebar;