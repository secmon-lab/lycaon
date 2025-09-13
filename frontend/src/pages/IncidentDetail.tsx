import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@apollo/client/react';
import { format } from 'date-fns';
import { GET_INCIDENT } from '../graphql/queries';
import { IncidentStatus } from '../types/incident';
import StatusSection from '../components/IncidentDetail/StatusSection';
import TaskList from '../components/IncidentDetail/TaskList';
import {
  MessageSquare,
  User,
  Calendar,
} from 'lucide-react';

const IncidentDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { loading, error, data } = useQuery<{ incident: any }>(GET_INCIDENT, {
    variables: { id },
    skip: !id,
  });

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-slate-200 border-t-blue-600 mx-auto" />
          <p className="mt-2 text-sm text-slate-500">Loading incident...</p>
        </div>
      </div>
    );
  }

  if (error || !data?.incident) {
    return (
      <div className="max-w-2xl mx-auto p-4">
        <div className="rounded-lg border border-red-200 bg-red-50 p-4">
          <p className="text-red-700">Error: {error?.message || 'Incident not found'}</p>
        </div>
      </div>
    );
  }

  const incident = data.incident;

  return (
    <div className="max-w-7xl mx-auto p-4">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/incidents')}
            className="flex items-center justify-center h-10 w-10 rounded-lg border-2 border-slate-500 bg-slate-100 hover:bg-slate-200 hover:border-slate-600 transition-colors text-slate-700 font-bold text-xl"
          >
            ←
          </button>
          <div>
            <h1 className="text-2xl font-bold">Incident #{incident.id}</h1>
            <p className="text-sm text-slate-500">
              Created {format(new Date(incident.createdAt), 'MMM d, yyyy HH:mm')}
            </p>
          </div>
        </div>
      </div>

      {/* Two Column Layout */}
      <div className="flex flex-col md:flex-row gap-6">
        {/* Left Column - Main Content */}
        <div className="flex-1">
          <div className="bg-white rounded-lg border p-6">
            <h2 className="text-lg font-semibold mb-3">{incident.title}</h2>
            <p className="text-slate-600">
              {incident.description || 'No description provided.'}
            </p>
          </div>

          {/* Tasks */}
          <div className="mt-6 bg-white rounded-lg border p-6">
            <TaskList 
              incidentId={incident.id} 
              tasks={incident.tasks || []} 
            />
          </div>
        </div>

        {/* Right Column - Sidebar */}
        <div className="w-full md:w-80">
          {/* Details Box */}
          <div className="bg-white border border-slate-200 rounded-lg p-4 mb-4">
            <h3 className="font-semibold mb-3">Details</h3>
            
            <div className="space-y-3">
              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <MessageSquare className="h-3 w-3" />
                  Channel
                </div>
                <p className="text-sm font-medium">#{incident.channelName}</p>
              </div>

              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <User className="h-3 w-3" />
                  Lead
                </div>
                <p className="text-sm font-medium">
                  {incident.leadUser?.name || incident.lead || 'Not assigned'}
                </p>
              </div>

              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <User className="h-3 w-3" />
                  Created By
                </div>
                <p className="text-sm font-medium">
                  {incident.createdByUser?.name || incident.createdBy}
                </p>
              </div>

              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <Calendar className="h-3 w-3" />
                  Created
                </div>
                <p className="text-sm">
                  {format(new Date(incident.createdAt), 'MMM d, yyyy')}
                </p>
              </div>
            </div>
          </div>

          {/* Status Management Section */}
          <StatusSection
            incidentId={incident.id}
            currentStatus={incident.status as IncidentStatus}
            statusHistories={incident.statusHistories || []}
            className="mb-4"
          />
        </div>
      </div>
    </div>
  );
};

export default IncidentDetail;