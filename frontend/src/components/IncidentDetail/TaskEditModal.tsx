import React, { useState } from 'react';
import { useMutation } from '@apollo/client/react';
import { X, Check, User } from 'lucide-react';
import { Task, TaskStatus, Incident, UpdateTaskInput } from '../../types/incident';
import { UPDATE_TASK } from '../../graphql/mutations';
import { GET_INCIDENT } from '../../graphql/queries';
import AssigneeSelector from '../common/AssigneeSelector';

interface TaskEditModalProps {
  isOpen: boolean;
  onClose: () => void;
  task: Task;
  incident: Incident;
}

interface TaskFormData {
  title: string;
  description: string;
  status: TaskStatus;
  assigneeId?: string;
}

const TaskEditModal: React.FC<TaskEditModalProps> = ({
  isOpen,
  onClose,
  task,
  incident
}) => {
  const [formData, setFormData] = useState<TaskFormData>({
    title: task.title,
    description: task.description,
    status: task.status,
    assigneeId: task.assigneeId
  });

  const [updateTask, { loading }] = useMutation(UPDATE_TASK, {
    refetchQueries: [{ query: GET_INCIDENT, variables: { id: incident.id } }]
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.title.trim()) return;

    try {
      const updates: UpdateTaskInput = {
        title: formData.title.trim(),
        description: formData.description.trim(),
        status: formData.status
      };

      // Handle assignee update, including clearing it.
      if (formData.assigneeId !== task.assigneeId) {
        updates.assigneeId = formData.assigneeId || null;
      }

      await updateTask({
        variables: {
          id: task.id,
          input: updates
        }
      });
      onClose();
    } catch (error) {
      console.error('Failed to update task:', error);
    }
  };

  const handleStatusChange = (status: TaskStatus) => {
    setFormData({ ...formData, status });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black bg-opacity-50" onClick={onClose} />
      <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full mx-4 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <h3 className="text-lg font-semibold">Edit Task</h3>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <X className="h-5 w-5 text-gray-500" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-4">
          <div className="space-y-4">
            {/* Title */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Title *
              </label>
              <input
                type="text"
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="Task title..."
                required
                autoFocus
              />
            </div>

            {/* Description */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 resize-none"
                placeholder="Task description (optional)..."
                rows={3}
              />
            </div>

            {/* Status */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Status
              </label>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => handleStatusChange(TaskStatus.INCOMPLETED)}
                  className={`flex-1 py-2 px-3 text-sm font-medium rounded-md border transition-colors ${
                    formData.status === TaskStatus.INCOMPLETED
                      ? 'bg-blue-50 border-blue-300 text-blue-700'
                      : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  To Do
                </button>
                <button
                  type="button"
                  onClick={() => handleStatusChange(TaskStatus.COMPLETED)}
                  className={`flex-1 py-2 px-3 text-sm font-medium rounded-md border transition-colors ${
                    formData.status === TaskStatus.COMPLETED
                      ? 'bg-green-50 border-green-300 text-green-700'
                      : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  Completed
                </button>
              </div>
            </div>

            {/* Assignee */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Assignee
              </label>
              <AssigneeSelector
                channelId={incident.channelId}
                selectedUserId={formData.assigneeId}
                onAssigneeChange={(userId) => setFormData({ ...formData, assigneeId: userId })}
                placeholder="Select assignee..."
              />
            </div>

            {/* Current assignee display for reference */}
            {task.assigneeUser && (
              <div className="text-xs text-gray-500">
                <span>Current assignee: </span>
                <div className="inline-flex items-center gap-1 mt-1">
                  {task.assigneeUser.avatarUrl ? (
                    <img
                      src={task.assigneeUser.avatarUrl}
                      alt={task.assigneeUser.name}
                      className="h-4 w-4 rounded-full"
                    />
                  ) : (
                    <div className="h-4 w-4 rounded-full bg-gray-200 flex items-center justify-center">
                      <User className="h-2 w-2 text-gray-500" />
                    </div>
                  )}
                  <span>{task.assigneeUser.displayName || task.assigneeUser.name}</span>
                </div>
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 mt-6 pt-4 border-t">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
              disabled={loading}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!formData.title.trim() || loading}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
            >
              {loading ? (
                <>
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  Updating...
                </>
              ) : (
                <>
                  <Check className="h-4 w-4" />
                  Update Task
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default TaskEditModal;