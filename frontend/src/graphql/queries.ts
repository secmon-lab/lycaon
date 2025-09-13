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
    status
    originChannelId
    originChannelName
    createdBy
    createdByUser {
      ...UserFields
    }
    createdAt
    updatedAt
  }
  ${USER_FIELDS}
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
    createdBy
    channelId
    messageTs
    createdAt
    updatedAt
    completedAt
  }
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