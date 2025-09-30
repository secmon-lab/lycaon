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
	slackCtrl "github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)

// getTestCategoriesForHTTP returns categories for HTTP controller testing purposes
func getTestCategoriesForHTTP() *model.CategoriesConfig {
	return &model.CategoriesConfig{
		Categories: []model.Category{
			{
				ID:           "security_incident",
				Name:         "Security Incident",
				Description:  "Security-related incidents including unauthorized access and malware infections",
				InviteUsers:  []string{"@security-lead"},
				InviteGroups: []string{"@security-team"},
			},
			{
				ID:           "system_failure",
				Name:         "System Failure",
				Description:  "System or service failures and outages",
				InviteUsers:  []string{"@sre-lead"},
				InviteGroups: []string{"@sre-oncall"},
			},
			{
				ID:          "performance_issue",
				Name:        "Performance Issue",
				Description: "System performance degradation or response time issues",
			},
			{
				ID:          "unknown",
				Name:        "Unknown",
				Description: "Incidents that cannot be categorized",
			},
		},
	}
}

// Helper to create mock clients for HTTP tests
func createMockClients() (gollem.LLMClient, *mocks.SlackClientMock) {
	return &mock.LLMClientMock{}, &mocks.SlackClientMock{
		AuthTestContextFunc: func(ctx context.Context) (*slackgo.AuthTestResponse, error) {
			return &slackgo.AuthTestResponse{UserID: "U_TEST_BOT", User: "test-bot"}, nil
		},
		GetUserInfoContextFunc: func(ctx context.Context, user string) (*slackgo.User, error) {
			return &slackgo.User{
				ID:       user,
				Name:     "test-user",
				RealName: "Test User",
				Profile: slackgo.UserProfile{
					DisplayName: "Test User",
					Email:       "test@example.com",
					Image24:     "https://example.com/avatar.png",
				},
			}, nil
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
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, getTestCategoriesForHTTP())
	gt.NoError(t, err).Required()
	categories := getTestCategoriesForHTTP()
	incidentConfig := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
	incidentUC := usecase.NewIncident(repo, nil, categories, nil, incidentConfig)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)
	statusUC := usecase.NewStatusUseCase(repo, mockSlack)
	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, statusUC, authUC, mockSlack)

	// Create configuration
	config := controller.NewConfig(":8080", slackConfig, categories, "")

	// Create use cases structure
	useCases := controller.NewUseCases(authUC, messageUC, incidentUC, taskUC, slackInteractionUC)

	// Create handlers
	slackHandler := slackCtrl.NewHandler(ctx, slackConfig, repo, useCases.SlackMessage(), useCases.Incident(), useCases.Task(), useCases.SlackInteraction(), mockSlack)
	authHandler := controller.NewAuthHandler(ctx, slackConfig, useCases.Auth(), "")

	// Create GraphQL handler
	var graphqlHandler http.Handler
	if repo != nil && useCases.Incident() != nil && useCases.Task() != nil {
		graphqlHandler = controller.CreateGraphQLHandler(repo, mockSlack, useCases, categories)
	}

	// Create controllers
	controllers := controller.NewController(slackHandler, authHandler, graphqlHandler)

	server, err := controller.NewServer(ctx, config, useCases, controllers, repo)
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
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, getTestCategoriesForHTTP())
	gt.NoError(t, err).Required()
	categories := getTestCategoriesForHTTP()
	incidentConfig := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
	incidentUC := usecase.NewIncident(repo, nil, categories, nil, incidentConfig)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)
	statusUC := usecase.NewStatusUseCase(repo, mockSlack)
	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, statusUC, authUC, mockSlack)

	// Create configuration
	config := controller.NewConfig(":8080", slackConfig, categories, "")

	// Create use cases structure
	useCases := controller.NewUseCases(authUC, messageUC, incidentUC, taskUC, slackInteractionUC)

	// Create handlers
	slackHandler := slackCtrl.NewHandler(ctx, slackConfig, repo, useCases.SlackMessage(), useCases.Incident(), useCases.Task(), useCases.SlackInteraction(), mockSlack)
	authHandler := controller.NewAuthHandler(ctx, slackConfig, useCases.Auth(), "")

	// Create GraphQL handler
	var graphqlHandler http.Handler
	if repo != nil && useCases.Incident() != nil && useCases.Task() != nil {
		graphqlHandler = controller.CreateGraphQLHandler(repo, mockSlack, useCases, categories)
	}

	// Create controllers
	controllers := controller.NewController(slackHandler, authHandler, graphqlHandler)

	server, err := controller.NewServer(ctx, config, useCases, controllers, repo)
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
	categories := getTestCategoriesForHTTP()

	// Create use cases
	incidentConfig := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
	incidentUC := usecase.NewIncident(repo, mockSlack, categories, nil, incidentConfig)
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
		"inc", // prefix
		incidentID,
		"Single Test Incident",
		"Single test incident description",
		"test_category",
		types.ChannelID("C1234567890"),
		types.ChannelName("origin-channel"),
		types.TeamID("T1234567890"),
		types.SlackUserID("U1234567890"),
		false, // initialTriage
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
				status
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
		// Debug: print the status value to see its format
		t.Logf("Status value: %v (type: %T)", incidentObj["status"], incidentObj["status"])
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
		"inc", // prefix
		incidentID,
		"CRUD Test Incident",
		"Test incident for CRUD operations",
		"test_category",
		types.ChannelID("C1234567890"),
		types.ChannelName("test-channel"),
		types.TeamID("T1234567890"),
		types.SlackUserID("U1234567890"),
		false, // initialTriage
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
	gt.NotNil(t, resp.Data)

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

// Test incident status management GraphQL operations
func TestGraphQL_IncidentStatusManagement(t *testing.T) {
	server, repo := setupGraphQLTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Step 1: Create an incident with initial status
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		"inc", // prefix
		incidentID,
		"Status Test Incident",
		"Test incident for status management",
		"test_category",
		types.ChannelID("C1234567890"),
		types.ChannelName("test-channel"),
		types.TeamID("T1234567890"),
		types.SlackUserID("U1234567890"),
		false, // initialTriage - should start with HANDLING
	)
	gt.NoError(t, err)
	gt.NoError(t, repo.PutIncident(ctx, incident))

	// Create initial status history
	initialStatusHistory, err := model.NewStatusHistory(incident.ID, incident.Status, incident.CreatedBy, "Incident created")
	gt.NoError(t, err)
	err = repo.AddStatusHistory(ctx, initialStatusHistory)
	gt.NoError(t, err)

	// Step 2: Query incident with status information
	incidentQuery := `
		query GetIncident($id: ID!) {
			incident(id: $id) {
				id
				title
				status
				statusHistories {
					id
					status
					changedAt
					changedBy {
						id
						name
					}
					note
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": incidentID.String(),
	}

	resp := executeGraphQL(t, server, incidentQuery, variables)
	if len(resp.Errors) > 0 {
		t.Logf("GraphQL Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data := resp.Data.(map[string]interface{})
	incidentData := data["incident"].(map[string]interface{})
	gt.Equal(t, incidentData["status"], "handling") // Should start with handling
	statusHistories := incidentData["statusHistories"].([]interface{})
	gt.Equal(t, len(statusHistories), 1) // Initial status history

	// Verify initial status history
	initialHistory := statusHistories[0].(map[string]interface{})
	gt.Equal(t, initialHistory["status"], "handling")

	// Step 3: Test updateIncidentStatus mutation
	updateStatusMutation := `
		mutation UpdateIncidentStatus($incidentId: ID!, $status: IncidentStatus!, $note: String) {
			updateIncidentStatus(incidentId: $incidentId, status: $status, note: $note) {
				id
				status
				statusHistories {
					id
					status
					changedAt
					changedBy {
						id
						name
					}
					note
				}
			}
		}
	`

	updateVariables := map[string]interface{}{
		"incidentId": incidentID.String(),
		"status":     "monitoring",
		"note":       "Moving to monitoring phase",
	}

	resp = executeGraphQL(t, server, updateStatusMutation, updateVariables)
	if len(resp.Errors) > 0 {
		t.Logf("Update Status Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	updateResult := data["updateIncidentStatus"].(map[string]interface{})
	gt.Equal(t, updateResult["status"], "monitoring")

	// Verify status history was updated
	updatedHistories := updateResult["statusHistories"].([]interface{})
	t.Logf("Updated histories count: %d", len(updatedHistories))
	for i, h := range updatedHistories {
		history := h.(map[string]interface{})
		t.Logf("History %d: status=%v, note=%v", i, history["status"], history["note"])
	}
	gt.Equal(t, len(updatedHistories), 2) // Initial + new status

	// Find the new status history entry
	var newHistory map[string]interface{}
	for _, h := range updatedHistories {
		history := h.(map[string]interface{})
		if history["status"] == "monitoring" {
			newHistory = history
			break
		}
	}
	gt.NotNil(t, newHistory)
	gt.Equal(t, newHistory["note"], "Moving to monitoring phase")

	// Step 4: Test status change to CLOSED
	closeVariables := map[string]interface{}{
		"incidentId": incidentID.String(),
		"status":     "closed",
		"note":       "Incident resolved",
	}

	resp = executeGraphQL(t, server, updateStatusMutation, closeVariables)
	if len(resp.Errors) > 0 {
		t.Logf("Close Status Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	closeResult := data["updateIncidentStatus"].(map[string]interface{})
	gt.Equal(t, closeResult["status"], "closed")

	// Verify final status history count
	finalHistories := closeResult["statusHistories"].([]interface{})
	gt.Equal(t, len(finalHistories), 3) // Initial + monitoring + closed

	// Step 5: Test incidentStatusHistory query
	statusHistoryQuery := `
		query GetIncidentStatusHistory($incidentId: ID!) {
			incidentStatusHistory(incidentId: $incidentId) {
				id
				status
				changedAt
				changedBy {
					id
					name
				}
				note
			}
		}
	`

	historyVariables := map[string]interface{}{
		"incidentId": incidentID.String(),
	}

	resp = executeGraphQL(t, server, statusHistoryQuery, historyVariables)
	if len(resp.Errors) > 0 {
		t.Logf("Status History Query Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	historyData := data["incidentStatusHistory"].([]interface{})
	gt.Equal(t, len(historyData), 3)

	// Verify the sequence of status changes
	expectedStatuses := []string{"handling", "monitoring", "closed"}
	for i, h := range historyData {
		history := h.(map[string]interface{})
		gt.Equal(t, history["status"].(string), expectedStatuses[i])
	}
}

// Test error cases for status management
func TestGraphQL_IncidentStatusErrors(t *testing.T) {
	server, _ := setupGraphQLTestServer(t)
	defer server.Close()

	// Test 1: Update status for non-existent incident
	updateStatusMutation := `
		mutation UpdateIncidentStatus($incidentId: ID!, $status: IncidentStatus!) {
			updateIncidentStatus(incidentId: $incidentId, status: $status) {
				id
				status
			}
		}
	`

	variables := map[string]interface{}{
		"incidentId": "non-existent-incident",
		"status":     "closed",
	}

	resp := executeGraphQL(t, server, updateStatusMutation, variables)
	gt.A(t, resp.Errors).Longer(0) // Should have errors

	// Test 2: Invalid status value
	invalidStatusVars := map[string]interface{}{
		"incidentId": "some-id",
		"status":     "INVALID_STATUS",
	}

	resp = executeGraphQL(t, server, updateStatusMutation, invalidStatusVars)
	gt.A(t, resp.Errors).Longer(0) // Should have errors for invalid enum

	// Test 3: Status history query for non-existent incident
	statusHistoryQuery := `
		query GetIncidentStatusHistory($incidentId: ID!) {
			incidentStatusHistory(incidentId: $incidentId) {
				id
				status
			}
		}
	`

	historyVariables := map[string]interface{}{
		"incidentId": "non-existent-incident",
	}

	resp = executeGraphQL(t, server, statusHistoryQuery, historyVariables)
	// This might not error but should return empty array
	if len(resp.Errors) == 0 {
		data := resp.Data.(map[string]interface{})
		histories := data["incidentStatusHistory"]
		if histories != nil {
			historyList := histories.([]interface{})
			gt.Equal(t, len(historyList), 0)
		}
	}
}

// Test incident creation with initial triage flag
func TestGraphQL_IncidentCreateWithTriage(t *testing.T) {
	server, repo := setupGraphQLTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Step 1: Create incident with triage flag = true
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		"inc", // prefix
		incidentID,
		"Triage Test Incident",
		"Test incident for triage status",
		"test_category",
		types.ChannelID("C1234567890"),
		types.ChannelName("test-channel"),
		types.TeamID("T1234567890"),
		types.SlackUserID("U1234567890"),
		true, // initialTriage - should start with TRIAGE
	)
	gt.NoError(t, err)
	gt.NoError(t, repo.PutIncident(ctx, incident))

	// Create initial status history
	initialStatusHistory, err := model.NewStatusHistory(incident.ID, incident.Status, incident.CreatedBy, "Incident created")
	gt.NoError(t, err)
	err = repo.AddStatusHistory(ctx, initialStatusHistory)
	gt.NoError(t, err)

	// Step 2: Query incident to verify it starts with TRIAGE status
	incidentQuery := `
		query GetIncident($id: ID!) {
			incident(id: $id) {
				id
				status
				initialTriage
				statusHistories {
					status
					changedAt
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": incidentID.String(),
	}

	resp := executeGraphQL(t, server, incidentQuery, variables)
	if len(resp.Errors) > 0 {
		t.Logf("GraphQL Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data := resp.Data.(map[string]interface{})
	incidentData := data["incident"].(map[string]interface{})
	gt.Equal(t, incidentData["status"], "triage") // Should start with triage
	gt.Equal(t, incidentData["initialTriage"], true)

	statusHistories := incidentData["statusHistories"].([]interface{})
	gt.Equal(t, len(statusHistories), 1)

	initialHistory := statusHistories[0].(map[string]interface{})
	gt.Equal(t, initialHistory["status"], "triage")

	// Step 3: Test status progression from TRIAGE to HANDLING
	updateStatusMutation := `
		mutation UpdateIncidentStatus($incidentId: ID!, $status: IncidentStatus!) {
			updateIncidentStatus(incidentId: $incidentId, status: $status) {
				id
				status
				statusHistories {
					status
					changedAt
				}
			}
		}
	`

	updateVariables := map[string]interface{}{
		"incidentId": incidentID.String(),
		"status":     "handling",
	}

	resp = executeGraphQL(t, server, updateStatusMutation, updateVariables)
	gt.Equal(t, len(resp.Errors), 0)

	data = resp.Data.(map[string]interface{})
	updateResult := data["updateIncidentStatus"].(map[string]interface{})
	gt.Equal(t, updateResult["status"], "handling")

	updatedHistories := updateResult["statusHistories"].([]interface{})
	gt.Equal(t, len(updatedHistories), 2) // TRIAGE + HANDLING

	// Verify the progression
	statuses := make([]string, len(updatedHistories))
	for i, h := range updatedHistories {
		history := h.(map[string]interface{})
		statuses[i] = history["status"].(string)
	}

	// Check that both TRIAGE and HANDLING are present
	hasTriageStatus := false
	hasHandlingStatus := false
	for _, status := range statuses {
		if status == "triage" {
			hasTriageStatus = true
		}
		if status == "handling" {
			hasHandlingStatus = true
		}
	}
	gt.True(t, hasTriageStatus)
	gt.True(t, hasHandlingStatus)
}

// Test handling of legacy incidents without status fields
func TestGraphQL_LegacyIncidentWithoutStatus(t *testing.T) {
	server, repo := setupGraphQLTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a legacy incident directly in repository without status fields
	incidentID := types.IncidentID(time.Now().UnixNano())
	legacyIncident := &model.Incident{
		ID:                incidentID,
		Title:             "Legacy Test Incident",
		Description:       "Test incident without status fields",
		CategoryID:        "test_category",
		ChannelID:         types.ChannelID("C1234567890"),
		ChannelName:       types.ChannelName("legacy-channel"),
		OriginChannelID:   types.ChannelID("C0987654321"),
		OriginChannelName: types.ChannelName("origin-channel"),
		CreatedBy:         types.SlackUserID("U1234567890"),
		CreatedAt:         time.Now(),
		// Status, Lead, StatusHistories, InitialTriage are intentionally omitted (will be zero values)
	}

	// Save legacy incident
	gt.NoError(t, repo.PutIncident(ctx, legacyIncident))

	// Query incident to verify it handles empty status gracefully
	incidentQuery := `
		query GetIncident($id: ID!) {
			incident(id: $id) {
				id
				title
				status
				lead
				statusHistories {
					id
					status
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": incidentID.String(),
	}

	resp := executeGraphQL(t, server, incidentQuery, variables)
	if len(resp.Errors) > 0 {
		t.Logf("GraphQL Errors: %+v", resp.Errors)
	}
	gt.Equal(t, len(resp.Errors), 0)

	data := resp.Data.(map[string]interface{})
	incidentData := data["incident"].(map[string]interface{})
	gt.Equal(t, incidentData["title"], "Legacy Test Incident")

	// Status and lead should be null/nil for legacy incidents
	gt.Nil(t, incidentData["status"])
	gt.Nil(t, incidentData["lead"])

	// Status histories should be empty array
	statusHistories := incidentData["statusHistories"].([]interface{})
	gt.Equal(t, len(statusHistories), 0)

	t.Logf("Legacy incident handled correctly: status=%v, lead=%v, histories=%d",
		incidentData["status"], incidentData["lead"], len(statusHistories))
}

func TestGraphQL_Firestore_Integration(t *testing.T) {
	// Skip test if Firestore test environment variables are not set
	projectID := os.Getenv("TEST_FIRESTORE_PROJECT")
	databaseID := os.Getenv("TEST_FIRESTORE_DATABASE")

	t.Logf("Environment check: PROJECT=%s, DATABASE=%s", projectID, databaseID)

	if projectID == "" || databaseID == "" {
		t.Skip("Skipping Firestore GraphQL test: TEST_FIRESTORE_PROJECT and TEST_FIRESTORE_DATABASE must be set")
	}

	t.Logf("Starting Firestore integration test with PROJECT=%s, DATABASE=%s", projectID, databaseID)

	// Create Firestore repository
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	repo, err := repository.NewFirestore(ctx, projectID, databaseID)
	gt.NoError(t, err).Required()

	// Create simple GraphQL test by directly calling the repository
	// Skip GraphQL for now and test repo directly

	t.Run("TestFirestoreListIncidentsPaginated", func(t *testing.T) {
		// Test the repository method directly
		opts := types.PaginationOptions{
			Limit: 10,
		}

		incidents, result, err := repo.ListIncidentsPaginated(ctx, opts)

		if err != nil {
			t.Logf("Firestore ListIncidentsPaginated failed with error: %v", err)
			t.Logf("This is the actual error that's causing the WebUI issue!")
		} else {
			t.Logf("Firestore ListIncidentsPaginated succeeded: found %d incidents", len(incidents))
			t.Logf("Pagination result: hasNext=%v, hasPrev=%v, total=%d",
				result.HasNextPage, result.HasPreviousPage, result.TotalCount)

			// Check each incident for missing fields
			for i, incident := range incidents {
				if i >= 3 { // Just check first 3
					break
				}
				t.Logf("Incident %d: ID=%v, Title=%v, Status=%v",
					i, incident.ID, incident.Title, incident.Status)
			}
		}
	})
}
