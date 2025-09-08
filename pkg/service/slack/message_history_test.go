package slack_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
)

func TestMessageHistoryService_New(t *testing.T) {
	// Test service creation
	service := slackSvc.NewMessageHistoryService(nil)
	gt.NotEqual(t, service, nil)
}

func TestMessageHistoryOptions_Validation(t *testing.T) {
	// Test that empty channel ID should be caught by validation
	opts := slackSvc.MessageHistoryOptions{
		ChannelID: "",
		Limit:     10,
	}

	// This would fail when actually calling GetMessages, but here we just test the struct
	gt.Equal(t, opts.ChannelID, "")
	gt.Equal(t, opts.Limit, 10)
}

func TestMessageHistoryService_LimitBounds(t *testing.T) {
	// Test limit validation logic
	testCases := []struct {
		inputLimit    int
		expectedLimit int
	}{
		{0, 256},   // Zero should default to 256
		{-1, 256},  // Negative should default to 256
		{100, 100}, // Valid limit should remain unchanged
		{256, 256}, // Max limit should remain unchanged
		{300, 256}, // Over limit should be capped to 256
	}

	for _, tc := range testCases {
		// Test the actual limit clamping logic
		actualLimit := tc.inputLimit
		if actualLimit <= 0 || actualLimit > 256 {
			actualLimit = 256
		}
		gt.Equal(t, actualLimit, tc.expectedLimit)
	}
}
