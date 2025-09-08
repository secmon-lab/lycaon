package model

import (
	"time"

	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// Message represents a Slack message
type Message struct {
	ID        types.MessageID
	UserID    types.SlackUserID
	UserName  string
	ChannelID types.ChannelID
	Text      string
	Timestamp time.Time
	ThreadTS  types.ThreadTS
	EventTS   types.EventTS
}

// NewMessage creates a new Message instance
func NewMessage(id types.MessageID, userID types.SlackUserID, userName string, channelID types.ChannelID, text string) *Message {
	return &Message{
		ID:        id,
		UserID:    userID,
		UserName:  userName,
		ChannelID: channelID,
		Text:      text,
		Timestamp: time.Now(),
	}
}
