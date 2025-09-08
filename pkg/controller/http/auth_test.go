package http_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	controller "github.com/secmon-lab/lycaon/pkg/controller/http"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

func TestAuthHandlerLoginNotConfigured(t *testing.T) {
	// Setup with empty config
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)

	handler := controller.NewAuthHandler(ctx, slackConfig, authUC, "")

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.HandleLogin(w, req)

	// Assert
	gt.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthHandlerLoginConfigured(t *testing.T) {
	// Setup with OAuth config
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)

	handler := controller.NewAuthHandler(ctx, slackConfig, authUC, "")

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	// Execute
	handler.HandleLogin(w, req)

	// Assert - should redirect to Slack OAuth (OpenID Connect)
	gt.Equal(t, http.StatusTemporaryRedirect, w.Code)
	location := w.Header().Get("Location")
	gt.True(t, strings.Contains(location, "slack.com/openid/connect/authorize"))
	gt.True(t, strings.Contains(location, "client_id=test-client-id"))
}

func TestAuthHandlerLogout(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)

	// Create session first
	session, err := authUC.CreateSession(ctx, "U123", "Test User", "test@example.com")
	gt.NoError(t, err)

	handler := controller.NewAuthHandler(ctx, slackConfig, authUC, "")

	// Create request with session cookie
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: session.ID.String(),
	})
	w := httptest.NewRecorder()

	// Execute
	handler.HandleLogout(w, req)

	// Assert
	gt.Equal(t, http.StatusOK, w.Code)

	// Check that cookies are cleared
	cookies := w.Result().Cookies()
	var sessionIDCookie, sessionSecretCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_id" {
			sessionIDCookie = c
		}
		if c.Name == "session_secret" {
			sessionSecretCookie = c
		}
	}

	gt.True(t, sessionIDCookie != nil)
	gt.Equal(t, "", sessionIDCookie.Value)
	gt.Equal(t, -1, sessionIDCookie.MaxAge)

	gt.True(t, sessionSecretCookie != nil)
	gt.Equal(t, "", sessionSecretCookie.Value)
	gt.Equal(t, -1, sessionSecretCookie.MaxAge)
}

func TestAuthHandlerUserMe(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)

	// Create session
	session, err := authUC.CreateSession(ctx, "U123", "Test User", "test@example.com")
	gt.NoError(t, err)

	handler := controller.NewAuthHandler(ctx, slackConfig, authUC, "")

	t.Run("With valid session", func(t *testing.T) {
		// Create request with session cookie
		req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session_id",
			Value: session.ID.String(),
		})
		w := httptest.NewRecorder()

		// Execute
		handler.HandleUserMe(w, req)

		// Assert
		gt.Equal(t, http.StatusOK, w.Code)
		gt.True(t, strings.Contains(w.Body.String(), "U123"))
		gt.True(t, strings.Contains(w.Body.String(), "Test User"))
		gt.True(t, strings.Contains(w.Body.String(), "test@example.com"))
	})

	t.Run("Without session", func(t *testing.T) {
		// Create request without cookie
		req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.HandleUserMe(w, req)

		// Assert
		gt.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
