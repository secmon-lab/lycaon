import React, { useState, useEffect, useRef } from 'react';
import { useQuery } from '@apollo/client/react';
import { ChevronDown, X, User, Search } from 'lucide-react';
import { GET_CHANNEL_MEMBERS } from '../../graphql/queries';

interface User {
  id: string;
  slackUserId: string;
  name: string;
  realName?: string;
  displayName?: string;
  email?: string;
  avatarUrl?: string;
}

interface AssigneeSelectorProps {
  channelId: string;
  selectedUserId?: string;
  onAssigneeChange: (userId: string | undefined) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

const AssigneeSelector: React.FC<AssigneeSelectorProps> = ({
  channelId,
  selectedUserId,
  onAssigneeChange,
  placeholder = "Select assignee...",
  disabled = false,
  className = ""
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // GraphQL query to get channel members
  const { data, loading, error } = useQuery(GET_CHANNEL_MEMBERS, {
    variables: { channelId },
    skip: !channelId,
  });

  const users: User[] = data?.channelMembers || [];

  // Find selected user
  const selectedUser = users.find(user => user.id === selectedUserId);

  // Filter users based on search query
  const filteredUsers = users.filter(user => {
    if (!searchQuery) return true;
    const query = searchQuery.toLowerCase();
    return (
      user.name.toLowerCase().includes(query) ||
      user.realName?.toLowerCase().includes(query) ||
      user.displayName?.toLowerCase().includes(query) ||
      user.email?.toLowerCase().includes(query)
    );
  });

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setSearchQuery('');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Focus search input when dropdown opens
  useEffect(() => {
    if (isOpen && searchInputRef.current) {
      searchInputRef.current.focus();
    }
  }, [isOpen]);

  const handleToggle = () => {
    if (!disabled) {
      setIsOpen(!isOpen);
      setSearchQuery('');
    }
  };

  const handleUserSelect = (user: User) => {
    onAssigneeChange(user.id);
    setIsOpen(false);
    setSearchQuery('');
  };

  const handleClearAssignment = () => {
    onAssigneeChange(undefined);
    setIsOpen(false);
    setSearchQuery('');
  };

  const getUserDisplayName = (user: User) => {
    return user.displayName || user.realName || user.name;
  };

  const renderUserOption = (user: User) => (
    <div
      key={user.id}
      onClick={() => handleUserSelect(user)}
      className="flex items-center gap-3 px-3 py-2 hover:bg-gray-50 cursor-pointer transition-colors"
    >
      {user.avatarUrl ? (
        <img
          src={user.avatarUrl}
          alt={getUserDisplayName(user)}
          className="w-8 h-8 rounded-full"
        />
      ) : (
        <div className="w-8 h-8 bg-gray-200 rounded-full flex items-center justify-center">
          <User className="w-4 h-4 text-gray-500" />
        </div>
      )}
      <div className="flex-1 min-w-0">
        <div className="font-medium text-gray-900 truncate">
          {getUserDisplayName(user)}
        </div>
        {user.email && (
          <div className="text-sm text-gray-500 truncate">
            {user.email}
          </div>
        )}
      </div>
    </div>
  );

  return (
    <div className={`relative ${className}`} ref={dropdownRef}>
      {/* Main selector button */}
      <button
        type="button"
        onClick={handleToggle}
        disabled={disabled}
        className={`
          w-full flex items-center justify-between px-3 py-2 bg-white border border-gray-300 rounded-md shadow-sm
          ${disabled
            ? 'bg-gray-50 cursor-not-allowed text-gray-400'
            : 'hover:border-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500'
          }
          ${error ? 'border-red-300' : ''}
        `}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          {selectedUser ? (
            <>
              {selectedUser.avatarUrl ? (
                <img
                  src={selectedUser.avatarUrl}
                  alt={getUserDisplayName(selectedUser)}
                  className="w-6 h-6 rounded-full"
                />
              ) : (
                <div className="w-6 h-6 bg-gray-200 rounded-full flex items-center justify-center">
                  <User className="w-3 h-3 text-gray-500" />
                </div>
              )}
              <span className="text-gray-900 truncate">
                {getUserDisplayName(selectedUser)}
              </span>
              {!disabled && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleClearAssignment();
                  }}
                  className="ml-auto p-1 hover:bg-gray-100 rounded transition-colors"
                >
                  <X className="w-4 h-4 text-gray-400" />
                </button>
              )}
            </>
          ) : (
            <span className="text-gray-500">{placeholder}</span>
          )}
        </div>
        {!selectedUser && (
          <ChevronDown className={`w-4 h-4 text-gray-400 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
        )}
      </button>

      {/* Error message */}
      {error && (
        <div className="mt-1 text-sm text-red-600">
          Failed to load channel members
        </div>
      )}

      {/* Dropdown menu */}
      {isOpen && (
        <div className="absolute z-10 mt-1 w-full bg-white border border-gray-300 rounded-md shadow-lg max-h-60 overflow-hidden">
          {/* Search input */}
          <div className="p-2 border-b border-gray-200">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search users..."
                className="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
            </div>
          </div>

          {/* Options list */}
          <div className="max-h-48 overflow-y-auto">
            {loading ? (
              <div className="px-3 py-2 text-sm text-gray-500">Loading...</div>
            ) : filteredUsers.length === 0 ? (
              <div className="px-3 py-2 text-sm text-gray-500">
                {searchQuery ? 'No users found' : 'No channel members available'}
              </div>
            ) : (
              <>
                {/* Clear assignment option */}
                {selectedUser && (
                  <>
                    <div
                      onClick={handleClearAssignment}
                      className="flex items-center gap-3 px-3 py-2 hover:bg-gray-50 cursor-pointer transition-colors border-b border-gray-100"
                    >
                      <div className="w-8 h-8 bg-gray-100 rounded-full flex items-center justify-center">
                        <X className="w-4 h-4 text-gray-400" />
                      </div>
                      <span className="text-gray-500 italic">Clear assignment</span>
                    </div>
                  </>
                )}

                {/* User options */}
                {filteredUsers.map(renderUserOption)}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default AssigneeSelector;