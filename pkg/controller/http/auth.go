package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	slackConfig *config.SlackConfig
	authUC      interfaces.Auth
	frontendURL string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(ctx context.Context, slackConfig *config.SlackConfig, authUC interfaces.Auth, frontendURL string) *AuthHandler {
	return &AuthHandler{
		slackConfig: slackConfig,
		authUC:      authUC,
		frontendURL: frontendURL,
	}
}

// generateRandomState generates a secure random state parameter
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Use RawURLEncoding to avoid padding characters (=) that can cause issues
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HandleLogin initiates the OAuth flow
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	logger := ctxlog.From(r.Context())

	// Generate state parameter for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		logger.Error("Failed to generate state", "error", err)
		writeError(w, goerr.Wrap(err, "failed to generate state"), http.StatusInternalServerError)
		return
	}

	// Generate OAuth URL using usecase
	oauthConfig := interfaces.OAuthConfig{
		ClientID:     h.slackConfig.ClientID,
		ClientSecret: h.slackConfig.ClientSecret,
		RedirectURI:  h.getRedirectURI(r),
		State:        state,
	}

	oauthURL, err := h.authUC.GenerateOAuthURL(r.Context(), oauthConfig)
	if err != nil {
		logger.Error("Failed to generate OAuth URL", "error", err)
		writeError(w, goerr.Wrap(err, "failed to generate OAuth URL"), http.StatusInternalServerError)
		return
	}

	// Store state in cookie for verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	logger.Info("Redirecting to Slack OAuth",
		"state", state,
		"redirectURI", h.getRedirectURI(r),
		"teamID", oauthURL.TeamID,
		"fullURL", oauthURL.URL,
	)

	// Redirect to Slack OAuth
	http.Redirect(w, r, oauthURL.URL, http.StatusTemporaryRedirect)
}

// HandleCallback handles the OAuth callback
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	logger := ctxlog.From(r.Context())

	// Try to extract query params from headers if direct query is empty
	if r.URL.RawQuery == "" {
		// Check X-Original-URL header (some proxies use this)
		if origURL := r.Header.Get("X-Original-URL"); origURL != "" {
			logger.Info("Attempting to parse X-Original-URL", "origURL", origURL)
			if u, err := url.Parse(origURL); err == nil && u.RawQuery != "" {
				r.URL.RawQuery = u.RawQuery
				logger.Info("Restored query from X-Original-URL", "query", u.RawQuery)
			}
		}

		// Check X-Forwarded-Uri header
		if fwdURI := r.Header.Get("X-Forwarded-Uri"); fwdURI != "" {
			logger.Info("Attempting to parse X-Forwarded-Uri", "fwdURI", fwdURI)
			if u, err := url.Parse(fwdURI); err == nil && u.RawQuery != "" {
				r.URL.RawQuery = u.RawQuery
				logger.Info("Restored query from X-Forwarded-Uri", "query", u.RawQuery)
			}
		}
	}

	if !h.slackConfig.IsOAuthConfigured() {
		writeError(w, goerr.New("Slack OAuth not configured"), http.StatusServiceUnavailable)
		return
	}

	// Get and verify state parameter for CSRF protection
	state := r.URL.Query().Get("state")
	storedStateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		logger.Error("OAuth state cookie not found", "error", err)
		writeError(w, goerr.New("OAuth state not found"), http.StatusBadRequest)
		return
	}

	if state == "" || state != storedStateCookie.Value {
		logger.Error("OAuth state mismatch",
			"receivedState", state,
			"storedState", storedStateCookie.Value,
		)
		writeError(w, goerr.New("Invalid OAuth state"), http.StatusBadRequest)
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Get the authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		logger.Error("Authorization code not found in callback",
			"query", r.URL.RawQuery,
			"url", r.URL.String(),
		)
		writeError(w, goerr.New("authorization code not provided"), http.StatusBadRequest)
		return
	}

	// Exchange code for token using OpenID Connect
	logger.Info("Exchanging OAuth code for token",
		"code", code,
		"redirectURI", h.getRedirectURI(r),
		"clientID", h.slackConfig.ClientID,
	)

	user, err := h.authUC.HandleCallback(r.Context(), code, h.getRedirectURI(r))
	if err != nil {
		logger.Error("Failed to handle OAuth callback",
			"error", err,
			"code", code,
			"redirectURI", h.getRedirectURI(r),
		)
		writeError(w, goerr.Wrap(err, "failed to handle OAuth callback"), http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := h.authUC.CreateSession(
		r.Context(),
		user.SlackUserID,
		user.Name,
		user.Email,
	)
	if err != nil {
		logger.Error("Failed to create session", "error", err)
		writeError(w, goerr.Wrap(err, "failed to create session"), http.StatusInternalServerError)
		return
	}

	// Set session cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "session_secret",
		Value:    session.Secret,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	logger.Info("User authenticated successfully",
		"userID", user.SlackUserID,
		"userName", user.Name,
	)

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// HandleLogout handles logout
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from cookie
	sessionIDCookie, err := r.Cookie("session_id")
	if err == nil {
		// Try to delete the session
		if err := h.authUC.DeleteSession(r.Context(), sessionIDCookie.Value); err != nil {
			logger := ctxlog.From(r.Context())
			logger.Debug("Failed to delete session", "error", err)
		}
	}

	// Clear cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "session_secret",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "logged out successfully",
	})
}

// HandleUserMe returns current user information
func (h *AuthHandler) HandleUserMe(w http.ResponseWriter, r *http.Request) {
	// Get session ID from cookie
	sessionIDCookie, err := r.Cookie("session_id")
	if err != nil {
		writeError(w, goerr.New("session not found"), http.StatusUnauthorized)
		return
	}

	// Get user from session
	user, err := h.authUC.GetUserFromSession(r.Context(), sessionIDCookie.Value)
	if err != nil {
		writeError(w, goerr.Wrap(err, "failed to get user"), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// getRedirectURI constructs the redirect URI
func (h *AuthHandler) getRedirectURI(r *http.Request) string {
	baseURL := GetFrontendURL(r, h.frontendURL)
	return baseURL + "/api/auth/callback"
}
