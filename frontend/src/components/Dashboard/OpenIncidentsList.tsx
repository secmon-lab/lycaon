import React from 'react';
import { format } from 'date-fns';
import { IncidentCard } from './IncidentCard';
import { GroupedIncidents } from '../../types/dashboard';

interface OpenIncidentsListProps {
  incidents: GroupedIncidents[];
  loading?: boolean;
  error?: Error;
}

export const OpenIncidentsList: React.FC<OpenIncidentsListProps> = ({
  incidents,
  loading,
  error,
}) => {
  if (loading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          Recent Open Incidents (Last 14 Days)
        </h2>
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-white rounded-lg border border-red-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          Recent Open Incidents (Last 14 Days)
        </h2>
        <div className="text-red-600">
          <p className="font-medium">Error loading incidents</p>
          <p className="text-sm mt-1">{error.message}</p>
        </div>
      </div>
    );
  }

  if (!incidents || incidents.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          Recent Open Incidents (Last 14 Days)
        </h2>
        <p className="text-gray-500 text-center py-8">
          No open incidents in the last 14 days
        </p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <h2 className="text-lg font-semibold text-gray-900 mb-4">
        Recent Open Incidents (Last 14 Days)
      </h2>
      <div className="space-y-6">
        {incidents.map((group) => (
          <div key={group.date}>
            <h3 className="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
              <span className="text-lg">📅</span>
              {format(new Date(group.date), 'MMMM d, yyyy')}
            </h3>
            <div className="space-y-2">
              {group.incidents.map((incident) => (
                <IncidentCard key={incident.id} incident={incident} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
