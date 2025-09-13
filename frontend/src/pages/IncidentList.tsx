import React from 'react';
import { useQuery } from '@apollo/client/react';
import { useNavigate } from 'react-router-dom';
import { format } from 'date-fns';
import { GET_INCIDENTS } from '../graphql/queries';
import { Button } from '../components/ui/Button';
import { IncidentStatus } from '../types/incident';
import StatusBadge from '../components/IncidentList/StatusBadge';
import {
  AlertCircle,
  RefreshCw,
  ChevronRight,
  Clock,
  Hash,
  User,
  MessageSquare,
  Plus,
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
  categoryName?: string;
  status: IncidentStatus;
  createdBy: string;
  createdByUser?: User;
  createdAt: string;
  updatedAt: string;
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
  const rowsPerPage = 20;

  const { loading, error, data, refetch } = useQuery<IncidentsData>(
    GET_INCIDENTS,
    {
      variables: {
        first: rowsPerPage,
        after: null,
      },
      fetchPolicy: 'cache-and-network',
      notifyOnNetworkStatusChange: true,
    }
  );


  const handleViewIncident = (id: string) => {
    navigate(`/incidents/${id}`);
  };


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

  const incidents = data?.incidents?.edges?.map((edge: IncidentEdge) => edge.node) || [];
  const totalCount = data?.incidents?.totalCount || 0;

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
        <div key="stat-total" className="rounded-lg border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-blue-100 p-2">
              <AlertCircle className="h-5 w-5 text-blue-600" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-slate-900">{totalCount}</p>
              <p className="text-sm text-slate-500">Total Incidents</p>
            </div>
          </div>
        </div>
        <div key="stat-open" className="rounded-lg border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-red-100 p-2">
              <AlertCircle className="h-5 w-5 text-red-600" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-slate-900">
                {incidents.filter(i => i.status === IncidentStatus.HANDLING || i.status === IncidentStatus.TRIAGE).length}
              </p>
              <p className="text-sm text-slate-500">Open</p>
            </div>
          </div>
        </div>
        <div key="stat-closed" className="rounded-lg border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-green-100 p-2">
              <AlertCircle className="h-5 w-5 text-green-600" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-slate-900">
                {incidents.filter(i => i.status === IncidentStatus.CLOSED).length}
              </p>
              <p className="text-sm text-slate-500">Closed</p>
            </div>
          </div>
        </div>
        <div key="stat-today" className="rounded-lg border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-purple-100 p-2">
              <Clock className="h-5 w-5 text-purple-600" />
            </div>
            <div>
              <p className="text-2xl font-semibold text-slate-900">
                {incidents.filter(i => {
                  const created = new Date(i.createdAt);
                  const now = new Date();
                  return (now.getTime() - created.getTime()) < 86400000;
                }).length}
              </p>
              <p className="text-sm text-slate-500">Today</p>
            </div>
          </div>
        </div>
      </div>

      {/* Incidents List */}
      <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
        {incidents.length === 0 ? (
          <div className="flex h-64 items-center justify-center">
            <div className="text-center">
              <AlertCircle className="mx-auto h-12 w-12 text-slate-300" />
              <p className="mt-3 text-sm text-slate-500">No incidents found</p>
              <Button size="sm" className="mt-4" variant="outline">
                Create your first incident
              </Button>
            </div>
          </div>
        ) : (
          <div className="divide-y divide-slate-200">
            {incidents.map((incident: Incident) => (
              <div
                key={incident.id}
                className="group flex items-center justify-between p-4 hover:bg-slate-50 cursor-pointer transition-colors"
                onClick={() => handleViewIncident(incident.id)}
              >
                <div className="flex-1 space-y-1">
                  <div className="flex items-start gap-3">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium text-slate-500">
                        #{incident.id}
                      </span>
                      <StatusBadge status={incident.status} size="sm" />
                    </div>
                  </div>
                  <h3 className="font-medium text-slate-900 group-hover:text-blue-600 transition-colors">
                    {incident.title}
                  </h3>
                  {incident.description && (
                    <p className="text-sm text-slate-500 line-clamp-2">
                      {incident.description}
                    </p>
                  )}
                  <div className="flex items-center gap-4 text-xs text-slate-500">
                    <span className="flex items-center gap-1">
                      <MessageSquare className="h-3 w-3" />
                      {incident.channelName}
                    </span>
                    <span className="flex items-center gap-1">
                      {incident.createdByUser?.avatarUrl ? (
                        <img 
                          src={incident.createdByUser.avatarUrl} 
                          alt={incident.createdByUser.displayName || incident.createdByUser.name}
                          className="h-4 w-4 rounded-full"
                        />
                      ) : (
                        <User className="h-3 w-3" />
                      )}
                      {incident.createdByUser?.displayName || 
                       incident.createdByUser?.realName || 
                       incident.createdByUser?.name || 
                       incident.createdBy}
                    </span>
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {format(new Date(incident.createdAt), 'MMM d, yyyy HH:mm')}
                    </span>
                    {(incident.categoryName || incident.categoryId) && (
                      <span className="flex items-center gap-1">
                        <Hash className="h-3 w-3" />
                        {incident.categoryName || incident.categoryId}
                      </span>
                    )}
                  </div>
                </div>
                <ChevronRight className="h-5 w-5 text-slate-400 group-hover:text-slate-600 transition-colors" />
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalCount > rowsPerPage && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-slate-500">
            Showing {incidents.length} of {totalCount} incidents
          </p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled>
              Previous
            </Button>
            <Button variant="outline" size="sm">
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

export default IncidentList;