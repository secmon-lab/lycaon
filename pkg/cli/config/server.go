package config

import (
	"github.com/urfave/cli/v3"
)

// Server holds server configuration
type Server struct {
	Addr        string
	FrontendURL string
}

// Flags returns CLI flags for Server configuration
func (s *Server) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "addr",
			Usage:       "Server address",
			Value:       "localhost:8080",
			Sources:     cli.EnvVars("LYCAON_ADDR"),
			Destination: &s.Addr,
		},
		&cli.StringFlag{
			Name:        "frontend-url",
			Usage:       "Frontend URL for OAuth callbacks (if not set, automatically detected from request headers)",
			Value:       "",
			Sources:     cli.EnvVars("LYCAON_FRONTEND_URL"),
			Destination: &s.FrontendURL,
		},
	}
}
