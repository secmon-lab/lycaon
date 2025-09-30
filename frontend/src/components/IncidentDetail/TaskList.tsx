import React, { useState } from 'react';
import { useMutation } from '@apollo/client/react';
import { Task, TaskStatus, Incident, CreateTaskInput } from '../../types/incident';
import { CREATE_TASK, DELETE_TASK } from '../../graphql/mutations';
import { GET_INCIDENT } from '../../graphql/queries';
import ConfirmationModal from '../common/ConfirmationModal';
import AssigneeSelector from '../common/AssigneeSelector';
import TaskEditModal from './TaskEditModal';
import {
  Plus,
  CheckCircle2,
  Circle,
  Edit3,
  Trash2,
  User,
  X,
  Check
} from 'lucide-react';

interface TaskListProps {
  incidentId: string;
  incident: Incident;
  tasks: Task[];
}

interface TaskFormData {
  title: string;
  description: string;
  assigneeId?: string;
}

const TaskList: React.FC<TaskListProps> = ({ incidentId, incident, tasks }) => {
  const [isCreating, setIsCreating] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [formData, setFormData] = useState<TaskFormData>({ title: '', description: '', assigneeId: undefined });

  const [createTask] = useMutation(CREATE_TASK, {
    refetchQueries: [{ query: GET_INCIDENT, variables: { id: incidentId } }]
  });


  const [deleteTask] = useMutation(DELETE_TASK, {
    refetchQueries: [{ query: GET_INCIDENT, variables: { id: incidentId } }]
  });

  const handleCreateTask = async (data: TaskFormData) => {
    if (!data.title.trim()) return;

    try {
      const input: CreateTaskInput = {
        incidentId,
        title: data.title.trim(),
        description: data.description.trim()
      };

      if (data.assigneeId !== undefined) {
        input.assigneeId = data.assigneeId || null;
      }

      await createTask({
        variables: { input }
      });
      setFormData({ title: '', description: '', assigneeId: undefined });
      setIsCreating(false);
    } catch (error) {
      console.error('Failed to create task:', error);
    }
  };




  const [deletingTaskId, setDeletingTaskId] = useState<string | null>(null);

  const confirmDeleteTask = async () => {
    if (!deletingTaskId) return;
    
    try {
      await deleteTask({
        variables: { id: deletingTaskId }
      });
      setDeletingTaskId(null);
    } catch (error) {
      console.error('Failed to delete task:', error);
      setDeletingTaskId(null);
    }
  };

  const TaskForm: React.FC<{
    onSubmit: (data: TaskFormData) => Promise<void>;
    onCancel: () => void
  }> = ({ onSubmit, onCancel }) => {
    const [localData, setLocalData] = useState<TaskFormData>(formData);

    const handleSubmit = async (e: React.FormEvent) => {
      e.preventDefault();
      if (!localData.title.trim()) return;
      await onSubmit(localData);
    };

    return (
      <form onSubmit={handleSubmit} className="border border-slate-200 rounded-lg p-3 bg-white">
        <div className="space-y-2">
          <input
            type="text"
            placeholder="Task title..."
            value={localData.title}
            onChange={(e) => setLocalData({ ...localData, title: e.target.value })}
            className="w-full px-2.5 py-1.5 border border-slate-200 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 text-sm"
            autoFocus
          />
          <textarea
            placeholder="Task description (optional)..."
            value={localData.description}
            onChange={(e) => setLocalData({ ...localData, description: e.target.value })}
            className="w-full px-2.5 py-1.5 border border-slate-200 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 text-sm resize-none"
            rows={2}
          />

          {/* Assignee Selector */}
          <div className="space-y-1">
            <label className="text-xs text-slate-600 font-medium">Assignee</label>
            <AssigneeSelector
              channelId={incident.channelId}
              selectedUserId={localData.assigneeId}
              onAssigneeChange={(userId) => setLocalData({ ...localData, assigneeId: userId })}
              placeholder="Select assignee..."
              className="text-sm"
            />
          </div>
          <div className="flex gap-1.5">
            <button
              type="submit"
              disabled={!localData.title.trim()}
              className="inline-flex items-center gap-1 px-2.5 py-1 bg-blue-600 text-white text-xs font-medium rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <Check className="h-3 w-3" />
              Create
            </button>
            <button
              type="button"
              onClick={onCancel}
              className="inline-flex items-center gap-1 px-2.5 py-1 bg-slate-100 text-slate-700 text-xs font-medium rounded hover:bg-slate-200 transition-colors"
            >
              <X className="h-3 w-3" />
              Cancel
            </button>
          </div>
        </div>
      </form>
    );
  };


  const getStatusStyles = (status: TaskStatus) => {
    const isCompleted = status === TaskStatus.COMPLETED;
    const isFollowUp = status === TaskStatus.FOLLOW_UP;

    let borderClass = 'border-slate-200';
    let bgClass = 'bg-white';

    if (isCompleted) {
      borderClass = 'border-slate-100';
      bgClass = 'bg-slate-50';
    } else if (isFollowUp) {
      borderClass = 'border-orange-200';
      bgClass = 'bg-orange-50';
    }

    return { borderClass, bgClass, isCompleted };
  };

  const TaskItem: React.FC<{ task: Task }> = ({ task }) => {
    const { borderClass, bgClass, isCompleted } = getStatusStyles(task.status);

    return (
      <div className={`group border ${borderClass} rounded-md ${bgClass} hover:border-slate-300 transition-all`}>
        <div className="flex items-center gap-3 px-2.5 py-1.5">
          <div className="flex-1 min-w-0 flex items-center gap-3">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h4 className={`text-sm font-medium ${isCompleted ? 'line-through text-slate-400' : 'text-slate-900'}`}>
                  {task.title}
                </h4>
                {task.assigneeUser && (
                  <div className="flex items-center gap-1">
                    {task.assigneeUser.avatarUrl ? (
                      <img
                        src={task.assigneeUser.avatarUrl}
                        alt={task.assigneeUser.name}
                        className="h-4 w-4 rounded-full"
                      />
                    ) : (
                      <div className="h-4 w-4 rounded-full bg-slate-200 flex items-center justify-center">
                        <User className="h-2.5 w-2.5 text-slate-500" />
                      </div>
                    )}
                    <span className="text-xs text-slate-500 font-medium">
                      {task.assigneeUser.name || task.assigneeUser.displayName || task.assigneeUser.realName}
                    </span>
                  </div>
                )}
              </div>
              {task.description && (
                <p className={`text-xs text-slate-500 mt-0.5 ${isCompleted ? 'line-through text-slate-400' : ''}`}>
                  {task.description}
                </p>
              )}
            </div>
          </div>

          <div className="flex items-center gap-0.5 flex-shrink-0">
            <button
              onClick={() => setEditingTask(task)}
              className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
              title="Edit task"
            >
              <Edit3 className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => setDeletingTaskId(task.id)}
              className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
              title="Delete task"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </div>
    );
  };

  // Normalize GraphQL enum values to frontend enum values
  const normalizeTaskStatus = (status: string): TaskStatus => {
    switch (status.toLowerCase()) {
      case 'todo':
        return TaskStatus.TODO;
      case 'follow_up':
        return TaskStatus.FOLLOW_UP;
      case 'completed':
        return TaskStatus.COMPLETED;
      default:
        return TaskStatus.TODO; // fallback
    }
  };

  // Normalize tasks with proper status values
  const normalizedTasks = (tasks || []).map(task => ({
    ...task,
    status: normalizeTaskStatus(task.status)
  }));

  // Group tasks by status
  const todoTasks = normalizedTasks.filter(task => task.status === TaskStatus.TODO);
  const followUpTasks = normalizedTasks.filter(task => task.status === TaskStatus.FOLLOW_UP);
  const completedTasks = normalizedTasks.filter(task => task.status === TaskStatus.COMPLETED);
  // Count active tasks for potential future use
  // const activeTasks = todoTasks.length + followUpTasks.length;

  return (
    <div className="space-y-2">
      {/* Header */}
      <div className="flex items-center justify-between mb-1">
        <div className="flex items-center gap-2">
          <h3 className="text-base font-semibold text-slate-900">Tasks</h3>
          <div className="flex items-center gap-1.5 text-xs">
            {todoTasks.length > 0 && (
              <span className="px-2 py-0.5 bg-blue-50 text-blue-600 rounded-full font-medium">
                {todoTasks.length} to do
              </span>
            )}
            {followUpTasks.length > 0 && (
              <span className="px-2 py-0.5 bg-orange-50 text-orange-600 rounded-full font-medium">
                {followUpTasks.length} follow up
              </span>
            )}
            {completedTasks.length > 0 && (
              <span className="px-2 py-0.5 bg-green-50 text-green-600 rounded-full font-medium">
                {completedTasks.length} completed
              </span>
            )}
          </div>
        </div>
        
        {!isCreating && (
          <button
            onClick={() => setIsCreating(true)}
            className="inline-flex items-center gap-1 px-2.5 py-1 bg-blue-600 text-white text-xs font-medium rounded hover:bg-blue-700 transition-colors"
          >
            <Plus className="h-3.5 w-3.5" />
            Add Task
          </button>
        )}
      </div>

      {/* Create new task form */}
      {isCreating && (
        <TaskForm
          onSubmit={handleCreateTask}
          onCancel={() => {
            setIsCreating(false);
            setFormData({ title: '', description: '', assigneeId: undefined });
          }}
        />
      )}

      {/* Tasks list grouped by status */}
      <div className="space-y-3">
        {/* To Do tasks */}
        {todoTasks.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-slate-600 mb-2 px-1 flex items-center gap-1">
              <Circle className="h-3 w-3 text-slate-400" />
              To Do ({todoTasks.length})
            </h4>
            <div className="space-y-2">
              {todoTasks.map(task => (
                <TaskItem key={task.id} task={task} />
              ))}
            </div>
          </div>
        )}

        {/* Follow Up tasks */}
        {followUpTasks.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-orange-600 mb-2 px-1 flex items-center gap-1">
              <Circle className="h-3 w-3 text-orange-500" />
              Follow Up ({followUpTasks.length})
            </h4>
            <div className="space-y-2">
              {followUpTasks.map(task => (
                <TaskItem key={task.id} task={task} />
              ))}
            </div>
          </div>
        )}

        {/* Completed tasks */}
        {completedTasks.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-green-600 mb-2 px-1 flex items-center gap-1">
              <CheckCircle2 className="h-3 w-3 text-green-500" />
              Completed ({completedTasks.length})
            </h4>
            <div className="space-y-2">
              {completedTasks.map(task => (
                <TaskItem key={task.id} task={task} />
              ))}
            </div>
          </div>
        )}

        {/* Empty state */}
        {(!tasks || tasks.length === 0) && !isCreating && (
          <div className="text-center py-6 text-slate-400">
            <div className="mb-2">
              <Circle className="h-10 w-10 mx-auto text-slate-300" />
            </div>
            <p className="text-sm font-medium">No tasks yet</p>
            <p className="text-xs mt-0.5">Create your first task to get started</p>
          </div>
        )}
      </div>

      {/* Delete confirmation modal */}
      <ConfirmationModal
        isOpen={!!deletingTaskId}
        onClose={() => setDeletingTaskId(null)}
        onConfirm={confirmDeleteTask}
        title="Delete Task"
        message="Are you sure you want to delete this task? This action cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
      />

      {/* Edit task modal */}
      {editingTask && (
        <TaskEditModal
          isOpen={!!editingTask}
          onClose={() => setEditingTask(null)}
          task={editingTask}
          incident={incident}
        />
      )}
    </div>
  );
};

export default TaskList;