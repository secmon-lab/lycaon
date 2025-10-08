import React, { useState, useRef, useEffect } from 'react';
import { useMutation } from '@apollo/client/react';
import { UPDATE_INCIDENT } from '../../graphql/mutations';
import { GET_INCIDENT } from '../../graphql/queries';
import { Button } from '../ui/Button';
import { getSeverityStyle, Asset } from '../../types/incident';
import { X, ChevronDown } from 'lucide-react';

interface Severity {
  id: string;
  name: string;
  level: number;
}

interface EditIncidentModalProps {
  incidentId: string;
  currentTitle: string;
  currentDescription: string;
  currentLead: string | null;
  currentSeverityId: string;
  currentAssetIds: string[];
  severities: Severity[];
  assets: Asset[];
  onClose: () => void;
  onUpdate: () => void;
}

export const EditIncidentModal: React.FC<EditIncidentModalProps> = ({
  incidentId,
  currentTitle,
  currentDescription,
  currentLead,
  currentSeverityId,
  currentAssetIds,
  severities,
  assets,
  onClose,
  onUpdate
}) => {
  const [title, setTitle] = useState(currentTitle);
  const [description, setDescription] = useState(currentDescription);
  const [lead, setLead] = useState(currentLead || '');
  const [severityId, setSeverityId] = useState(currentSeverityId);
  const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>(currentAssetIds);
  const [isAssetDropdownOpen, setIsAssetDropdownOpen] = useState(false);
  const assetDropdownRef = useRef<HTMLDivElement>(null);

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
    const input: { title?: string; description?: string; lead?: string | null; severityId?: string; assetIds?: string[] } = {};
    if (title !== currentTitle) {
      input.title = title;
    }
    if (description !== currentDescription) {
      input.description = description;
    }
    if (lead !== currentLead) {
      input.lead = lead || null;
    }
    if (severityId !== currentSeverityId) {
      input.severityId = severityId;
    }
    // Check if asset IDs have changed
    const currentAssetIdsSet = new Set(currentAssetIds);
    const assetIdsChanged = selectedAssetIds.length !== currentAssetIds.length || !selectedAssetIds.every(id => currentAssetIdsSet.has(id));
    if (assetIdsChanged) {
      input.assetIds = selectedAssetIds;
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

  const toggleAsset = (assetId: string) => {
    setSelectedAssetIds(prev =>
      prev.includes(assetId)
        ? prev.filter(id => id !== assetId)
        : [...prev, assetId]
    );
  };

  const removeAsset = (assetId: string) => {
    setSelectedAssetIds(prev => prev.filter(id => id !== assetId));
  };

  const selectedAssets = assets.filter(asset => selectedAssetIds.includes(asset.id));
  const availableAssets = assets.filter(asset => !selectedAssetIds.includes(asset.id));

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (assetDropdownRef.current && !assetDropdownRef.current.contains(event.target as Node)) {
        setIsAssetDropdownOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

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

          {/* Severity field */}
          <div>
            <label htmlFor="severity" className="block text-sm font-medium text-gray-700 mb-1">
              Severity (optional)
            </label>
            <select
              id="severity"
              value={severityId}
              onChange={(e) => setSeverityId(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              {severities.map((severity) => {
                const style = getSeverityStyle(severity.level);
                return (
                  <option key={severity.id} value={severity.id}>
                    {style.icon} {severity.name}
                  </option>
                );
              })}
            </select>
            <p className="mt-1 text-sm text-gray-500">
              Select the severity level for this incident
            </p>
          </div>

          {/* Assets field */}
          {assets.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Assets (optional)
              </label>

              {/* Selected assets badges */}
              {selectedAssets.length > 0 && (
                <div className="flex flex-wrap gap-2 mb-2">
                  {selectedAssets.map((asset) => (
                    <span
                      key={asset.id}
                      className="inline-flex items-center gap-1 px-2 py-1 rounded text-sm bg-blue-100 text-blue-800"
                    >
                      {asset.name}
                      <button
                        type="button"
                        onClick={() => removeAsset(asset.id)}
                        className="hover:bg-blue-200 rounded-full p-0.5"
                      >
                        <X size={14} />
                      </button>
                    </span>
                  ))}
                </div>
              )}

              {/* Dropdown selector */}
              <div className="relative" ref={assetDropdownRef}>
                <button
                  type="button"
                  onClick={() => setIsAssetDropdownOpen(!isAssetDropdownOpen)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent flex items-center justify-between bg-white hover:bg-gray-50"
                >
                  <span className="text-sm text-gray-700">
                    {availableAssets.length > 0 ? 'Add assets...' : 'All assets selected'}
                  </span>
                  <ChevronDown size={16} className="text-gray-400" />
                </button>

                {isAssetDropdownOpen && availableAssets.length > 0 && (
                  <div className="absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded-md shadow-lg max-h-60 overflow-y-auto">
                    {availableAssets.map((asset) => (
                      <label
                        key={asset.id}
                        className="flex items-start gap-2 px-3 py-2 hover:bg-gray-50 border-b border-gray-100 last:border-b-0 cursor-pointer"
                      >
                        <input
                          type="checkbox"
                          checked={false}
                          onChange={() => toggleAsset(asset.id)}
                          className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                        />
                        <div className="flex-1">
                          <div className="font-medium text-sm">{asset.name}</div>
                          {asset.description && (
                            <div className="text-xs text-gray-500 mt-0.5">{asset.description}</div>
                          )}
                        </div>
                      </label>
                    ))}
                  </div>
                )}
              </div>

              <p className="mt-1 text-sm text-gray-500">
                Select the assets affected by this incident
              </p>
            </div>
          )}
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