package model

// SanitizeForSlackChannelName exposes the private sanitizeForSlackChannelName function for testing
func SanitizeForSlackChannelName(text string) string {
	return sanitizeForSlackChannelName(text)
}
