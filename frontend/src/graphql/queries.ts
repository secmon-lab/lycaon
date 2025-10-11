import { gql } from '@apollo/client';

// Fragment for user fields
export const USER_FIELDS = gql`
  fragment UserFields on User {
    id
    slackUserId
    name
    realName
    displayName
    email
    avatarUrl
  }
`;

// Fragment for status history fields
export const STATUS_HISTORY_FIELDS = gql`
  fragment StatusHistoryFields on StatusHistory {
    id
    incidentId
    status
    changedBy {
      ...UserFields
    }
    changedAt
    note
  }
  ${USER_FIELDS}
`;

// Fragment for incident fields
export const INCIDENT_FIELDS = gql`
  fragment IncidentFields on Incident {
    id
    channelId
    channelName
    title
    description
    categoryId
    categoryName
    severityId
    severityName
    severityLevel
    assetIds
    assetNames
    status
    lead
    leadUser {
      ...UserFields
    }
    originChannelId
    originChannelName
    teamId
    createdBy
    createdByUser {
      ...UserFields
    }
    createdAt
    updatedAt
    private
    statusHistories {
      ...StatusHistoryFields
    }
  }
  ${USER_FIELDS}
  ${STATUS_HISTORY_FIELDS}
`;

// Fragment for task fields
export const TASK_FIELDS = gql`
  fragment TaskFields on Task {
    id
    incidentId
    title
    description
    status
    assigneeId
    assigneeUser {
      ...UserFields
    }
    createdBy
    channelId
    messageTs
    createdAt
    updatedAt
    completedAt
  }
  ${USER_FIELDS}
`;

// Query to get incidents list with pagination
export const GET_INCIDENTS = gql`
  ${INCIDENT_FIELDS}
  query GetIncidents($first: Int, $after: String) {
    incidents(first: $first, after: $after) {
      edges {
        node {
          ...IncidentFields
        }
        cursor
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
      totalCount
    }
  }
`;

// Query to get a single incident with tasks
export const GET_INCIDENT = gql`
  ${INCIDENT_FIELDS}
  ${TASK_FIELDS}
  query GetIncident($id: ID!) {
    incident(id: $id) {
      ...IncidentFields
      tasks {
        ...TaskFields
      }
    }
  }
`;

// Query to get tasks for an incident
export const GET_TASKS = gql`
  ${TASK_FIELDS}
  query GetTasks($incidentId: ID!) {
    tasks(incidentId: $incidentId) {
      ...TaskFields
    }
  }
`;

// Query to get a single task
export const GET_TASK = gql`
  ${TASK_FIELDS}
  query GetTask($id: ID!) {
    task(id: $id) {
      ...TaskFields
    }
  }
`;

// Query to get status history for an incident
export const GET_INCIDENT_STATUS_HISTORY = gql`
  ${STATUS_HISTORY_FIELDS}
  query GetIncidentStatusHistory($incidentId: ID!) {
    incidentStatusHistory(incidentId: $incidentId) {
      ...StatusHistoryFields
    }
  }
`;

// Query to get channel members for assignee selection
export const GET_CHANNEL_MEMBERS = gql`
  ${USER_FIELDS}
  query GetChannelMembers($channelId: String!) {
    channelMembers(channelId: $channelId) {
      ...UserFields
    }
  }
`;

// Query to get all severities
export const GET_SEVERITIES = gql`
  query GetSeverities {
    severities {
      id
      name
      description
      level
    }
  }
`;

// Query to get all assets
export const GET_ASSETS = gql`
  query GetAssets {
    assets {
      id
      name
      description
    }
  }
`;

// Dashboard queries

// Query to get recent open incidents grouped by date
export const GET_RECENT_OPEN_INCIDENTS = gql`
  query GetRecentOpenIncidents($days: Int) {
    recentOpenIncidents(days: $days) {
      date
      incidents {
        id
        title
        description
        severityId
        severityName
        severityLevel
        status
        lead
        leadUser {
          id
          name
          displayName
          avatarUrl
        }
        createdAt
      }
    }
  }
`;

// Query to get incident trend by severity
export const GET_INCIDENT_TREND_BY_SEVERITY = gql`
  query GetIncidentTrendBySeverity($weeks: Int) {
    incidentTrendBySeverity(weeks: $weeks) {
      weekStart
      weekEnd
      weekLabel
      severityCounts {
        severityId
        severityName
        severityLevel
        count
      }
    }
  }
`;