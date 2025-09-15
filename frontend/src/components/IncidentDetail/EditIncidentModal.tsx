import React, { useState } from 'react';
import { useMutation } from '@apollo/client/react';
import { UPDATE_INCIDENT } from '../../graphql/mutations';
import { GET_INCIDENT } from '../../graphql/queries';
import { Button } from '../ui/Button';
import { X } from 'lucide-react';

interface EditIncidentModalProps {
  incidentId: string;
  currentTitle: string;
  currentDescription: string;
  currentLead: string | null;
  onClose: () => void;
  onUpdate: () => void;
}

export const EditIncidentModal: React.FC<EditIncidentModalProps> = ({
  incidentId,
  currentTitle,
  currentDescription,
  currentLead,
  onClose,
  onUpdate
}) => {
  const [title, setTitle] = useState(currentTitle);
  const [description, setDescription] = useState(currentDescription);
  const [lead, setLead] = useState(currentLead || '');

  const [updateIncident, { loading }] = useMutation(UPDATE_INCIDENT, {
    refetchQueries: [
      { query: GET_INCIDENT, variables: { id: incidentId } }
    ],
    onCompleted: () => {
      onUpdate();
      onClose();
    },
    onError: (error) => {
      console.error('Failed to update incident:', error);
      // You might want to show a toast notification here
    }
  });

  const handleSave = async () => {
    // Only include fields that have changed
    const input: { title?: string; description?: string; lead?: string | null } = {};
    if (title !== currentTitle) {
      input.title = title;
    }
    if (description !== currentDescription) {
      input.description = description;
    }
    if (lead !== currentLead) {
      input.lead = lead || null;
    }

    // Only update if there are changes
    if (Object.keys(input).length === 0) {
      onClose();
      return;
    }

    await updateIncident({
      variables: {
        id: incidentId,
        input
      }
    });
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl mx-4">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Edit Incident Details</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          {/* Title field */}
          <div>
            <label htmlFor="title" className="block text-sm font-medium text-gray-700 mb-1">
              Title
            </label>
            <input
              id="title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Enter incident title"
            />
          </div>

          {/* Description field */}
          <div>
            <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
              placeholder="Enter incident description"
            />
          </div>

          {/* Lead field */}
          <div>
            <label htmlFor="lead" className="block text-sm font-medium text-gray-700 mb-1">
              Lead (Slack User ID)
            </label>
            <input
              id="lead"
              type="text"
              value={lead}
              onChange={(e) => setLead(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Enter Slack user ID (e.g., U12345678)"
            />
            <p className="mt-1 text-sm text-gray-500">
              Enter the Slack user ID of the incident lead
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t bg-gray-50">
          <Button
            variant="ghost"
            onClick={onClose}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={handleSave}
            disabled={loading || !title.trim()}
          >
            {loading ? 'Saving...' : 'Save'}
          </Button>
        </div>
      </div>
    </div>
  );
};