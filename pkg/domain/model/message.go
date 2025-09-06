package model

import (
	"time"
)

// Message represents a Slack message
type Message struct {
	ID        string
	UserID    string
	UserName  string
	ChannelID string
	Text      string
	Timestamp time.Time
	ThreadTS  string
	EventTS   string
}

// NewMessage creates a new Message instance
func NewMessage(id, userID, userName, channelID, text string) *Message {
	return &Message{
		ID:        id,
		UserID:    userID,
		UserName:  userName,
		ChannelID: channelID,
		Text:      text,
		Timestamp: time.Now(),
	}
}
