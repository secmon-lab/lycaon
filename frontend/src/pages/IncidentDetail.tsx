import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@apollo/client/react';
import { format } from 'date-fns';
import { GET_INCIDENT, GET_SEVERITIES, GET_ASSETS } from '../graphql/queries';
import { IncidentStatus, toIncidentStatus, Asset } from '../types/incident';
import StatusSection from '../components/IncidentDetail/StatusSection';
import TaskList from '../components/IncidentDetail/TaskList';
import { EditIncidentModal } from '../components/IncidentDetail/EditIncidentModal';
import { Button } from '../components/ui/Button';
import SlackChannelLink from '../components/common/SlackChannelLink';
import SeverityBadge from '../components/common/SeverityBadge';
import TestBadge from '../components/common/TestBadge';
import {
  User,
  Calendar,
  Edit,
  Tag,
  AlertTriangle,
  Server,
  Lock,
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
  const { data: assetsData } = useQuery<{ assets: Asset[] }>(GET_ASSETS);

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

  // Check if this is a private incident with restricted access
  const isPrivateRestricted = incident.private && !incident.viewerCanAccess;

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
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold">Incident #{incident.id}</h1>
              {isPrivateRestricted && (
                <Lock className="h-5 w-5 text-slate-400" />
              )}
              {incident.isTest && <TestBadge size="md" />}
            </div>
            <p className="text-sm text-slate-500">
              Created {format(new Date(incident.createdAt), 'MMM d, yyyy HH:mm')}
            </p>
          </div>
        </div>
      </div>

      {/* Private Incident Restricted View */}
      {isPrivateRestricted ? (
        <div className="bg-white rounded-lg border p-8">
          <div className="flex flex-col items-center justify-center text-center max-w-md mx-auto">
            <div className="rounded-full bg-slate-100 p-4 mb-4">
              <Lock className="h-12 w-12 text-slate-400" />
            </div>
            <h2 className="text-xl font-semibold text-slate-900 mb-2">
              Private Incident
            </h2>
            <p className="text-slate-600 mb-6">
              This is a private incident. You don't have access to view the full details.
              Only members who have joined the incident Slack channel can see the complete information.
            </p>
            <div className="w-full bg-slate-50 rounded-lg p-4 text-left space-y-2">
              <p className="text-sm text-slate-500">
                You can view limited information:
              </p>
              <ul className="text-sm text-slate-600 space-y-1 ml-4">
                <li>• Status: {validStatus}</li>
                <li>• Category: {incident.categoryName || incident.categoryId}</li>
                <li>• Severity: {incident.severityName}</li>
                <li>• Created: {format(new Date(incident.createdAt), 'MMM d, yyyy HH:mm')}</li>
              </ul>
            </div>
          </div>
        </div>
      ) : (
        <>
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

              {/* Visibility */}
              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <Lock className="h-3 w-3" />
                  Visibility
                </div>
                {incident.private ? (
                  <div className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-slate-100 text-slate-700">
                    <Lock className="h-3 w-3" />
                    <span className="text-xs font-medium">Private</span>
                  </div>
                ) : (
                  <span className="text-sm text-slate-600">Public</span>
                )}
              </div>

              {/* Assets */}
              <div>
                <div className="flex items-center gap-1 text-xs text-slate-500 mb-1">
                  <Server className="h-3 w-3" />
                  Assets
                </div>
                {incident.assetNames && incident.assetNames.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {incident.assetNames.map((assetName: string, index: number) => (
                      <span
                        key={incident.assetIds[index]}
                        className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800"
                      >
                        {assetName}
                      </span>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm font-medium text-slate-400">No assets assigned</p>
                )}
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
      {!isPrivateRestricted && showEditModal && severitiesData?.severities && (
        <EditIncidentModal
          incidentId={incident.id}
          currentTitle={incident.title}
          currentDescription={incident.description || ''}
          currentLead={incident.lead}
          currentSeverityId={incident.severityId}
          currentAssetIds={incident.assetIds || []}
          severities={severitiesData.severities}
          assets={assetsData?.assets || []}
          onClose={() => setShowEditModal(false)}
          onUpdate={() => {
            // Optionally, you can add a success toast here
          }}
        />
      )}
        </>
      )}
    </div>
  );
};

export default IncidentDetail;