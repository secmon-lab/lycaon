package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/controller/graphql"
	controller "github.com/secmon-lab/lycaon/pkg/controller/http"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)

// Helper to create mock clients for HTTP tests
func createMockClients() (gollem.LLMClient, *mocks.SlackClientMock) {
	return &mock.LLMClientMock{}, &mocks.SlackClientMock{
		AuthTestContextFunc: func(ctx context.Context) (*slackgo.AuthTestResponse, error) {
			return &slackgo.AuthTestResponse{UserID: "U_TEST_BOT", User: "test-bot"}, nil
		},
	}
}

func TestServerHealthCheck(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)
	mockLLM, mockSlack := createMockClients()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, model.GetDefaultCategories())
	gt.NoError(t, err).Required()
	categories := model.GetDefaultCategories()
	incidentUC := usecase.NewIncident(repo, nil, categories, nil)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)
	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, mockSlack)

	server, err := controller.NewServer(
		ctx,
		":8080",
		slackConfig,
		categories,
		repo,
		authUC,
		messageUC,
		incidentUC,
		taskUC,
		slackInteractionUC,
		mockSlack,
		"",
	)
	gt.NoError(t, err).Required()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	server.Server.Handler.ServeHTTP(w, req)

	// Assert
	gt.Equal(t, http.StatusOK, w.Code)
	gt.True(t, strings.Contains(w.Body.String(), "healthy"))
	gt.True(t, strings.Contains(w.Body.String(), "lycaon"))
}

func TestServerFallbackHome(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)
	mockLLM, mockSlack := createMockClients()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, model.GetDefaultCategories())
	gt.NoError(t, err).Required()
	categories := model.GetDefaultCategories()
	incidentUC := usecase.NewIncident(repo, nil, categories, nil)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)
	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, mockSlack)

	server, err := controller.NewServer(
		ctx,
		":8080",
		slackConfig,
		categories,
		repo,
		authUC,
		messageUC,
		incidentUC,
		taskUC,
		slackInteractionUC,
		mockSlack,
		"",
	)
	gt.NoError(t, err).Required()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Execute
	server.Server.Handler.ServeHTTP(w, req)

	// Assert
	gt.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	t.Logf("Response body: %s", body)
	// Check that we got an HTML response
	gt.True(t, strings.Contains(body, "<!DOCTYPE html>") || strings.Contains(body, "<!doctype html>"))
	gt.True(t, strings.Contains(body, "<html"))
	gt.True(t, strings.Contains(body, "</html>"))
}

// GraphQL E2E Tests

type GraphQLResponse struct {
	Data   interface{}    `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

func setupGraphQLTestServer(t *testing.T) (*httptest.Server, *repository.Memory) {
	t.Helper()

	// Create memory repository
	repo := repository.NewMemory()

	// Create mock slack client
	_, mockSlack := createMockClients()

	// Create minimal categories config
	categories := model.GetDefaultCategories()

	// Create use cases
	incidentUC := usecase.NewIncident(repo, mockSlack, categories, nil)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)

	// Create Auth UC with mock Slack config
	slackConfig := &config.SlackConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	authUC := usecase.NewAuth(context.Background(), repo, slackConfig)

	// Create GraphQL resolver
	resolver := graphql.NewResolver(repo, mockSlack, &graphql.UseCases{
		IncidentUC: incidentUC,
		TaskUC:     taskUC,
		AuthUC:     authUC,
	}, categories)

	// Create GraphQL server directly without auth middleware
	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}))

	// Create simple HTTP mux for test
	mux := http.NewServeMux()
	mux.Handle("/graphql", srv)

	// Create test server
	testServer := httptest.NewServer(mux)

	return testServer, repo.(*repository.Memory)
}

func executeGraphQL(t *testing.T, server *httptest.Server, query string, variables map[string]interface{}) *GraphQLResponse {
	t.Helper()

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	gt.NoError(t, err)

	resp, err := http.Post(
		server.URL+"/graphql",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	gt.NoError(t, err)
	defer resp.Body.Close()

	var graphqlResp GraphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&graphqlResp)
	gt.NoError(t, err)

	return &graphqlResp
}

// Test basic GraphQL endpoint connectivity
func TestGraphQL_Endpoint(t *testing.T) {
	server, _ := setupGraphQLTestServer(t)
	defer server.Close()

	// Test basic introspection query
	query := `
		query {
			__schema {
				types {
					name
				}
			}
		}
	`

	resp := executeGraphQL(t, server, query, nil)
	gt.Equal(t, len(resp.Errors), 0)
	gt.NotNil(t, resp.Data)
}

// Test empty incidents query
func TestGraphQL_EmptyIncidents(t *testing.T) {
	server, _ := setupGraphQLTestServer(t)
	defer server.Close()

	query := `
		query {
			incidents(first: 10) {
				edges {
					node {
						id
						title
						channelName
					}
				}
				pageInfo {
					hasNextPage
					hasPreviousPage
				}
				totalCount
			}
		}
	`

	resp := executeGraphQL(t, server, query, nil)

	if len(resp.Errors) > 0 {
		t.Logf("GraphQL Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)
	gt.NotNil(t, resp.Data)

	// Verify empty response structure
	data := resp.Data.(map[string]interface{})
	incidents := data["incidents"].(map[string]interface{})
	edges := incidents["edges"].([]interface{})
	totalCount := incidents["totalCount"].(float64)

	gt.Equal(t, len(edges), 0)
	gt.Equal(t, totalCount, float64(0))
}

// Test single incident query
func TestGraphQL_SingleIncident(t *testing.T) {
	server, repo := setupGraphQLTestServer(t)
	defer server.Close()

	// Create test incident
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		incidentID,
		"Single Test Incident",
		"Single test incident description",
		"test_category",
		types.ChannelID("C1234567890"),
		types.ChannelName("origin-channel"),
		types.SlackUserID("U1234567890"),
	)
	gt.NoError(t, err)

	incident.ChannelID = types.ChannelID("C1234567890")

	// Save incident
	ctx := context.Background()
	gt.NoError(t, repo.PutIncident(ctx, incident))

	query := `
		query GetIncident($id: ID!) {
			incident(id: $id) {
				id
				title
				description
				channelName
				channelId
				createdBy
				tasks {
					id
					title
					status
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": incidentID.String(),
	}

	resp := executeGraphQL(t, server, query, variables)

	if len(resp.Errors) > 0 {
		t.Logf("GraphQL Errors: %+v", resp.Errors)
	}

	gt.Equal(t, len(resp.Errors), 0)
	gt.NotNil(t, resp.Data)

	// Verify the incident was found and returned
	data := resp.Data.(map[string]interface{})
	incidentData := data["incident"]
	if incidentData != nil {
		incidentObj := incidentData.(map[string]interface{})
		gt.Equal(t, incidentObj["title"], "Single Test Incident")
	}
}

// Test error handling for invalid queries
func TestGraphQL_InvalidQuery(t *testing.T) {
	server, _ := setupGraphQLTestServer(t)
	defer server.Close()

	query := `
		query {
			incidents(first: 10) {
				invalidField
			}
		}
	`

	resp := executeGraphQL(t, server, query, nil)

	// Should have errors for invalid field
	gt.A(t, resp.Errors).Longer(0)
}

// Test complete CRUD operations with GraphQL mutations
func TestGraphQL_CompleteCRUDOperations(t *testing.T) {
	server, repo := setupGraphQLTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Step 1: Create an incident first
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		incidentID,
		"CRUD Test Incident",
		"Test incident for CRUD operations",
		"test_category",
		types.ChannelID("C1234567890"),
		types.ChannelName("test-channel"),
		types.SlackUserID("U1234567890"),
	)
	gt.NoError(t, err)
	gt.NoError(t, repo.PutIncident(ctx, incident))

	// Step 2: Test CreateTask mutation
	createTaskMutation := `
		mutation CreateTask($input: CreateTaskInput!) {
			createTask(input: $input) {
				id
				title
				description
				status
				incidentId
			}
		}
	`

	createTaskVars := map[string]interface{}{
		"input": map[string]interface{}{
			"incidentId":  incidentID.String(),
			"title":       "Test Task",
			"description": "This is a test task",
		},
	}

	resp := executeGraphQL(t, server, createTaskMutation, createTaskVars)
	gt.Equal(t, len(resp.Errors), 0)
	gt.NotNil(t, resp.Data)

	// Extract task ID from response
	data := resp.Data.(map[string]interface{})
	createTaskData := data["createTask"].(map[string]interface{})
	taskID := createTaskData["id"].(string)
	gt.Equal(t, createTaskData["title"], "Test Task")
	gt.Equal(t, createTaskData["description"], "This is a test task")
	gt.Equal(t, createTaskData["incidentId"].(string), incidentID.String())

	// Step 3: Test UpdateTask mutation
	updateTaskMutation := `
		mutation UpdateTask($id: ID!, $input: UpdateTaskInput!) {
			updateTask(id: $id, input: $input) {
				id
				title
				description
				status
			}
		}
	`

	updateTaskVars := map[string]interface{}{
		"id": taskID,
		"input": map[string]interface{}{
			"title":       "Updated Test Task",
			"description": "This task has been updated",
			"status":      "completed",
		},
	}

	resp = executeGraphQL(t, server, updateTaskMutation, updateTaskVars)
	if len(resp.Errors) > 0 {
		t.Logf("Update Task Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	updateTaskData := data["updateTask"].(map[string]interface{})
	gt.Equal(t, updateTaskData["title"], "Updated Test Task")
	gt.Equal(t, updateTaskData["description"], "This task has been updated")
	gt.Equal(t, updateTaskData["status"], "completed")

	// Step 4: Test UpdateIncident mutation
	updateIncidentMutation := `
		mutation UpdateIncident($id: ID!, $input: UpdateIncidentInput!) {
			updateIncident(id: $id, input: $input) {
				id
				title
				description
			}
		}
	`

	updateIncidentVars := map[string]interface{}{
		"id": incidentID.String(),
		"input": map[string]interface{}{
			"title":       "Updated CRUD Test Incident",
			"description": "This incident has been updated",
		},
	}

	resp = executeGraphQL(t, server, updateIncidentMutation, updateIncidentVars)
	if len(resp.Errors) > 0 {
		t.Logf("Update Incident Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	updateIncidentData := data["updateIncident"].(map[string]interface{})
	gt.Equal(t, updateIncidentData["title"], "Updated CRUD Test Incident")
	gt.Equal(t, updateIncidentData["description"], "This incident has been updated")

	// Step 5: Test task query to verify updates
	taskQuery := `
		query GetTask($id: ID!) {
			task(id: $id) {
				id
				title
				description
				status
			}
		}
	`

	taskVars := map[string]interface{}{
		"id": taskID,
	}

	resp = executeGraphQL(t, server, taskQuery, taskVars)
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	taskData := data["task"].(map[string]interface{})
	gt.Equal(t, taskData["title"], "Updated Test Task")
	gt.Equal(t, taskData["status"], "completed")

	// Step 6: Test DeleteTask mutation
	deleteTaskMutation := `
		mutation DeleteTask($id: ID!) {
			deleteTask(id: $id)
		}
	`

	deleteTaskVars := map[string]interface{}{
		"id": taskID,
	}

	resp = executeGraphQL(t, server, deleteTaskMutation, deleteTaskVars)
	if len(resp.Errors) > 0 {
		t.Logf("Delete Task Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	deleted := data["deleteTask"].(bool)
	gt.True(t, deleted)

	// Step 7: Verify task is deleted by trying to fetch it
	resp = executeGraphQL(t, server, taskQuery, taskVars)
	// Should return null for deleted task
	data = resp.Data.(map[string]interface{})
	gt.Nil(t, data["task"])
}
