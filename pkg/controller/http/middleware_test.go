package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	controller "github.com/secmon-lab/lycaon/pkg/controller/http"
	slackCtrl "github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	slackservice "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

// TestHTTPAccessControlPrivateIncidents tests that the HTTP layer properly enforces
// access control for private incidents through GraphQL queries
func TestHTTPAccessControlPrivateIncidents(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)
	mockLLM, mockSlack := createMockClients()
	slackSvc := slackservice.NewUIService(mockSlack, testConfig())
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, slackSvc, testConfig())
	gt.NoError(t, err).Required()
	incidentConfig := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
	incidentUC := usecase.NewIncident(repo, nil, slackSvc, testConfig(), nil, incidentConfig)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)
	statusUC := usecase.NewStatusUseCase(repo, slackSvc, testConfig())
	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, statusUC, authUC, mockSlack, slackSvc, nil)

	// Create configuration
	httpConfig := controller.NewConfig(":8080", slackConfig, testConfig(), "")

	// Create use cases structure
	useCases := controller.NewUseCases(authUC, messageUC, incidentUC, taskUC, slackInteractionUC)

	// Create handlers
	slackHandler := slackCtrl.NewHandler(ctx, slackConfig, repo, useCases.SlackMessage(), useCases.Incident(), useCases.Task(), useCases.SlackInteraction(), mockSlack, testConfig())
	authHandler := controller.NewAuthHandler(ctx, slackConfig, useCases.Auth(), "")

	// Create GraphQL handler
	graphqlHandler := controller.CreateGraphQLHandler(repo, mockSlack, useCases, testConfig())

	// Create controllers
	controllers := controller.NewController(slackHandler, authHandler, graphqlHandler)

	server, err := controller.NewServer(ctx, httpConfig, useCases, controllers, repo)
	gt.NoError(t, err).Required()

	// Create test sessions for different users with random IDs to avoid test conflicts
	now := time.Now()
	nanoSuffix := now.UnixNano()

	nonMemberUserID := types.UserID("U-NON-MEMBER")
	nonMemberSessionID := types.SessionID(fmt.Sprintf("session-non-member-%d", nanoSuffix))
	nonMemberSession := &model.Session{
		ID:        nonMemberSessionID,
		Secret:    types.SessionSecret(fmt.Sprintf("secret-non-member-%d", nanoSuffix)),
		UserID:    nonMemberUserID, // UserID is SlackUserID
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	gt.NoError(t, repo.SaveSession(ctx, nonMemberSession))

	memberUserID := types.UserID("U-MEMBER1")
	memberSessionID := types.SessionID(fmt.Sprintf("session-member-%d", nanoSuffix))
	memberSession := &model.Session{
		ID:        memberSessionID,
		Secret:    types.SessionSecret(fmt.Sprintf("secret-member-%d", nanoSuffix)),
		UserID:    memberUserID, // UserID is SlackUserID
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	gt.NoError(t, repo.SaveSession(ctx, memberSession))

	randomUserID := types.UserID("U-RANDOM-USER")
	randomSessionID := types.SessionID(fmt.Sprintf("session-random-%d", nanoSuffix))
	randomUserSession := &model.Session{
		ID:        randomSessionID,
		Secret:    types.SessionSecret(fmt.Sprintf("secret-random-%d", nanoSuffix)),
		UserID:    randomUserID, // UserID is SlackUserID
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	gt.NoError(t, repo.SaveSession(ctx, randomUserSession))

	// Create test incidents with random IDs to avoid test conflicts
	publicIncidentID := types.IncidentID(nanoSuffix)
	publicIncident := &model.Incident{
		ID:          publicIncidentID,
		Title:       "Public Incident",
		Description: "This is a public incident with full details",
		CategoryID:  "security_incident",
		SeverityID:  types.SeverityID("high"),
		ChannelID:   types.ChannelID(fmt.Sprintf("C-PUBLIC-%d", nanoSuffix)),
		ChannelName: types.ChannelName("public-channel"),
		Status:      types.IncidentStatusTriage,
		Private:     false,
		CreatedBy:   types.SlackUserID("U-CREATOR"),
	}
	gt.NoError(t, repo.PutIncident(ctx, publicIncident))

	privateIncidentID := types.IncidentID(nanoSuffix + 1)
	privateIncident := &model.Incident{
		ID:              privateIncidentID,
		Title:           "Private Incident Real Title",
		Description:     "This is private and contains sensitive information",
		CategoryID:      "security_incident",
		SeverityID:      types.SeverityID("critical"),
		ChannelID:       types.ChannelID(fmt.Sprintf("C-PRIVATE-%d", nanoSuffix)),
		ChannelName:     types.ChannelName("private-channel"),
		Status:          types.IncidentStatusHandling,
		Private:         true,
		JoinedMemberIDs: []types.SlackUserID{types.SlackUserID(memberUserID), "U-MEMBER2"},
		CreatedBy:       types.SlackUserID("U-CREATOR"),
	}
	gt.NoError(t, repo.PutIncident(ctx, privateIncident))

	t.Run("Non-member cannot access private incident details via GraphQL", func(t *testing.T) {
		// Create GraphQL query for specific incident
		query := `
		query GetIncident($id: ID!) {
			incident(id: $id) {
				id
				title
				description
				status
				categoryName
			}
		}
		`
		variables := map[string]interface{}{
			"id": fmt.Sprintf("%d", privateIncidentID),
		}

		requestBody := map[string]interface{}{
			"query":     query,
			"variables": variables,
		}
		bodyJSON, _ := json.Marshal(requestBody)

		// Create authenticated request as non-member
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(bodyJSON)))
		req.Header.Set("Content-Type", "application/json")

		// Add session cookies for authentication
		req.AddCookie(&http.Cookie{Name: "session_id", Value: string(nonMemberSessionID)})
		req.AddCookie(&http.Cookie{Name: "session_secret", Value: string(nonMemberSession.Secret)})

		w := httptest.NewRecorder()

		// Execute
		server.Server.Handler.ServeHTTP(w, req)

		// Assert
		gt.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		gt.NoError(t, err)

		// Check that incident data is filtered
		data := response["data"].(map[string]interface{})
		incident := data["incident"].(map[string]interface{})

		// Title should be filtered to "Private Incident"
		gt.Equal(t, "Private Incident", incident["title"])
		// Description should be empty
		gt.Equal(t, "", incident["description"])
		// Status should still be visible (part of allowed fields)
		gt.Equal(t, "handling", incident["status"])
	})

	t.Run("Member can access full private incident details via GraphQL", func(t *testing.T) {
		// Create GraphQL query for specific incident
		query := `
		query GetIncident($id: ID!) {
			incident(id: $id) {
				id
				title
				description
				status
				categoryName
			}
		}
		`
		variables := map[string]interface{}{
			"id": fmt.Sprintf("%d", privateIncidentID),
		}

		requestBody := map[string]interface{}{
			"query":     query,
			"variables": variables,
		}
		bodyJSON, _ := json.Marshal(requestBody)

		// Create authenticated request as member
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(bodyJSON)))
		req.Header.Set("Content-Type", "application/json")

		// Add session cookies for authentication
		req.AddCookie(&http.Cookie{Name: "session_id", Value: string(memberSessionID)})
		req.AddCookie(&http.Cookie{Name: "session_secret", Value: string(memberSession.Secret)})

		w := httptest.NewRecorder()

		// Execute
		server.Server.Handler.ServeHTTP(w, req)

		// Assert
		gt.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		gt.NoError(t, err)

		// Check that incident data is NOT filtered
		data := response["data"].(map[string]interface{})
		incident := data["incident"].(map[string]interface{})

		// Full details should be visible
		gt.Equal(t, "Private Incident Real Title", incident["title"])
		gt.Equal(t, "This is private and contains sensitive information", incident["description"])
		gt.Equal(t, "handling", incident["status"])
	})

	t.Run("Public incident is accessible to all users via GraphQL", func(t *testing.T) {
		// Create GraphQL query for public incident
		query := `
		query GetIncident($id: ID!) {
			incident(id: $id) {
				id
				title
				description
				status
			}
		}
		`
		variables := map[string]interface{}{
			"id": fmt.Sprintf("%d", publicIncidentID),
		}

		requestBody := map[string]interface{}{
			"query":     query,
			"variables": variables,
		}
		bodyJSON, _ := json.Marshal(requestBody)

		// Create authenticated request as any user
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(bodyJSON)))
		req.Header.Set("Content-Type", "application/json")

		// Add session cookies for authentication
		req.AddCookie(&http.Cookie{Name: "session_id", Value: string(randomSessionID)})
		req.AddCookie(&http.Cookie{Name: "session_secret", Value: string(randomUserSession.Secret)})

		w := httptest.NewRecorder()

		// Execute
		server.Server.Handler.ServeHTTP(w, req)

		// Assert
		gt.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		gt.NoError(t, err)

		// Check that public incident is fully visible
		data := response["data"].(map[string]interface{})
		incident := data["incident"].(map[string]interface{})

		gt.Equal(t, "Public Incident", incident["title"])
		gt.Equal(t, "This is a public incident with full details", incident["description"])
	})

	t.Run("Incidents list filters private incidents for non-members via GraphQL", func(t *testing.T) {
		// Create GraphQL query for incidents list
		query := `
		query GetIncidents($first: Int) {
			incidents(first: $first) {
				edges {
					node {
						id
						title
						description
						private
					}
				}
			}
		}
		`
		variables := map[string]interface{}{
			"first": 10,
		}

		requestBody := map[string]interface{}{
			"query":     query,
			"variables": variables,
		}
		bodyJSON, _ := json.Marshal(requestBody)

		// Create authenticated request as non-member
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(bodyJSON)))
		req.Header.Set("Content-Type", "application/json")

		// Add session cookies for authentication
		req.AddCookie(&http.Cookie{Name: "session_id", Value: string(nonMemberSessionID)})
		req.AddCookie(&http.Cookie{Name: "session_secret", Value: string(nonMemberSession.Secret)})

		w := httptest.NewRecorder()

		// Execute
		server.Server.Handler.ServeHTTP(w, req)

		// Assert
		gt.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		gt.NoError(t, err)

		// Check incidents list
		data := response["data"].(map[string]interface{})
		incidents := data["incidents"].(map[string]interface{})
		edges := incidents["edges"].([]interface{})

		gt.Equal(t, 2, len(edges))

		// Find incidents by ID
		var publicInc, privateInc map[string]interface{}
		publicIDStr := fmt.Sprintf("%d", publicIncidentID)
		privateIDStr := fmt.Sprintf("%d", privateIncidentID)
		for _, edge := range edges {
			node := edge.(map[string]interface{})["node"].(map[string]interface{})
			if node["id"] == publicIDStr {
				publicInc = node
			} else if node["id"] == privateIDStr {
				privateInc = node
			}
		}

		// Public incident should be fully visible
		gt.V(t, publicInc).NotNil()
		gt.Equal(t, "Public Incident", publicInc["title"])
		gt.Equal(t, "This is a public incident with full details", publicInc["description"])

		// Private incident should be filtered
		gt.V(t, privateInc).NotNil()
		gt.Equal(t, "Private Incident", privateInc["title"])
		gt.Equal(t, "", privateInc["description"])
		gt.Equal(t, true, privateInc["private"])
	})
}
