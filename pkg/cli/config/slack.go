package config

import (
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/urfave/cli/v3"
)

// Slack holds Slack configuration
type Slack struct {
	ClientID      string
	ClientSecret  string
	SigningSecret string
	OAuthToken    string
}

// Flags returns CLI flags for Slack configuration
func (s *Slack) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "slack-client-id",
			Usage:       "Slack OAuth client ID",
			Category:    "Slack",
			Sources:     cli.EnvVars("LYCAON_SLACK_CLIENT_ID"),
			Destination: &s.ClientID,
		},
		&cli.StringFlag{
			Name:        "slack-client-secret",
			Usage:       "Slack OAuth client secret",
			Category:    "Slack",
			Sources:     cli.EnvVars("LYCAON_SLACK_CLIENT_SECRET"),
			Destination: &s.ClientSecret,
		},
		&cli.StringFlag{
			Name:        "slack-signing-secret",
			Usage:       "Slack signing secret for request verification",
			Category:    "Slack",
			Sources:     cli.EnvVars("LYCAON_SLACK_SIGNING_SECRET"),
			Destination: &s.SigningSecret,
		},
		&cli.StringFlag{
			Name:        "slack-oauth-token",
			Usage:       "Slack OAuth token for API access",
			Category:    "Slack",
			Sources:     cli.EnvVars("LYCAON_SLACK_OAUTH_TOKEN"),
			Destination: &s.OAuthToken,
		},
	}
}

// Configure creates and returns a Slack client
func (s *Slack) Configure() *slack.Client {
	if !s.IsConfigured() {
		return nil
	}
	return slack.New(s.OAuthToken)
}

// ConfigureOptional creates a Slack client if configured, returns nil if not
func (s *Slack) ConfigureOptional(logger *slog.Logger) *slack.Client {
	if !s.IsConfigured() {
		logger.Warn("Slack not configured - webhook endpoints will not work")
		return nil
	}

	logger.Info("Configuring Slack client")
	return slack.New(s.OAuthToken)
}

// IsConfigured checks if Slack is properly configured for basic operations
func (s *Slack) IsConfigured() bool {
	return s.OAuthToken != ""
}

// IsFullyConfigured checks if Slack is fully configured including signing secret
func (s *Slack) IsFullyConfigured() bool {
	return s.SigningSecret != "" && s.OAuthToken != ""
}

// IsOAuthConfigured checks if OAuth is configured
func (s *Slack) IsOAuthConfigured() bool {
	return s.ClientID != "" && s.ClientSecret != ""
}

// LogValue returns structured log value
func (s Slack) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("has_client_id", s.ClientID != ""),
		slog.Bool("has_client_secret", s.ClientSecret != ""),
		slog.Bool("has_signing_secret", s.SigningSecret != ""),
		slog.Bool("has_oauth_token", s.OAuthToken != ""),
	)
}

// SlackConfig is an alias for backward compatibility
type SlackConfig = Slack
