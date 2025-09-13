import { gql } from '@apollo/client';
import { INCIDENT_FIELDS, TASK_FIELDS } from './queries';

// Mutation to update an incident
export const UPDATE_INCIDENT = gql`
  ${INCIDENT_FIELDS}
  mutation UpdateIncident($id: ID!, $input: UpdateIncidentInput!) {
    updateIncident(id: $id, input: $input) {
      ...IncidentFields
    }
  }
`;

// Mutation to create a task
export const CREATE_TASK = gql`
  ${TASK_FIELDS}
  mutation CreateTask($input: CreateTaskInput!) {
    createTask(input: $input) {
      ...TaskFields
    }
  }
`;

// Mutation to update a task
export const UPDATE_TASK = gql`
  ${TASK_FIELDS}
  mutation UpdateTask($id: ID!, $input: UpdateTaskInput!) {
    updateTask(id: $id, input: $input) {
      ...TaskFields
    }
  }
`;

// Mutation to delete a task
export const DELETE_TASK = gql`
  mutation DeleteTask($id: ID!) {
    deleteTask(id: $id)
  }
`;