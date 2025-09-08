package model

import "github.com/m-mizutani/goerr/v2"

// Sentinel errors for domain operations
var (
	ErrIncidentRequestNotFound = goerr.New("incident request not found")
	ErrIncidentNotFound        = goerr.New("incident not found")
)
