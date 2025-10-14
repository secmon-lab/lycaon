import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { useQuery } from '@apollo/client/react';
import { useNavigate } from 'react-router-dom';
import { format } from 'date-fns';
import { GET_INCIDENTS } from '../graphql/queries';
import { Button } from '../components/ui/Button';
import { IncidentStatus, StatusHistory, Task } from '../types/incident';
import StatusBadge from '../components/IncidentList/StatusBadge';
import SeverityBadge from '../components/common/SeverityBadge';
import TestBadge from '../components/common/TestBadge';
import SlackChannelLink from '../components/common/SlackChannelLink';
import { StatCard } from '../components/IncidentList/StatCard';
import { useIncidentStats } from '../hooks/useIncidentStats';
import {
  AlertCircle,
  RefreshCw,
  Clock,
  User,
  Plus,
  ChevronUp,
  ChevronDown,
  Search,
  X,
  Check,
  TrendingUp,
  Timer,
  Lock,
} from 'lucide-react';

interface User {
  id: string;
  slackUserId: string;
  name: string;
  realName: string;
  displayName: string;
  email: string;
  avatarUrl: string;
}

interface Incident {
  id: string;
  channelId: string;
  channelName: string;
  title: string;
  description: string;
  categoryId: string;
  categoryName: string;
  severityId: string;
  severityName: string;
  severityLevel: number;
  assetIds: string[];
  assetNames: string[];
  status: IncidentStatus;
  lead: string;
  leadUser?: User;
  originChannelId: string;
  originChannelName: string;
  teamId?: string;
  createdBy: string;
  createdByUser?: User;
  createdAt: string;
  updatedAt: string;
  private: boolean;
  viewerCanAccess: boolean;
  isTest: boolean;
  statusHistories: StatusHistory[];
  tasks: Task[];
}

interface IncidentEdge {
  node: Incident;
  cursor: string;
}

interface IncidentsData {
  incidents: {
    edges: IncidentEdge[];
    pageInfo: {
      hasNextPage: boolean;
      hasPreviousPage: boolean;
      startCursor: string | null;
      endCursor: string | null;
    };
    totalCount: number;
  };
}

const IncidentList: React.FC = () => {
  const navigate = useNavigate();

  // State management
  const [searchInput, setSearchInput] = useState('');
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<Set<IncidentStatus>>(new Set());
  const [severityFilter, setSeverityFilter] = useState<Set<number>>(new Set());
  const [dateRange, setDateRange] = useState<{ start: Date | null; end: Date | null }>({
    start: null,
    end: null,
  });
  const [sortColumn, setSortColumn] = useState<string>('createdAt');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
  const [currentPage, setCurrentPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [statusDropdownOpen, setStatusDropdownOpen] = useState(false);
  const [severityDropdownOpen, setSeverityDropdownOpen] = useState(false);

  // Fetch all incidents
  const { loading, error, data, refetch } = useQuery<IncidentsData>(
    GET_INCIDENTS,
    {
      variables: {
        first: 10000,
        after: null,
      },
      fetchPolicy: 'cache-and-network',
      notifyOnNetworkStatusChange: true,
    }
  );

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setSearchText(searchInput);
      setCurrentPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput]);

  const incidents = useMemo(() => {
    return data?.incidents?.edges?.map((edge: IncidentEdge) => edge.node) || [];
  }, [data?.incidents?.edges]);

  // Calculate statistics
  const stats = useIncidentStats(incidents);

  // Helper functions
  const matchesSearchText = (incident: Incident, search: string): boolean => {
    const lowerSearch = search.toLowerCase();
    return (
      incident.id.toLowerCase().includes(lowerSearch) ||
      incident.title.toLowerCase().includes(lowerSearch) ||
      (incident.categoryName?.toLowerCase().includes(lowerSearch) ?? false) ||
      (incident.categoryId?.toLowerCase().includes(lowerSearch) ?? false)
    );
  };

  const matchesDateRange = (
    incident: Incident,
    range: { start: Date | null; end: Date | null }
  ): boolean => {
    if (!range.start && !range.end) return true;
    const createdAt = new Date(incident.createdAt);
    if (range.start && createdAt < range.start) return false;
    if (range.end && createdAt > range.end) return false;
    return true;
  };

  // Pre-parse timestamps for efficient sorting (optimized)
  const timestampMap = useMemo(() => {
    const map = new Map<string, number>();
    incidents.forEach(i => {
      map.set(i.id, new Date(i.createdAt).getTime());
    });
    return map;
  }, [incidents]);

  const compareIncidents = useCallback((a: Incident, b: Incident, column: string): number => {
    switch (column) {
      case 'id':
        return a.id.localeCompare(b.id);
      case 'status':
        return a.status.localeCompare(b.status);
      case 'severity':
        return a.severityLevel - b.severityLevel;
      case 'title':
        return a.title.localeCompare(b.title);
      case 'createdBy': {
        const aName = a.createdByUser?.displayName || a.createdByUser?.name || a.createdBy;
        const bName = b.createdByUser?.displayName || b.createdByUser?.name || b.createdBy;
        return aName.localeCompare(bName);
      }
      case 'createdAt': {
        const aTime = timestampMap.get(a.id) || 0;
        const bTime = timestampMap.get(b.id) || 0;
        return aTime - bTime;
      }
      case 'category': {
        const aCategory = a.categoryName || a.categoryId || '';
        const bCategory = b.categoryName || b.categoryId || '';
        return aCategory.localeCompare(bCategory);
      }
      default:
        return 0;
    }
  }, [timestampMap]);

  // Data processing pipeline
  const filteredIncidents = useMemo(() => {
    return incidents.filter(incident => {
      if (searchText && !matchesSearchText(incident, searchText)) return false;
      if (statusFilter.size > 0 && !statusFilter.has(incident.status)) return false;
      if (severityFilter.size > 0 && !severityFilter.has(incident.severityLevel)) return false;
      if (!matchesDateRange(incident, dateRange)) return false;
      return true;
    });
  }, [incidents, searchText, statusFilter, severityFilter, dateRange]);

  const sortedIncidents = useMemo(() => {
    return [...filteredIncidents].sort((a, b) => {
      const compareValue = compareIncidents(a, b, sortColumn);
      return sortDirection === 'asc' ? compareValue : -compareValue;
    });
  }, [filteredIncidents, sortColumn, sortDirection, compareIncidents]);

  const paginatedIncidents = useMemo(() => {
    const startIndex = (currentPage - 1) * perPage;
    const endIndex = startIndex + perPage;
    return sortedIncidents.slice(startIndex, endIndex);
  }, [sortedIncidents, currentPage, perPage]);

  const totalPages = Math.ceil(sortedIncidents.length / perPage);
  const startIndex = (currentPage - 1) * perPage;
  const endIndex = Math.min(startIndex + perPage, sortedIncidents.length);

  // Event handlers
  const handleViewIncident = (id: string) => {
    navigate(`/incidents/${id}`);
  };

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection(prev => prev === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const handleStatusToggle = (status: IncidentStatus) => {
    const newFilter = new Set(statusFilter);
    if (newFilter.has(status)) {
      newFilter.delete(status);
    } else {
      newFilter.add(status);
    }
    setStatusFilter(newFilter);
    setCurrentPage(1);
  };

  const handleSeverityToggle = (level: number) => {
    const newFilter = new Set(severityFilter);
    if (newFilter.has(level)) {
      newFilter.delete(level);
    } else {
      newFilter.add(level);
    }
    setSeverityFilter(newFilter);
    setCurrentPage(1);
  };

  const handleDateRangePreset = (preset: 'today' | '7days' | '30days') => {
    const now = new Date();
    const start = new Date();

    switch (preset) {
      case 'today':
        start.setHours(0, 0, 0, 0);
        break;
      case '7days':
        start.setDate(now.getDate() - 7);
        break;
      case '30days':
        start.setDate(now.getDate() - 30);
        break;
    }

    setDateRange({ start, end: now });
    setCurrentPage(1);
  };

  const clearFilters = () => {
    setSearchInput('');
    setSearchText('');
    setStatusFilter(new Set());
    setSeverityFilter(new Set());
    setDateRange({ start: null, end: null });
    setCurrentPage(1);
  };

  const handlePerPageChange = (newPerPage: number) => {
    setPerPage(newPerPage);
    setCurrentPage(1);
  };

  const handleRowKeyDown = (e: React.KeyboardEvent, id: string) => {
    if (e.key === 'Enter') {
      handleViewIncident(id);
    }
  };

  // Get unique severity levels with names (optimized)
  const severityLevelMap = useMemo(() => {
    const map = new Map<number, string>();
    incidents.forEach(i => {
      if (!map.has(i.severityLevel)) {
        map.set(i.severityLevel, i.severityName);
      }
    });
    return map;
  }, [incidents]);

  const uniqueSeverityLevels = useMemo(() => {
    return Array.from(severityLevelMap.keys()).sort((a, b) => b - a);
  }, [severityLevelMap]);

  const hasActiveFilters = searchText || statusFilter.size > 0 || severityFilter.size > 0 || dateRange.start || dateRange.end;

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <RefreshCw className="h-8 w-8 animate-spin text-slate-400" />
          <p className="text-sm text-slate-500">Loading incidents...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-red-600 mt-0.5" />
          <div>
            <h3 className="font-medium text-red-900">Error loading incidents</h3>
            <p className="mt-1 text-sm text-red-700">{error.message}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Incidents</h1>
          <p className="mt-1 text-sm text-slate-500">
            Manage and track all incidents in your organization
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            className="gap-2"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" className="gap-2">
            <Plus className="h-4 w-4" />
            New Incident
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          icon={AlertCircle}
          iconColor="text-red-600"
          iconBgColor="bg-red-100"
          mainValue={stats.openCount}
          label="Open"
          subInfo={`Triage: ${stats.triageCount} / Handling: ${stats.handlingCount}`}
        />
        <StatCard
          icon={Clock}
          iconColor="text-orange-600"
          iconBgColor="bg-orange-100"
          mainValue={stats.longOpenCount}
          label="Long Open"
          subInfo={stats.maxDaysOpen > 0 ? `Max: ${stats.maxDaysOpen} days` : 'None'}
        />
        <StatCard
          icon={TrendingUp}
          iconColor="text-green-600"
          iconBgColor="bg-green-100"
          mainValue={`${stats.newThisWeek} / ${stats.resolvedThisWeek}`}
          label="This Week"
          subInfo={`Resolution Rate: ${stats.resolutionRate}%`}
        />
        <StatCard
          icon={Timer}
          iconColor="text-blue-600"
          iconBgColor="bg-blue-100"
          mainValue={`${stats.averageResponseHours}h`}
          label="Avg Response Time"
          subInfo={
            stats.responseTimeDiff !== 0 ? (
              <span>
                Last week: {stats.averageResponseHoursLastWeek}h{' '}
                <span className={stats.responseTimeDiff > 0 ? 'text-green-600' : 'text-red-600'}>
                  ({stats.responseTimeDiff > 0 ? '-' : '+'}{Math.abs(stats.responseTimeDiff)}h)
                </span>
              </span>
            ) : (
              `This week only`
            )
          }
        />
      </div>

      {/* Filters */}
      <div className="rounded-lg border border-slate-200 bg-white p-4">
        <div className="flex flex-col gap-3">
          {/* First Row: Search, Status, Severity */}
          <div className="flex flex-wrap items-center gap-3">
            {/* Search */}
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Search by title, ID, or category..."
                className="w-full rounded-md border border-slate-300 pl-10 pr-10 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
              />
              {searchInput && (
                <button
                  onClick={() => setSearchInput('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>

            {/* Status Dropdown */}
            <div className="relative">
              <button
                onClick={() => setStatusDropdownOpen(!statusDropdownOpen)}
                className={`h-10 rounded-md border px-3 py-2 text-sm min-w-[140px] flex items-center justify-between shadow-sm cursor-pointer transition-all ${
                  statusFilter.size > 0
                    ? 'border-blue-500 bg-blue-50 hover:bg-blue-100 hover:shadow-md'
                    : 'border-slate-300 bg-slate-50 hover:border-slate-400 hover:bg-slate-100 hover:shadow-md'
                }`}
              >
                <span className={statusFilter.size > 0 ? 'text-blue-700 font-medium' : 'text-slate-700 font-medium'}>
                  {statusFilter.size > 0 ? `Status (${statusFilter.size})` : 'Status'}
                </span>
                <ChevronDown className={`h-4 w-4 ml-2 ${statusFilter.size > 0 ? 'text-blue-600' : 'text-slate-500'}`} />
              </button>
              {statusDropdownOpen && (
                <>
                  <div
                    className="fixed inset-0 z-10"
                    onClick={() => setStatusDropdownOpen(false)}
                  />
                  <div className="absolute z-20 mt-1 w-48 rounded-md border border-slate-200 bg-white shadow-lg">
                    <div className="py-1">
                      {Object.values(IncidentStatus).map((status) => (
                        <label
                          key={status}
                          className="flex items-center gap-2 px-3 py-2 hover:bg-slate-50 cursor-pointer"
                        >
                          <input
                            type="checkbox"
                            checked={statusFilter.has(status)}
                            onChange={() => handleStatusToggle(status)}
                            className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                          />
                          <span className="text-sm text-slate-700 capitalize">
                            {status.toLowerCase()}
                          </span>
                          {statusFilter.has(status) && (
                            <Check className="ml-auto h-4 w-4 text-blue-600" />
                          )}
                        </label>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </div>

            {/* Severity Dropdown */}
            <div className="relative">
              <button
                onClick={() => setSeverityDropdownOpen(!severityDropdownOpen)}
                className={`h-10 rounded-md border px-3 py-2 text-sm min-w-[140px] flex items-center justify-between shadow-sm cursor-pointer transition-all ${
                  severityFilter.size > 0
                    ? 'border-blue-500 bg-blue-50 hover:bg-blue-100 hover:shadow-md'
                    : 'border-slate-300 bg-slate-50 hover:border-slate-400 hover:bg-slate-100 hover:shadow-md'
                }`}
              >
                <span className={severityFilter.size > 0 ? 'text-blue-700 font-medium' : 'text-slate-700 font-medium'}>
                  {severityFilter.size > 0 ? `Severity (${severityFilter.size})` : 'Severity'}
                </span>
                <ChevronDown className={`h-4 w-4 ml-2 ${severityFilter.size > 0 ? 'text-blue-600' : 'text-slate-500'}`} />
              </button>
              {severityDropdownOpen && (
                <>
                  <div
                    className="fixed inset-0 z-10"
                    onClick={() => setSeverityDropdownOpen(false)}
                  />
                  <div className="absolute z-20 mt-1 w-48 rounded-md border border-slate-200 bg-white shadow-lg">
                    <div className="py-1">
                      {uniqueSeverityLevels.map((level) => (
                        <label
                          key={level}
                          className="flex items-center gap-2 px-3 py-2 hover:bg-slate-50 cursor-pointer"
                        >
                          <input
                            type="checkbox"
                            checked={severityFilter.has(level)}
                            onChange={() => handleSeverityToggle(level)}
                            className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                          />
                          <span className="text-sm text-slate-700">
                            {severityLevelMap.get(level) || `Level ${level}`}
                          </span>
                          {severityFilter.has(level) && (
                            <Check className="ml-auto h-4 w-4 text-blue-600" />
                          )}
                        </label>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </div>

            {/* Date Range Buttons */}
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleDateRangePreset('today')}
                className={dateRange.start?.toDateString() === new Date().toDateString() ? 'bg-blue-50 border-blue-300' : ''}
              >
                Today
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleDateRangePreset('7days')}
              >
                7d
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleDateRangePreset('30days')}
              >
                30d
              </Button>
              {(dateRange.start || dateRange.end) && (
                <button
                  onClick={() => { setDateRange({ start: null, end: null }); setCurrentPage(1); }}
                  className="text-slate-400 hover:text-slate-600"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>

          {/* Second Row: Filter Summary */}
          <div className="flex items-center justify-between text-sm text-slate-600">
            <span>
              Showing {sortedIncidents.length} of {incidents.length} incidents
            </span>
            {hasActiveFilters && (
              <button
                onClick={clearFilters}
                className="text-blue-600 hover:text-blue-800 font-medium"
              >
                Clear all filters
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
        {paginatedIncidents.length === 0 ? (
          <div className="flex h-64 items-center justify-center">
            <div className="text-center">
              <AlertCircle className="mx-auto h-12 w-12 text-slate-300" />
              {incidents.length === 0 ? (
                <>
                  <p className="mt-3 text-sm text-slate-500">No incidents found</p>
                  <Button size="sm" className="mt-4" variant="outline">
                    Create your first incident
                  </Button>
                </>
              ) : (
                <>
                  <p className="mt-3 text-sm text-slate-500">No incidents match your filters</p>
                  <Button size="sm" className="mt-4" variant="outline" onClick={clearFilters}>
                    Clear Filters
                  </Button>
                </>
              )}
            </div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <th
                    scope="col"
                    className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider cursor-pointer hover:bg-slate-100"
                    onClick={() => handleSort('id')}
                  >
                    <div className="flex items-center gap-1">
                      ID
                      {sortColumn === 'id' && (
                        sortDirection === 'asc' ?
                          <ChevronUp className="h-3 w-3" /> :
                          <ChevronDown className="h-3 w-3" />
                      )}
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider cursor-pointer hover:bg-slate-100"
                    onClick={() => handleSort('status')}
                  >
                    <div className="flex items-center gap-1">
                      Status
                      {sortColumn === 'status' && (
                        sortDirection === 'asc' ?
                          <ChevronUp className="h-3 w-3" /> :
                          <ChevronDown className="h-3 w-3" />
                      )}
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider cursor-pointer hover:bg-slate-100"
                    onClick={() => handleSort('severity')}
                  >
                    <div className="flex items-center gap-1">
                      Severity
                      {sortColumn === 'severity' && (
                        sortDirection === 'asc' ?
                          <ChevronUp className="h-3 w-3" /> :
                          <ChevronDown className="h-3 w-3" />
                      )}
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider cursor-pointer hover:bg-slate-100"
                    onClick={() => handleSort('title')}
                  >
                    <div className="flex items-center gap-1">
                      Title
                      {sortColumn === 'title' && (
                        sortDirection === 'asc' ?
                          <ChevronUp className="h-3 w-3" /> :
                          <ChevronDown className="h-3 w-3" />
                      )}
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider"
                  >
                    Slack
                  </th>
                  <th
                    scope="col"
                    className="hidden md:table-cell px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider cursor-pointer hover:bg-slate-100"
                    onClick={() => handleSort('createdBy')}
                  >
                    <div className="flex items-center gap-1">
                      Created By
                      {sortColumn === 'createdBy' && (
                        sortDirection === 'asc' ?
                          <ChevronUp className="h-3 w-3" /> :
                          <ChevronDown className="h-3 w-3" />
                      )}
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider cursor-pointer hover:bg-slate-100"
                    onClick={() => handleSort('createdAt')}
                  >
                    <div className="flex items-center gap-1">
                      Created At
                      {sortColumn === 'createdAt' && (
                        sortDirection === 'asc' ?
                          <ChevronUp className="h-3 w-3" /> :
                          <ChevronDown className="h-3 w-3" />
                      )}
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="hidden lg:table-cell px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider cursor-pointer hover:bg-slate-100"
                    onClick={() => handleSort('category')}
                  >
                    <div className="flex items-center gap-1">
                      Category
                      {sortColumn === 'category' && (
                        sortDirection === 'asc' ?
                          <ChevronUp className="h-3 w-3" /> :
                          <ChevronDown className="h-3 w-3" />
                      )}
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="hidden xl:table-cell px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
                  >
                    Assets
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 bg-white">
                {paginatedIncidents.map((incident: Incident) => (
                  <tr
                    key={incident.id}
                    tabIndex={0}
                    onClick={() => handleViewIncident(incident.id)}
                    onKeyDown={(e) => handleRowKeyDown(e, incident.id)}
                    className="hover:bg-slate-50 cursor-pointer transition-colors"
                  >
                    <td className="px-4 py-3 text-xs font-medium text-slate-500 whitespace-nowrap">
                      #{incident.id}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <StatusBadge status={incident.status} size="sm" />
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <SeverityBadge
                        severityLevel={incident.severityLevel}
                        severityName={incident.severityName}
                        size="sm"
                      />
                    </td>
                    <td className="px-4 py-3 max-w-md">
                      <div className="flex items-center gap-2">
                        {incident.private && !incident.viewerCanAccess && (
                          <Lock className="h-4 w-4 text-slate-400 flex-shrink-0" />
                        )}
                        <div className="flex items-center gap-2">
                          <div className={`font-medium truncate ${
                            incident.private && !incident.viewerCanAccess
                              ? 'text-slate-400 italic'
                              : 'text-slate-900'
                          }`}>
                            {incident.title}
                          </div>
                          {incident.isTest && <TestBadge size="sm" />}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-center" onClick={(e) => e.stopPropagation()}>
                      <SlackChannelLink
                        channelId={incident.channelId}
                        channelName={incident.channelName}
                        teamId={incident.teamId}
                        iconOnly
                      />
                    </td>
                    <td className="hidden md:table-cell px-4 py-3">
                      <div className="flex items-center gap-2">
                        {incident.createdByUser?.avatarUrl ? (
                          <img
                            src={incident.createdByUser.avatarUrl}
                            alt={incident.createdByUser.displayName || incident.createdByUser.name}
                            className="h-6 w-6 rounded-full"
                          />
                        ) : (
                          <div className="flex h-6 w-6 items-center justify-center rounded-full bg-slate-100">
                            <User className="h-4 w-4 text-slate-400" />
                          </div>
                        )}
                        <span className="text-sm text-slate-600 truncate max-w-[120px]">
                          {incident.createdByUser?.displayName ||
                           incident.createdByUser?.realName ||
                           incident.createdByUser?.name ||
                           incident.createdBy}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-slate-600 whitespace-nowrap">
                      {format(new Date(incident.createdAt), 'MMM d, yyyy HH:mm')}
                    </td>
                    <td className="hidden lg:table-cell px-4 py-3 text-sm text-slate-600">
                      <span className="truncate max-w-[100px] block">
                        {incident.categoryName || incident.categoryId || '-'}
                      </span>
                    </td>
                    <td className="hidden xl:table-cell px-4 py-3">
                      {incident.assetNames && incident.assetNames.length > 0 ? (
                        <div className="flex flex-wrap gap-1 max-w-[200px]">
                          {incident.assetNames.slice(0, 3).map((assetName: string, index: number) => (
                            <span
                              key={incident.assetIds[index]}
                              className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800"
                            >
                              {assetName}
                            </span>
                          ))}
                          {incident.assetNames.length > 3 && (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600">
                              +{incident.assetNames.length - 3}
                            </span>
                          )}
                        </div>
                      ) : (
                        <span className="text-sm text-slate-400">-</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Pagination */}
      {sortedIncidents.length > 0 && (
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-slate-600">Show</span>
            <select
              value={perPage}
              onChange={(e) => handlePerPageChange(Number(e.target.value))}
              className="rounded-md border border-slate-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value={10}>10</option>
              <option value={20}>20</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
            </select>
            <span className="text-sm text-slate-600">per page</span>
          </div>

          <div className="text-sm text-slate-600">
            Showing {startIndex + 1}-{endIndex} of {sortedIncidents.length} incidents
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={currentPage === 1}
              onClick={() => setCurrentPage(p => p - 1)}
            >
              Previous
            </Button>
            <span className="text-sm text-slate-600">
              Page {currentPage} of {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={currentPage === totalPages}
              onClick={() => setCurrentPage(p => p + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

export default IncidentList;
