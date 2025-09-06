package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/urfave/cli/v3"
)

// Firestore holds Firestore configuration
type Firestore struct {
	ProjectID  string
	DatabaseID string
}

// Flags returns CLI flags for Firestore configuration
func (f *Firestore) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "firestore-project",
			Usage:       "GCP project ID for Firestore",
			Category:    "Firestore",
			Sources:     cli.EnvVars("LYCAON_FIRESTORE_PROJECT"),
			Destination: &f.ProjectID,
		},
		&cli.StringFlag{
			Name:        "firestore-database",
			Usage:       "Firestore database ID",
			Category:    "Firestore",
			Value:       "(default)",
			Sources:     cli.EnvVars("LYCAON_FIRESTORE_DATABASE"),
			Destination: &f.DatabaseID,
		},
	}
}

// Configure creates and returns a Firestore repository
func (f *Firestore) Configure(ctx context.Context) (interfaces.Repository, error) {
	logger := ctxlog.From(ctx)

	if !f.IsConfigured() {
		logger.Warn("Using memory database instead of firestore. The data will be removed when shutting down")
		// Return in-memory repository if not configured
		return repository.NewMemory(), nil
	}

	// Try to create Firestore repository
	repo, err := repository.NewFirestore(ctx, f.ProjectID, f.DatabaseID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to init firestore",
			goerr.V("project", f.ProjectID),
			goerr.V("database", f.DatabaseID),
		)
	}

	return repo, nil
}

// IsConfigured checks if Firestore is properly configured
func (f *Firestore) IsConfigured() bool {
	return f.ProjectID != ""
}

// LogValue returns structured log value
func (f Firestore) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project", f.ProjectID),
		slog.String("database", f.DatabaseID),
	)
}
