import React from 'react';
import { Link } from 'react-router-dom';
import { format } from 'date-fns';
import * as Avatar from '@radix-ui/react-avatar';
import SeverityBadge from '../common/SeverityBadge';
import StatusBadge from '../IncidentList/StatusBadge';

interface Incident {
  id: string;
  title: string;
  description: string;
  severityId: string;
  severityName: string;
  severityLevel: number;
  status: string;
  lead: string;
  leadUser?: {
    id: string;
    name: string;
    displayName: string;
    avatarUrl: string;
  };
  createdAt: string;
}

interface IncidentCardProps {
  incident: Incident;
}

export const IncidentCard: React.FC<IncidentCardProps> = ({ incident }) => {
  const truncateText = (text: string, maxLength: number) => {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
  };

  const leadName = incident.leadUser?.displayName || incident.leadUser?.name || incident.lead;
  const leadInitials = leadName
    ? leadName
        .split(' ')
        .map((n) => n[0])
        .join('')
        .toUpperCase()
        .substring(0, 2)
    : '';

  return (
    <Link
      to={`/incidents/${incident.id}`}
      className="block p-4 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-sm font-medium text-gray-500">#{incident.id}</span>
            <h3 className="text-base font-medium text-gray-900">{incident.title}</h3>
          </div>
          {incident.description && (
            <p className="text-sm text-gray-600 mb-2">
              {truncateText(incident.description, 128)}
            </p>
          )}
          <div className="flex items-center gap-3 text-sm">
            <SeverityBadge
              severityName={incident.severityName}
              severityLevel={incident.severityLevel}
              size="sm"
            />
            <StatusBadge status={incident.status} size="sm" />
            <div className="flex items-center gap-1.5 text-gray-600">
              <Avatar.Root className="inline-flex h-5 w-5 select-none items-center justify-center overflow-hidden rounded-full bg-gray-200">
                {leadName ? (
                  <>
                    <Avatar.Image
                      className="h-full w-full object-cover"
                      src={incident.leadUser?.avatarUrl}
                      alt={leadName}
                    />
                    <Avatar.Fallback className="flex h-full w-full items-center justify-center bg-gray-200 text-xs font-medium text-gray-600">
                      {leadInitials}
                    </Avatar.Fallback>
                  </>
                ) : (
                  <Avatar.Fallback className="flex h-full w-full items-center justify-center bg-gray-200 text-xs font-medium text-gray-600">
                    -
                  </Avatar.Fallback>
                )}
              </Avatar.Root>
              <span className={leadName ? '' : 'text-gray-400'}>
                {leadName || 'No Assign'}
              </span>
            </div>
            <span className="text-gray-500">
              {format(new Date(incident.createdAt), 'HH:mm')}
            </span>
          </div>
        </div>
      </div>
    </Link>
  );
};
