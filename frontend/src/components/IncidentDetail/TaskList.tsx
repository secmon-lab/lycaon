import React, { useState } from 'react';
import { useMutation } from '@apollo/client/react';
import { Task, TaskStatus } from '../../types/incident';
import { CREATE_TASK, UPDATE_TASK, DELETE_TASK } from '../../graphql/mutations';
import { GET_INCIDENT } from '../../graphql/queries';
import ConfirmationModal from '../common/ConfirmationModal';
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
  tasks: Task[];
}

interface TaskFormData {
  title: string;
  description: string;
}

const TaskList: React.FC<TaskListProps> = ({ incidentId, tasks }) => {
  const [isCreating, setIsCreating] = useState(false);
  const [editingTask, setEditingTask] = useState<string | null>(null);
  const [formData, setFormData] = useState<TaskFormData>({ title: '', description: '' });

  const [createTask] = useMutation(CREATE_TASK, {
    refetchQueries: [{ query: GET_INCIDENT, variables: { id: incidentId } }]
  });

  const [updateTask] = useMutation(UPDATE_TASK, {
    refetchQueries: [{ query: GET_INCIDENT, variables: { id: incidentId } }]
  });

  const [deleteTask] = useMutation(DELETE_TASK, {
    refetchQueries: [{ query: GET_INCIDENT, variables: { id: incidentId } }]
  });

  const handleCreateTask = async () => {
    if (!formData.title.trim()) return;

    try {
      await createTask({
        variables: {
          input: {
            incidentId,
            title: formData.title.trim(),
            description: formData.description.trim()
          }
        }
      });
      setFormData({ title: '', description: '' });
      setIsCreating(false);
    } catch (error) {
      console.error('Failed to create task:', error);
    }
  };

  const handleUpdateTask = async (taskId: string, updates: Partial<Task>) => {
    try {
      await updateTask({
        variables: {
          id: taskId,
          input: updates
        }
      });
      setEditingTask(null);
    } catch (error) {
      console.error('Failed to update task:', error);
    }
  };

  const handleToggleTaskStatus = async (task: Task) => {
    const newStatus = task.status === TaskStatus.COMPLETED 
      ? TaskStatus.INCOMPLETED 
      : TaskStatus.COMPLETED;
    
    await handleUpdateTask(task.id, { status: newStatus });
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
    task?: Task; 
    onSubmit: (data: TaskFormData) => void; 
    onCancel: () => void 
  }> = ({ task, onSubmit, onCancel }) => {
    const [localData, setLocalData] = useState<TaskFormData>(
      task ? { title: task.title, description: task.description } : formData
    );

    const handleSubmit = (e: React.FormEvent) => {
      e.preventDefault();
      if (!localData.title.trim()) return;
      onSubmit(localData);
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
          <div className="flex gap-1.5">
            <button
              type="submit"
              disabled={!localData.title.trim()}
              className="inline-flex items-center gap-1 px-2.5 py-1 bg-blue-600 text-white text-xs font-medium rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <Check className="h-3 w-3" />
              {task ? 'Update' : 'Create'}
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

  const TaskItem: React.FC<{ task: Task }> = ({ task }) => {
    const isCompleted = task.status === TaskStatus.COMPLETED;
    const isEditing = editingTask === task.id;

    if (isEditing) {
      return (
        <TaskForm
          task={task}
          onSubmit={(data) => handleUpdateTask(task.id, data)}
          onCancel={() => setEditingTask(null)}
        />
      );
    }

    return (
      <div className={`group border border-slate-200 rounded-lg bg-white hover:border-slate-300 hover:shadow-sm transition-all ${
        isCompleted ? 'bg-slate-50 border-slate-100' : ''
      }`}>
        <div className="flex items-start gap-2 p-3">
          <button
            onClick={() => handleToggleTaskStatus(task)}
            className="mt-0.5 flex-shrink-0 text-slate-400 hover:text-blue-600 transition-colors"
          >
            {isCompleted ? (
              <CheckCircle2 className="h-5 w-5 text-green-500" />
            ) : (
              <Circle className="h-5 w-5 hover:text-blue-500" />
            )}
          </button>
          
          <div className="flex-1 min-w-0 mr-2">
            <h4 className={`text-sm font-medium ${isCompleted ? 'line-through text-slate-400' : 'text-slate-900'}`}>
              {task.title}
            </h4>
            {task.description && (
              <p className={`mt-0.5 text-xs text-slate-600 ${isCompleted ? 'line-through text-slate-400' : ''}`}>
                {task.description}
              </p>
            )}
            
            <div className="mt-1.5 flex items-center gap-1.5">
              {task.assigneeUser ? (
                <>
                  {task.assigneeUser.avatarUrl ? (
                    <img 
                      src={task.assigneeUser.avatarUrl} 
                      alt={task.assigneeUser.name}
                      className="h-5 w-5 rounded-full"
                    />
                  ) : (
                    <div className="h-5 w-5 rounded-full bg-slate-200 flex items-center justify-center">
                      <User className="h-3 w-3 text-slate-500" />
                    </div>
                  )}
                  <span className="text-xs text-slate-500">{task.assigneeUser.name || task.assigneeUser.displayName || task.assigneeUser.realName}</span>
                </>
              ) : (
                <>
                  <div className="h-5 w-5 rounded-full bg-slate-100 flex items-center justify-center">
                    <User className="h-3 w-3 text-slate-400" />
                  </div>
                  <span className="text-xs text-slate-400">No assign</span>
                </>
              )}
            </div>
          </div>

          <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
            <button
              onClick={() => setEditingTask(task.id)}
              className="p-1 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
              title="Edit task"
            >
              <Edit3 className="h-3 w-3" />
            </button>
            <button
              onClick={() => setDeletingTaskId(task.id)}
              className="p-1 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
              title="Delete task"
            >
              <Trash2 className="h-3 w-3" />
            </button>
          </div>
        </div>
      </div>
    );
  };

  const completedTasks = (tasks || []).filter(task => task.status === TaskStatus.COMPLETED);
  const incompleteTasks = (tasks || []).filter(task => task.status === TaskStatus.INCOMPLETED);

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <h3 className="text-base font-semibold text-slate-900">Tasks</h3>
          <div className="flex items-center gap-1.5 text-xs">
            <span className="px-2 py-0.5 bg-blue-50 text-blue-600 rounded-full font-medium">
              {incompleteTasks.length} active
            </span>
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
            setFormData({ title: '', description: '' });
          }}
        />
      )}

      {/* Tasks list */}
      <div className="space-y-2">
        {/* Active tasks */}
        {incompleteTasks.map(task => (
          <TaskItem key={task.id} task={task} />
        ))}

        {/* Completed tasks */}
        {completedTasks.length > 0 && (
          <>
            {incompleteTasks.length > 0 && (
              <div className="pt-2 mt-2">
                <h4 className="text-xs font-medium text-slate-500 mb-2">Completed Tasks</h4>
              </div>
            )}
            <div className="space-y-2">
              {completedTasks.map(task => (
                <TaskItem key={task.id} task={task} />
              ))}
            </div>
          </>
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
    </div>
  );
};

export default TaskList;