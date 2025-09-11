package model

// InvitationResult represents the result of an invitation operation
type InvitationResult struct {
	Details []InviteDetail // All user details
}

// InviteDetail represents the details of a single user invitation
type InviteDetail struct {
	UserID       string // Slack User ID
	Username     string // Display name or original config value
	Status       string // "success" or "failed"
	SourceConfig string // Original config value (@username, U123456, etc.)
	Error        string // Error message if failed
}

