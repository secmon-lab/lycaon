import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@apollo/client/react';
import { format } from 'date-fns';
import { GET_INCIDENT, GET_SEVERITIES } from '../graphql/queries';
import { IncidentStatus, toIncidentStatus } from '../types/incident';
import StatusSection from '../components/IncidentDetail/StatusSection';
import TaskList from '../components/IncidentDetail/TaskList';
import { EditIncidentModal } from '../components/IncidentDetail/EditIncidentModal';
import { Button } from '../components/ui/Button';
import SlackChannelLink from '../components/common/SlackChannelLink';
import SeverityBadge from '../components/common/SeverityBadge';
import {
  User,
  Calendar,
  Edit,
  Tag,
  AlertTriangle,
} from 'lucide-react';

const IncidentDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [showEditModal, setShowEditModal] = useState(false);

  const { loading, error, data } = useQuery<{ incident: any }>(GET_INCIDENT, {
    variables: { id },
    skip: !id,
  });

  const { data: severitiesData } = useQuery<{ severities: Array<{ id: string; name: string; level: number }> }>(GET_SEVERITIES);

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

  // Validate and convert status safely
  const validStatus = toIncidentStatus(incident.status) || IncidentStatus.TRIAGE;

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
            <div className="flex items-start justify-between mb-3">
              <h2 className="text-lg font-semibold">{incident.title}</h2>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowEditModal(true)}
                className="flex items-center gap-1"
              >
                <Edit className="h-4 w-4" />
                Edit
              </Button>
            </div>
            <p className="text-slate-600">
              {incident.description || 'No description provided.'}
            </p>
          </div>

          {/* Tasks */}
          <div className="mt-6 bg-white rounded-lg border p-6">
            <TaskList
              incidentId={incident.id}
              incident={incident}
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
                <div className="text-xs text-slate-500 mb-1">
                  Channel
                </div>
                <SlackChannelLink
                  channelId={incident.channelId}
                  channelName={incident.channelName}
                  teamId={incident.teamId}
                  className="text-sm font-medium"
                />
              </div>

              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <Tag className="h-3 w-3" />
                  Category
                </div>
                <p className="text-sm font-medium">
                  {incident.categoryName || 'No category assigned'}
                </p>
              </div>

              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <AlertTriangle className="h-3 w-3" />
                  Severity
                </div>
                <SeverityBadge
                  severityLevel={incident.severityLevel}
                  severityName={incident.severityName}
                />
              </div>

              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <User className="h-3 w-3" />
                  Lead
                </div>
                <div className="flex items-center gap-2">
                  {incident.leadUser?.avatarUrl ? (
                    <img
                      src={incident.leadUser.avatarUrl}
                      alt={incident.leadUser.name || 'User'}
                      className="h-5 w-5 rounded-full"
                    />
                  ) : incident.leadUser ? (
                    <div className="h-5 w-5 rounded-full bg-slate-300 flex items-center justify-center text-xs font-medium text-slate-600">
                      {(incident.leadUser.name || incident.leadUser.displayName || '?').charAt(0).toUpperCase()}
                    </div>
                  ) : null}
                  <p className="text-sm font-medium">
                    {incident.leadUser?.name || incident.lead || 'Not assigned'}
                  </p>
                </div>
              </div>

              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <User className="h-3 w-3" />
                  Created By
                </div>
                <div className="flex items-center gap-2">
                  {incident.createdByUser?.avatarUrl ? (
                    <img
                      src={incident.createdByUser.avatarUrl}
                      alt={incident.createdByUser.name || 'User'}
                      className="h-5 w-5 rounded-full"
                    />
                  ) : incident.createdByUser ? (
                    <div className="h-5 w-5 rounded-full bg-slate-300 flex items-center justify-center text-xs font-medium text-slate-600">
                      {(incident.createdByUser.name || incident.createdByUser.displayName || '?').charAt(0).toUpperCase()}
                    </div>
                  ) : null}
                  <p className="text-sm font-medium">
                    {incident.createdByUser?.name || incident.createdBy}
                  </p>
                </div>
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
            currentStatus={validStatus}
            statusHistories={incident.statusHistories || []}
            className="mb-4"
          />
        </div>
      </div>

      {/* Edit Incident Modal */}
      {showEditModal && severitiesData?.severities && (
        <EditIncidentModal
          incidentId={incident.id}
          currentTitle={incident.title}
          currentDescription={incident.description || ''}
          currentLead={incident.lead}
          currentSeverityId={incident.severityId}
          severities={severitiesData.severities}
          onClose={() => setShowEditModal(false)}
          onUpdate={() => {
            // Optionally, you can add a success toast here
          }}
        />
      )}
    </div>
  );
};

export default IncidentDetail;