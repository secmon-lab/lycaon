package types

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/m-mizutani/goerr/v2"
)

// StatusHistoryID represents a status history identifier (UUID v7)
type StatusHistoryID string

// IncidentStatus represents the status of an incident
type IncidentStatus string

const (
	IncidentStatusTriage     IncidentStatus = "triage"
	IncidentStatusHandling   IncidentStatus = "handling"
	IncidentStatusMonitoring IncidentStatus = "monitoring"
	IncidentStatusClosed     IncidentStatus = "closed"
)

// String returns the string representation of the status
func (s IncidentStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s IncidentStatus) IsValid() bool {
	switch s {
	case IncidentStatusTriage, IncidentStatusHandling, IncidentStatusMonitoring, IncidentStatusClosed:
		return true
	default:
		return false
	}
}

// NewStatusHistoryID generates a new UUID v7 status history ID
func NewStatusHistoryID() StatusHistoryID {
	// UUID v7: timestamp (48 bits) + version (4 bits) + random (12 bits) + variant (2 bits) + random (62 bits)

	// Get current timestamp in milliseconds
	now := time.Now().UnixMilli()

	// Create 16 byte array for UUID
	uuid := make([]byte, 16)

	// Set timestamp (48 bits = 6 bytes)
	uuid[0] = byte(now >> 40)
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	// Fill remaining bytes with random data
	if _, err := rand.Read(uuid[6:]); err != nil {
		// Fallback to timestamp-based generation if crypto/rand fails
		for i := 6; i < 16; i++ {
			shift := 8 * (i - 6)
			if shift < 64 { // Prevent shift overflow
				uuid[i] = byte(now >> shift)
			} else {
				uuid[i] = 0
			}
		}
	}

	// Set version (7) in the upper 4 bits of byte 6
	uuid[6] = (uuid[6] & 0x0f) | 0x70

	// Set variant (10) in the upper 2 bits of byte 8
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	// Convert to hex string with dashes
	return StatusHistoryID(formatUUID(uuid))
}

// formatUUID formats a 16-byte array as a UUID string
func formatUUID(uuid []byte) string {
	return hex.EncodeToString(uuid[0:4]) + "-" +
		hex.EncodeToString(uuid[4:6]) + "-" +
		hex.EncodeToString(uuid[6:8]) + "-" +
		hex.EncodeToString(uuid[8:10]) + "-" +
		hex.EncodeToString(uuid[10:16])
}

// String returns the string representation of the status history ID
func (id StatusHistoryID) String() string {
	return string(id)
}

// Validate checks if the status history ID is valid (non-empty)
func (id StatusHistoryID) Validate() error {
	if id == "" {
		return goerr.New("status history ID cannot be empty")
	}
	return nil
}
