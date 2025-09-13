// Incident status types
export enum IncidentStatus {
  TRIAGE = 'triage',
  HANDLING = 'handling',
  MONITORING = 'monitoring',
  CLOSED = 'closed'
}

// Status history type
export interface StatusHistory {
  id: string;
  incidentId: string;
  status: IncidentStatus;
  changedBy: User;
  changedAt: string;
  note?: string;
}

// User type
export interface User {
  id: string;
  slackUserId: string;
  name: string;
  realName: string;
  displayName: string;
  email: string;
  avatarUrl: string;
}

// Extended incident type with status fields
export interface Incident {
  id: string;
  channelId: string;
  channelName: string;
  title: string;
  description: string;
  categoryId: string;
  categoryName: string;
  status: IncidentStatus;
  lead: string;
  leadUser?: User;
  originChannelId: string;
  originChannelName: string;
  createdBy: string;
  createdByUser?: User;
  createdAt: string;
  updatedAt: string;
  statusHistories: StatusHistory[];
  tasks: Task[];
}

// Task types
export enum TaskStatus {
  INCOMPLETED = 'incompleted',
  COMPLETED = 'completed'
}

export interface Task {
  id: string;
  incidentId: string;
  title: string;
  description: string;
  status: TaskStatus;
  assigneeId?: string;
  assigneeUser?: User;
  createdBy: string;
  channelId: string;
  messageTs: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

// Type guard to check if a value is a valid IncidentStatus
export const isIncidentStatus = (value: unknown): value is IncidentStatus => {
  return Object.values(IncidentStatus).includes(value as IncidentStatus);
};

// Safe conversion function with validation
export const toIncidentStatus = (value: unknown): IncidentStatus | null => {
  if (isIncidentStatus(value)) {
    return value;
  }
  return null;
};

// Status display configurations
export const STATUS_CONFIG = {
  [IncidentStatus.TRIAGE]: {
    label: 'Triage',
    color: '#f59e0b', // Amber
    icon: '🟡',
    description: 'Initial evaluation and classification'
  },
  [IncidentStatus.HANDLING]: {
    label: 'Handling',
    color: '#f44336', // Red
    icon: '🔴',
    description: 'Incident response in progress'
  },
  [IncidentStatus.MONITORING]: {
    label: 'Monitoring',
    color: '#ff9800', // Orange
    icon: '🟠',
    description: 'Monitoring after response'
  },
  [IncidentStatus.CLOSED]: {
    label: 'Closed',
    color: '#4caf50', // Green
    icon: '🟢',
    description: 'Incident resolved'
  }
} as const;

// Helper functions
export const getStatusConfig = (status: IncidentStatus | string | null | undefined) => {
  if (!status || typeof status !== 'string') {
    return {
      label: 'Not Set',
      color: '#6b7280',
      icon: '❓',
      description: 'Status has not been set'
    };
  }
  
  // Use type guard to safely check if it's a valid status
  const validStatus = toIncidentStatus(status);
  if (!validStatus) {
    return {
      label: 'Not Set',
      color: '#6b7280',
      icon: '❓',
      description: 'Status has not been set'
    };
  }
  
  return STATUS_CONFIG[validStatus];
};

export const getStatusLabel = (status: IncidentStatus | string | null | undefined) => {
  const config = getStatusConfig(status);
  return config.label;
};

export const getStatusColor = (status: IncidentStatus | string | null | undefined) => {
  const config = getStatusConfig(status);
  return config.color;
};

export const getStatusIcon = (status: IncidentStatus | string | null | undefined) => {
  const config = getStatusConfig(status);
  return config.icon;
};