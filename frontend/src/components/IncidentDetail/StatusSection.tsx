import React, { useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { IncidentStatus, StatusHistory, getStatusConfig } from '../../types/incident';
import StatusBadge from '../IncidentList/StatusBadge';
import StatusIcon from '../common/StatusIcon';
import StatusChangeModal from './StatusChangeModal';
import { Button } from '../ui/Button';

interface StatusSectionProps {
  incidentId: string;
  currentStatus: IncidentStatus;
  statusHistories: StatusHistory[];
  className?: string;
}

export const StatusSection: React.FC<StatusSectionProps> = ({
  incidentId,
  currentStatus,
  statusHistories,
  className = ''
}) => {
  const [showChangeModal, setShowChangeModal] = useState(false);
  const currentConfig = getStatusConfig(currentStatus);
  
  // Sort histories by timestamp (newest first for display)
  const sortedHistories = [...statusHistories].sort(
    (a, b) => new Date(b.changedAt).getTime() - new Date(a.changedAt).getTime()
  );

  const handleStatusChange = () => {
    // Status change will be handled by the modal
    setShowChangeModal(false);
  };

  return (
    <div className={`bg-white border border-gray-200 rounded-lg p-4 ${className}`}>
      {/* Header with status title and update button */}
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold">Status</h3>
        <Button
          onClick={() => setShowChangeModal(true)}
          variant="primary"
          size="sm"
          className="bg-blue-600 hover:bg-blue-700 text-white shadow-sm transition-all duration-200 hover:shadow-md"
        >
          Update Status
        </Button>
      </div>
      {sortedHistories.length === 0 ? (
        <p className="text-gray-500 text-center py-4">No status history available</p>
      ) : (
        <div className="space-y-3">
          {sortedHistories.map((history, index) => {
            const isLast = index === sortedHistories.length - 1;
            const config = getStatusConfig(history.status);

            return (
              <div key={history.id} className="relative">
                {/* Timeline connector */}
                {!isLast && (
                  <div className="absolute left-3 top-8 bottom-0 w-0.5 bg-gray-200" />
                )}

                <div className="flex items-start gap-3">
                  {/* Status icon */}
                  <div className="relative flex-shrink-0">
                    <div
                      className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                      style={{
                        backgroundColor: config.color,
                        color: 'white'
                      }}
                    >
                      {config.icon}
                    </div>
                  </div>

                  {/* Status information */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <StatusBadge status={history.status} size="sm" />
                      <span className="text-sm text-gray-500">
                        {formatDistanceToNow(new Date(history.changedAt), { addSuffix: true })}
                      </span>
                    </div>

                    {/* User info */}
                    <div className="text-sm text-gray-600 mb-1">
                      Changed by{' '}
                      <span className="font-medium">
                        @{history.changedBy.displayName || history.changedBy.name}
                      </span>
                    </div>

                    {/* Note */}
                    {history.note && (
                      <div className="text-sm text-gray-500 italic bg-gray-50 rounded p-2 mt-1">
                        "{history.note}"
                      </div>
                    )}

                    {/* Exact timestamp */}
                    <div className="text-xs text-gray-400 mt-1">
                      {new Date(history.changedAt).toLocaleString()}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Status Change Modal */}
      {showChangeModal && (
        <StatusChangeModal
          incidentId={incidentId}
          currentStatus={currentStatus}
          onClose={() => setShowChangeModal(false)}
          onStatusChange={handleStatusChange}
        />
      )}
    </div>
  );
};

export default StatusSection;