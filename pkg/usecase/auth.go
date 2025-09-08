package usecase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/slack-go/slack"
)

// Auth implements Auth interface with repository-based storage
type Auth struct {
	repo        interfaces.Repository
	slackConfig *config.SlackConfig
}

// NewAuth creates a new Auth use case
func NewAuth(ctx context.Context, repo interfaces.Repository, slackConfig *config.SlackConfig) *Auth {
	return &Auth{
		repo:        repo,
		slackConfig: slackConfig,
	}
}

// GenerateOAuthURL generates Slack OAuth URL with team ID from API
func (a *Auth) GenerateOAuthURL(ctx context.Context, config interfaces.OAuthConfig) (*interfaces.OAuthURL, error) {
	logger := ctxlog.From(ctx)

	if !a.slackConfig.IsOAuthConfigured() {
		return nil, goerr.New("Slack OAuth not configured")
	}

	// Get team ID from Slack API if OAuth token is available
	var teamID string
	if a.slackConfig.OAuthToken != "" {
		client := slack.New(a.slackConfig.OAuthToken)
		authTest, err := client.AuthTestContext(ctx)
		if err != nil {
			logger.Warn("Failed to get team ID from Slack API", "error", err)
		} else {
			teamID = authTest.TeamID
			logger.Info("Retrieved team ID from Slack API",
				"teamID", teamID,
				"team", authTest.Team,
				"userID", authTest.UserID,
			)
		}
	} else {
		logger.Debug("No OAuth token configured, skipping team ID retrieval")
	}

	// Generate OAuth URL using OpenID Connect endpoint (same as warren)
	oauthURL := url.URL{
		Scheme: "https",
		Host:   "slack.com",
		Path:   "/openid/connect/authorize",
	}

	q := oauthURL.Query()
	q.Set("client_id", a.slackConfig.ClientID)
	// Use OpenID Connect scopes for user authentication only
	q.Set("scope", "openid,email,profile")
	q.Set("redirect_uri", config.RedirectURI)
	q.Set("response_type", "code")
	q.Set("state", config.State)

	// Add team parameter if we have team ID
	if teamID != "" {
		q.Set("team", teamID)
		logger.Info("Adding team parameter to OAuth URL", "teamID", teamID)
	} else {
		logger.Info("No team ID available, OAuth URL will not include team parameter")
	}

	oauthURL.RawQuery = q.Encode()

	logger.Info("Generated OAuth URL",
		"url", oauthURL.String(),
		"teamID", teamID,
		"hasTeam", teamID != "",
	)

	return &interfaces.OAuthURL{
		URL:    oauthURL.String(),
		State:  config.State,
		TeamID: teamID,
	}, nil
}

// CreateSession creates a new session for a user
func (a *Auth) CreateSession(ctx context.Context, slackUserID, userName, userEmail string) (*model.Session, error) {
	logger := ctxlog.From(ctx)

	if slackUserID == "" {
		return nil, goerr.New("slack user ID is required")
	}

	// Find or create user
	user, err := a.repo.GetUserBySlackID(ctx, slackUserID)
	if err != nil {
		// User doesn't exist, create new one
		user = model.NewUser(slackUserID, userName, userEmail)
		user.ID = slackUserID // Use Slack ID directly as user ID

		if err := a.repo.SaveUser(ctx, user); err != nil {
			return nil, goerr.Wrap(err, "failed to save user")
		}

		logger.Info("Created new user",
			"userID", user.ID,
			"slackUserID", slackUserID,
			"userName", userName,
		)
	}

	// Create new session (24 hours validity)
	session, err := model.NewSession(user.ID, 24*time.Hour)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create session")
	}

	// Store session
	if err := a.repo.SaveSession(ctx, session); err != nil {
		return nil, goerr.Wrap(err, "failed to save session")
	}

	logger.Info("Created new session",
		"sessionID", session.ID,
		"userID", user.ID,
		"expiresAt", session.ExpiresAt,
	)

	return session, nil
}

// ValidateSession validates a session by ID and secret
func (a *Auth) ValidateSession(ctx context.Context, sessionID, sessionSecret string) (*model.Session, error) {
	if sessionID == "" || sessionSecret == "" {
		return nil, goerr.New("session ID and secret are required")
	}

	session, err := a.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, goerr.Wrap(err, "session not found")
	}

	// Validate secret
	if session.Secret != sessionSecret {
		return nil, goerr.New("invalid session secret")
	}

	// Check expiration
	if session.IsExpired() {
		return nil, goerr.New("session expired")
	}

	return session, nil
}

// DeleteSession deletes a session
func (a *Auth) DeleteSession(ctx context.Context, sessionID string) error {
	logger := ctxlog.From(ctx)

	if sessionID == "" {
		return goerr.New("session ID is required")
	}

	if err := a.repo.DeleteSession(ctx, sessionID); err != nil {
		return goerr.Wrap(err, "failed to delete session")
	}

	logger.Info("Deleted session",
		"sessionID", sessionID,
	)

	return nil
}

// GetUserFromSession gets user information from a session
func (a *Auth) GetUserFromSession(ctx context.Context, sessionID string) (*model.User, error) {
	if sessionID == "" {
		return nil, goerr.New("session ID is required")
	}

	session, err := a.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, goerr.Wrap(err, "session not found")
	}

	if session.IsExpired() {
		return nil, goerr.New("session expired")
	}

	user, err := a.repo.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, goerr.Wrap(err, "user not found")
	}

	return user, nil
}

// SlackTokenResponse represents the response from Slack token exchange
type SlackTokenResponse struct {
	OK          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	BotUserID   string `json:"bot_user_id"`
	AppID       string `json:"app_id"`
	Team        struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"team"`
	Enterprise interface{} `json:"enterprise"`
	AuthedUser struct {
		ID          string `json:"id"`
		Scope       string `json:"scope"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	} `json:"authed_user"`
	IDToken string `json:"id_token"`
	Error   string `json:"error"`
}

// SlackIDToken represents the decoded ID token from Slack
type SlackIDToken struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// OpenIDConfiguration represents Slack's OpenID Connect configuration
type OpenIDConfiguration struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JWKSURI                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	ResponseModesSupported            []string `json:"response_modes_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	ClaimsParameterSupported          bool     `json:"claims_parameter_supported"`
	RequestParameterSupported         bool     `json:"request_parameter_supported"`
	RequestURIParameterSupported      bool     `json:"request_uri_parameter_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// HandleCallback processes the OAuth callback using OpenID Connect
func (a *Auth) HandleCallback(ctx context.Context, code, redirectURI string) (*model.User, error) {
	logger := ctxlog.From(ctx)

	// Exchange code for access token
	tokenResp, err := a.exchangeCodeForToken(code, redirectURI)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to exchange code for token")
	}

	if !tokenResp.OK || tokenResp.Error != "" {
		return nil, goerr.New("slack oauth error", goerr.V("error", tokenResp.Error))
	}

	// Decode and verify ID token
	idToken, err := a.decodeIDToken(ctx, tokenResp.IDToken)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to decode ID token")
	}

	logger.Info("Successfully decoded ID token",
		"userID", idToken.Sub,
		"userName", idToken.Name,
		"userEmail", idToken.Email,
	)

	return &model.User{
		ID:          idToken.Sub,
		SlackUserID: idToken.Sub,
		Name:        idToken.Name,
		Email:       idToken.Email,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// exchangeCodeForToken exchanges the authorization code for an access token using OpenID Connect
func (a *Auth) exchangeCodeForToken(code, redirectURI string) (*SlackTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", a.slackConfig.ClientID)
	data.Set("client_secret", a.slackConfig.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm("https://slack.com/api/openid.connect.token", data)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to make token request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	var tokenResp SlackTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, goerr.Wrap(err, "failed to parse token response")
	}

	return &tokenResp, nil
}

// getOpenIDConfiguration fetches Slack's OpenID Connect configuration
func (a *Auth) getOpenIDConfiguration(ctx context.Context) (*OpenIDConfiguration, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://slack.com/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch OpenID configuration")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, goerr.New("failed to fetch OpenID configuration", goerr.V("status", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read OpenID configuration response")
	}

	var config OpenIDConfiguration
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse OpenID configuration")
	}

	return &config, nil
}

// decodeIDToken decodes and verifies the ID token using Slack's public keys
func (a *Auth) decodeIDToken(ctx context.Context, idToken string) (*SlackIDToken, error) {
	logger := ctxlog.From(ctx)
	// Get OpenID Connect configuration to find JWKS URI
	config, err := a.getOpenIDConfiguration(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get OpenID configuration")
	}

	// Fetch Slack's public JWK set from the discovered URI
	keySet, err := jwk.Fetch(ctx, config.JWKSURI)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch Slack's public keys", goerr.V("jwks_uri", config.JWKSURI))
	}

	// Parse and verify the JWT token with clock skew tolerance
	clockFunc := jwt.ClockFunc(func() time.Time { return time.Now() })
	token, err := jwt.Parse([]byte(idToken),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithAudience(a.slackConfig.ClientID),
		jwt.WithClock(clockFunc),
		jwt.WithAcceptableSkew(5*time.Minute))
	if err != nil {
		logger.Error("JWT token validation failed",
			"error", err,
			"audience", a.slackConfig.ClientID,
			"clock_skew_tolerance", "5m")
		return nil, goerr.Wrap(err, "failed to parse or verify JWT token")
	}

	// Extract claims
	sub, ok := token.Get("sub")
	if !ok {
		return nil, goerr.New("sub claim not found in token")
	}

	email, ok := token.Get("email")
	if !ok {
		return nil, goerr.New("email claim not found in token")
	}

	name, ok := token.Get("name")
	if !ok {
		return nil, goerr.New("name claim not found in token")
	}

	// Convert to string values
	subStr, ok := sub.(string)
	if !ok {
		return nil, goerr.New("sub claim is not a string")
	}

	emailStr, ok := email.(string)
	if !ok {
		return nil, goerr.New("email claim is not a string")
	}

	nameStr, ok := name.(string)
	if !ok {
		return nil, goerr.New("name claim is not a string")
	}

	return &SlackIDToken{
		Sub:   subStr,
		Email: emailStr,
		Name:  nameStr,
	}, nil
}
