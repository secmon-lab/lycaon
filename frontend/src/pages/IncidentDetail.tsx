import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@apollo/client/react';
import { format } from 'date-fns';
import { GET_INCIDENT } from '../graphql/queries';
import { CREATE_TASK, UPDATE_TASK, DELETE_TASK } from '../graphql/mutations';
import { cn } from '../lib/utils';
import { Button } from '../components/ui/Button';
import * as Dialog from '@radix-ui/react-dialog';
import {
  ArrowLeft,
  Edit2,
  Plus,
  Trash2,
  CheckCircle,
  Circle,
  Clock,
  User,
  MessageSquare,
  Calendar,
  Hash,
  AlertCircle,
  X,
} from 'lucide-react';

interface User {
  id: string;
  slackUserId: string;
  name: string;
  realName: string;
  displayName: string;
  email: string;
  avatarUrl: string;
}

interface Task {
  id: string;
  title: string;
  description: string;
  status: string;
  assigneeId: string | null;
  createdAt: string;
}

interface Incident {
  id: string;
  channelId: string;
  channelName: string;
  title: string;
  description: string;
  categoryId: string;
  categoryName?: string;
  status: string;
  createdBy: string;
  createdByUser?: User;
  createdAt: string;
  updatedAt: string;
  tasks: Task[];
}

const IncidentDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [isAddTaskOpen, setIsAddTaskOpen] = useState(false);
  const [newTaskTitle, setNewTaskTitle] = useState('');
  const [newTaskDescription, setNewTaskDescription] = useState('');

  const { loading, error, data, refetch } = useQuery<{ incident: Incident }>(
    GET_INCIDENT,
    {
      variables: { id },
      skip: !id,
    }
  );

  const [createTask] = useMutation(CREATE_TASK, {
    onCompleted: () => {
      refetch();
      setIsAddTaskOpen(false);
      setNewTaskTitle('');
      setNewTaskDescription('');
    },
  });

  const [updateTask] = useMutation(UPDATE_TASK, {
    onCompleted: () => refetch(),
  });

  const [deleteTask] = useMutation(DELETE_TASK, {
    onCompleted: () => refetch(),
  });


  const handleCreateTask = () => {
    if (!newTaskTitle.trim() || !id) return;
    
    createTask({
      variables: {
        input: {
          incidentId: id,
          title: newTaskTitle,
          description: newTaskDescription,
        },
      },
    });
  };

  const handleToggleTaskStatus = (taskId: string, currentStatus: string) => {
    const newStatus = currentStatus === 'completed' ? 'incompleted' : 'completed';
    updateTask({
      variables: {
        id: taskId,
        input: { status: newStatus },
      },
    });
  };

  const handleDeleteTask = (taskId: string) => {
    if (window.confirm('Are you sure you want to delete this task?')) {
      deleteTask({ variables: { id: taskId } });
    }
  };

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-slate-200 border-t-blue-600" />
          <p className="text-sm text-slate-500">Loading incident...</p>
        </div>
      </div>
    );
  }

  if (error || !data?.incident) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-red-600 mt-0.5" />
          <div>
            <h3 className="font-medium text-red-900">Error loading incident</h3>
            <p className="mt-1 text-sm text-red-700">
              {error?.message || 'Incident not found'}
            </p>
          </div>
        </div>
      </div>
    );
  }

  const incident = data.incident;
  const completedTasks = incident.tasks.filter(t => t.status === 'completed').length;
  const totalTasks = incident.tasks.length;
  const progress = totalTasks > 0 ? (completedTasks / totalTasks) * 100 : 0;

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'open':
        return 'bg-red-100 text-red-700 border-red-200';
      case 'closed':
        return 'bg-green-100 text-green-700 border-green-200';
      default:
        return 'bg-gray-100 text-gray-700 border-gray-200';
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate('/incidents')}
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-slate-900">
                Incident #{incident.id}
              </h1>
              <span className={cn(
                "inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium",
                getStatusColor(incident.status)
              )}>
                {incident.status}
              </span>
            </div>
            <p className="mt-1 text-sm text-slate-500">
              Created {format(new Date(incident.createdAt), 'MMM d, yyyy HH:mm')}
            </p>
          </div>
        </div>
        <Button size="sm" variant="outline" className="gap-2">
          <Edit2 className="h-4 w-4" />
          Edit
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Description */}
          <div className="rounded-lg border border-slate-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-slate-900 mb-3">
              {incident.title}
            </h2>
            <p className="text-sm text-slate-600 leading-relaxed">
              {incident.description || 'No description provided.'}
            </p>
          </div>

          {/* Tasks */}
          <div className="rounded-lg border border-slate-200 bg-white">
            <div className="flex items-center justify-between border-b border-slate-200 p-6">
              <div className="flex items-center gap-3">
                <h2 className="text-lg font-semibold text-slate-900">Tasks</h2>
                <span className="text-sm text-slate-500">
                  {completedTasks} of {totalTasks} completed
                </span>
              </div>
              <Dialog.Root open={isAddTaskOpen} onOpenChange={setIsAddTaskOpen}>
                <Dialog.Trigger asChild>
                  <Button size="sm" className="gap-2">
                    <Plus className="h-4 w-4" />
                    Add Task
                  </Button>
                </Dialog.Trigger>
                <Dialog.Portal>
                  <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
                  <Dialog.Content className="fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2 w-full max-w-md rounded-lg bg-white p-6 shadow-lg">
                    <Dialog.Title className="text-lg font-semibold text-slate-900 mb-4">
                      Add New Task
                    </Dialog.Title>
                    <div className="space-y-4">
                      <div>
                        <label className="text-sm font-medium text-slate-700">
                          Title
                        </label>
                        <input
                          type="text"
                          value={newTaskTitle}
                          onChange={(e) => setNewTaskTitle(e.target.value)}
                          className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                          placeholder="Enter task title"
                        />
                      </div>
                      <div>
                        <label className="text-sm font-medium text-slate-700">
                          Description (optional)
                        </label>
                        <textarea
                          value={newTaskDescription}
                          onChange={(e) => setNewTaskDescription(e.target.value)}
                          className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                          rows={3}
                          placeholder="Enter task description"
                        />
                      </div>
                    </div>
                    <div className="mt-6 flex justify-end gap-3">
                      <Dialog.Close asChild>
                        <Button variant="outline" size="sm">
                          Cancel
                        </Button>
                      </Dialog.Close>
                      <Button size="sm" onClick={handleCreateTask}>
                        Add Task
                      </Button>
                    </div>
                    <Dialog.Close asChild>
                      <button
                        className="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-white transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-slate-950 focus:ring-offset-2"
                        aria-label="Close"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </Dialog.Close>
                  </Dialog.Content>
                </Dialog.Portal>
              </Dialog.Root>
            </div>

            {/* Progress Bar */}
            {totalTasks > 0 && (
              <div className="px-6 py-3 border-b border-slate-200">
                <div className="h-2 bg-slate-100 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-blue-500 to-purple-600 transition-all duration-300"
                    style={{ width: `${progress}%` }}
                  />
                </div>
              </div>
            )}

            {/* Tasks List */}
            <div className="divide-y divide-slate-200">
              {incident.tasks.length === 0 ? (
                <div className="p-12 text-center">
                  <CheckCircle className="mx-auto h-12 w-12 text-slate-300" />
                  <p className="mt-3 text-sm text-slate-500">No tasks yet</p>
                  <p className="text-xs text-slate-400 mt-1">
                    Add your first task to start tracking progress
                  </p>
                </div>
              ) : (
                incident.tasks.map((task) => (
                  <div
                    key={task.id}
                    className="group flex items-center gap-4 p-4 hover:bg-slate-50 transition-colors"
                  >
                    <button
                      onClick={() => handleToggleTaskStatus(task.id, task.status)}
                      className="flex-shrink-0"
                    >
                      {task.status === 'completed' ? (
                        <CheckCircle className="h-5 w-5 text-green-600" />
                      ) : (
                        <Circle className="h-5 w-5 text-slate-400 hover:text-blue-600 transition-colors" />
                      )}
                    </button>
                    <div className="flex-1 min-w-0">
                      <p className={cn(
                        "text-sm font-medium text-slate-900",
                        task.status === 'completed' && "line-through text-slate-500"
                      )}>
                        {task.title}
                      </p>
                      {task.description && (
                        <p className="text-xs text-slate-500 mt-0.5 truncate">
                          {task.description}
                        </p>
                      )}
                    </div>
                    <span className={cn(
                      "flex-shrink-0 text-xs px-2 py-1 rounded-full",
                      task.status === 'completed'
                        ? "bg-green-100 text-green-700"
                        : "bg-slate-100 text-slate-600"
                    )}>
                      {task.status}
                    </span>
                    <button
                      onClick={() => handleDeleteTask(task.id)}
                      className="opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <Trash2 className="h-4 w-4 text-slate-400 hover:text-red-600 transition-colors" />
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

        {/* Sidebar - Metadata */}
        <div className="lg:col-span-1">
          <div className="rounded-lg border border-slate-200 bg-white p-6 sticky top-6">
            <h3 className="text-sm font-semibold text-slate-900 mb-4">Details</h3>
            <dl className="space-y-4">
              <div>
                <dt className="text-xs font-medium text-slate-500 flex items-center gap-1">
                  <MessageSquare className="h-3 w-3" />
                  Channel
                </dt>
                <dd className="mt-1 text-sm text-slate-900">
                  #{incident.channelName}
                </dd>
              </div>
              
              <div>
                <dt className="text-xs font-medium text-slate-500 flex items-center gap-1">
                  <Hash className="h-3 w-3" />
                  Category
                </dt>
                <dd className="mt-1 text-sm text-slate-900">
                  {incident.categoryName || incident.categoryId || 'Uncategorized'}
                </dd>
              </div>

              <div>
                <dt className="text-xs font-medium text-slate-500 flex items-center gap-1">
                  <User className="h-3 w-3" />
                  Created By
                </dt>
                <dd className="mt-1 text-sm text-slate-900 flex items-center gap-2">
                  {incident.createdByUser?.avatarUrl && (
                    <img 
                      src={incident.createdByUser.avatarUrl}
                      alt={incident.createdByUser.displayName || incident.createdByUser.name}
                      className="h-6 w-6 rounded-full"
                    />
                  )}
                  {incident.createdByUser?.displayName || 
                   incident.createdByUser?.realName || 
                   incident.createdByUser?.name || 
                   incident.createdBy}
                </dd>
              </div>

              <div>
                <dt className="text-xs font-medium text-slate-500 flex items-center gap-1">
                  <Calendar className="h-3 w-3" />
                  Created At
                </dt>
                <dd className="mt-1 text-sm text-slate-900">
                  {format(new Date(incident.createdAt), 'MMM d, yyyy')}
                  <br />
                  <span className="text-xs text-slate-500">
                    {format(new Date(incident.createdAt), 'HH:mm:ss')}
                  </span>
                </dd>
              </div>

              <div>
                <dt className="text-xs font-medium text-slate-500 flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  Last Updated
                </dt>
                <dd className="mt-1 text-sm text-slate-900">
                  {format(new Date(incident.updatedAt), 'MMM d, yyyy')}
                  <br />
                  <span className="text-xs text-slate-500">
                    {format(new Date(incident.updatedAt), 'HH:mm:ss')}
                  </span>
                </dd>
              </div>
            </dl>

            {/* Quick Actions */}
            <div className="mt-6 pt-6 border-t border-slate-200">
              <h4 className="text-xs font-semibold text-slate-900 mb-3">Quick Actions</h4>
              <div className="space-y-2">
                <Button variant="outline" size="sm" className="w-full justify-start gap-2">
                  <MessageSquare className="h-4 w-4" />
                  Open in Slack
                </Button>
                <Button variant="outline" size="sm" className="w-full justify-start gap-2">
                  <User className="h-4 w-4" />
                  Assign
                </Button>
                <Button 
                  variant="outline" 
                  size="sm" 
                  className="w-full justify-start gap-2 text-red-600 hover:text-red-700 hover:bg-red-50"
                >
                  <X className="h-4 w-4" />
                  Close Incident
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default IncidentDetail;