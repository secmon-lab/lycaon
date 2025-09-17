package http

import "github.com/secmon-lab/lycaon/pkg/domain/interfaces"

// Test-only accessor methods for UseCases
func (u *UseCases) Auth() interfaces.Auth {
	return u.auth
}

func (u *UseCases) SlackMessage() interfaces.SlackMessage {
	return u.slackMessage
}

func (u *UseCases) Incident() interfaces.Incident {
	return u.incident
}

func (u *UseCases) Task() interfaces.Task {
	return u.task
}

func (u *UseCases) SlackInteraction() interfaces.SlackInteraction {
	return u.slackInteraction
}
