package model

import "github.com/m-mizutani/goerr/v2"

// Sentinel errors for incident requests
var (
	ErrIncidentRequestNotFound = goerr.New("incident request not found")
	ErrIncidentRequestExpired  = goerr.New("incident request has expired")
	ErrIncidentNotFound        = goerr.New("incident not found")
)