import React, { useState } from 'react';
import { useMutation } from '@apollo/client/react';
import { IncidentStatus, getStatusConfig } from '../../types/incident';
import { UPDATE_INCIDENT_STATUS } from '../../graphql/mutations';
import { GET_INCIDENT } from '../../graphql/queries';
import StatusBadge from '../IncidentList/StatusBadge';
import { Button } from '../ui/Button';
import { X } from 'lucide-react';

interface StatusChangeModalProps {
  incidentId: string;
  currentStatus: IncidentStatus | string | null | undefined;
  onClose: () => void;
  onStatusChange: (status: IncidentStatus) => void;
}

export const StatusChangeModal: React.FC<StatusChangeModalProps> = ({
  incidentId,
  currentStatus,
  onClose,
  onStatusChange
}) => {
  const [selectedStatus, setSelectedStatus] = useState<IncidentStatus | null>(null);
  const [note, setNote] = useState('');

  const [updateIncidentStatus, { loading }] = useMutation(UPDATE_INCIDENT_STATUS, {
    refetchQueries: [
      { query: GET_INCIDENT, variables: { id: incidentId } }
    ],
    onCompleted: () => {
      if (selectedStatus) {
        onStatusChange(selectedStatus);
      }
      onClose();
    },
    onError: (error) => {
      console.error('Failed to update status:', error);
      // You might want to show a toast notification here
    }
  });

  const statusOptions = [
    IncidentStatus.TRIAGE,
    IncidentStatus.HANDLING,
    IncidentStatus.MONITORING,
    IncidentStatus.CLOSED
  ];

  const handleConfirmChange = async () => {
    if (!selectedStatus) return;

    try {
      await updateIncidentStatus({
        variables: {
          incidentId,
          status: selectedStatus,
          note: note.trim() || undefined
        }
      });
    } catch (error) {
      // Error is handled in the onError callback
    }
  };


  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Change Status</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="p-4">
          {/* Status Options */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Status
            </label>
            <div className="grid grid-cols-1 gap-2">
              {statusOptions.map((status) => {
                const config = getStatusConfig(status);
                const isCurrent = status === currentStatus && currentStatus != null;
                const isSelected = status === selectedStatus;
                
                return (
                  <button
                    key={status}
                    onClick={() => setSelectedStatus(status)}
                    disabled={isCurrent}
                    className={`
                      w-full p-3 text-left rounded-lg border-2 transition-all
                      ${isCurrent 
                        ? 'bg-gray-100 border-gray-200 cursor-not-allowed opacity-60' 
                        : isSelected
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                      }
                    `}
                  >
                    <div className="flex items-center gap-3">
                      <StatusBadge status={status} size="sm" />
                      <div className="flex-1">
                        <div className="text-sm text-gray-600">
                          {config.description}
                        </div>
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Note */}
          <div className="mb-4">
            <label htmlFor="status-note" className="block text-sm font-medium text-gray-700 mb-2">
              Note (optional)
            </label>
            <textarea
              id="status-note"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Add a note about this status change..."
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 resize-none"
              rows={3}
            />
          </div>

        </div>

        {/* Footer */}
        <div className="flex gap-2 p-4 border-t bg-gray-50 rounded-b-lg">
          <Button
            onClick={onClose}
            variant="secondary"
            disabled={loading}
            className="flex-1"
          >
            Cancel
          </Button>
          <Button
            onClick={handleConfirmChange}
            disabled={loading || !selectedStatus || selectedStatus === currentStatus}
            className="flex-1"
          >
            {loading ? 'Updating...' : 'Confirm Change'}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default StatusChangeModal;