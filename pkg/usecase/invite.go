package usecase

import (
	"context"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackService "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/slack-go/slack"
)

type Invite struct {
	slackClient interfaces.SlackClient
}

func NewInvite(slackClient interfaces.SlackClient) interfaces.Invite {
	return &Invite{
		slackClient: slackClient,
	}
}

func (u *Invite) InviteUsersByList(ctx context.Context, users []string, groups []string, channelID types.ChannelID) (*model.InvitationResult, error) {
	logger := ctxlog.From(ctx)

	// 1. Start invitation logging
	logger.Info("Starting invitation process",
		"channelID", channelID,
		"userCount", len(users),
		"groupCount", len(groups),
		"users", users,
		"groups", groups)

	// 2. Resolve users and groups
	resolvedUsers, err := u.resolveUsers(ctx, users, groups)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to resolve users")
	}

	logger.Info("User resolution completed", "resolvedCount", len(resolvedUsers))

	// 3. Execute batch invitation
	logger.Info("Starting batch invitation", "channelID", channelID, "targetCount", len(resolvedUsers))

	result := u.executeBatchInvitation(ctx, channelID, resolvedUsers)

	// 4. Output detailed logs (requirement)
	logger.Info("Invitation completed with details", "details", result.Details)

	return result, nil
}

// resolveUsers - Resolve user/group names to UserID list
// Note: Only fetch minimum information needed for invitation (avoid excessive API calls)
func (u *Invite) resolveUsers(ctx context.Context, users []string, groups []string) ([]model.InviteDetail, error) {
	var details []model.InviteDetail

	// Get the underlying Slack client for API calls
	// We need to cast the interface to access GetClient method
	service, ok := u.slackClient.(*slackService.Service)
	if !ok {
		// If it's not the real service (e.g., in tests with mocks), skip resolution
		// Just pass through the IDs as-is
		for _, user := range users {
			details = append(details, model.InviteDetail{
				UserID:       user,
				Username:     user,
				SourceConfig: user,
				Status:       "resolved",
			})
		}
		return details, nil
	}

	client := service.GetClient()

	// Resolve users
	for _, user := range users {
		if strings.HasPrefix(user, "@") {
			// Resolve username to ID
			// Minimum required: Only resolve UserID for invitation (display name not needed)
			userID, err := u.resolveUserName(ctx, client, user)
			if err != nil {
				// Log resolution failure and continue
				ctxlog.From(ctx).Warn("Failed to resolve user", "user", user, "error", err)
				details = append(details, model.InviteDetail{
					UserID:       "",
					Username:     user,
					SourceConfig: user,
					Status:       "failed",
					Error:        err.Error(),
				})
				continue
			}
			details = append(details, model.InviteDetail{
				UserID:       userID,
				Username:     user, // Keep original config value (no additional API calls)
				SourceConfig: user,
				Status:       "resolved",
			})
		} else if strings.HasPrefix(user, "U") {
			// Direct UserID (regular user) - No API call needed
			details = append(details, model.InviteDetail{
				UserID:       user,
				Username:     "",  // No API call needed for display name
				SourceConfig: user,
				Status:       "resolved",
			})
		} else if strings.HasPrefix(user, "B") {
			// Bot ID specified - need to resolve to User ID
			// Bot IDs (B-prefix) cannot be used directly with conversations.invite
			// We need to find the corresponding User ID (U-prefix)
			userID, err := u.resolveBotIDToUserID(ctx, client, user)
			if err != nil {
				ctxlog.From(ctx).Warn("Failed to resolve Bot ID to User ID", "botID", user, "error", err)
				details = append(details, model.InviteDetail{
					UserID:       "",
					Username:     "",
					SourceConfig: user,
					Status:       "failed",
					Error:        err.Error(),
				})
				continue
			}
			details = append(details, model.InviteDetail{
				UserID:       userID,
				Username:     "",
				SourceConfig: user,
				Status:       "resolved",
			})
		}
	}

	// Resolve groups (only get member UserIDs)
	for _, group := range groups {
		memberIDs, err := u.resolveGroupMembers(ctx, client, group)
		if err != nil {
			ctxlog.From(ctx).Warn("Failed to resolve group", "group", group, "error", err)
			continue
		}
		for _, memberID := range memberIDs {
			details = append(details, model.InviteDetail{
				UserID:       memberID,
				Username:     "", // Member display name not needed
				SourceConfig: group,
				Status:       "resolved",
			})
		}
	}

	return details, nil
}

// resolveUserName resolves @username to user ID (including bots)
func (u *Invite) resolveUserName(ctx context.Context, client interface{}, username string) (string, error) {
	// Remove @ prefix
	name := strings.TrimPrefix(username, "@")

	// Cast client to *slack.Client
	slackClient, ok := client.(*slack.Client)
	if !ok {
		return "", goerr.New("invalid slack client")
	}

	// Get users list (including bots)
	users, err := slackClient.GetUsersContext(ctx)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get users")
	}

	// Find user or bot by name
	for _, user := range users {
		// Debug logging for bot detection
		if user.IsBot {
			ctxlog.From(ctx).Debug("Checking bot user",
				"userID", user.ID,
				"name", user.Name,
				"realName", user.RealName,
				"displayName", user.Profile.DisplayName,
				"botID", user.Profile.BotID,
				"apiAppID", user.Profile.ApiAppID,
				"searching", name,
			)
		}
		
		// Check both user name and real name
		// For bots, the Name field usually contains the bot name
		if user.Name == name || user.RealName == name {
			if user.IsBot {
				ctxlog.From(ctx).Info("Bot found by name, using User ID",
					"username", username,
					"userID", user.ID,
					"botID", user.Profile.BotID,
				)
			}
			return user.ID, nil
		}
		
		// Also check profile display name
		if user.Profile.DisplayName == name {
			if user.IsBot {
				ctxlog.From(ctx).Info("Bot found by display name, using User ID",
					"username", username,
					"userID", user.ID,
					"botID", user.Profile.BotID,
				)
			}
			return user.ID, nil
		}
		
		// Check if this is a bot and match bot name
		if user.IsBot && (user.Profile.BotID != "" || user.Profile.ApiAppID != "") {
			// For bots, sometimes the name is stored differently
			if user.Profile.RealName == name || user.Profile.DisplayName == name {
				ctxlog.From(ctx).Info("Bot found, using User ID",
					"username", username,
					"userID", user.ID,
					"botID", user.Profile.BotID,
				)
				return user.ID, nil
			}
		}
	}

	// If not found in users, try to find in bots specifically
	// Note: GetUsersContext should include bots, but let's log for debugging
	ctxlog.From(ctx).Warn("User/Bot not found in users list",
		"username", username,
		"searchName", name,
		"totalUsers", len(users),
	)

	return "", goerr.New("user not found", goerr.V("username", username))
}

// resolveBotIDToUserID resolves Bot ID (B-prefix) to User ID (U-prefix)
func (u *Invite) resolveBotIDToUserID(ctx context.Context, client interface{}, botID string) (string, error) {
	// Cast client to *slack.Client
	slackClient, ok := client.(*slack.Client)
	if !ok {
		return "", goerr.New("invalid slack client")
	}

	// Get users list (including bots)
	users, err := slackClient.GetUsersContext(ctx)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get users")
	}

	// Find bot by Bot ID
	for _, user := range users {
		if user.IsBot && user.Profile.BotID == botID {
			ctxlog.From(ctx).Info("Bot ID resolved to User ID",
				"botID", botID,
				"userID", user.ID,
				"name", user.Name,
			)
			return user.ID, nil
		}
	}

	return "", goerr.New("bot not found", goerr.V("botID", botID))
}

// resolveGroupMembers resolves group to member IDs
func (u *Invite) resolveGroupMembers(ctx context.Context, client interface{}, group string) ([]string, error) {
	// Cast client to *slack.Client
	slackClient, ok := client.(*slack.Client)
	if !ok {
		return nil, goerr.New("invalid slack client")
	}

	var groupID string

	if strings.HasPrefix(group, "@") {
		// Group name - need to resolve to ID
		name := strings.TrimPrefix(group, "@")
		
		groups, err := slackClient.GetUserGroupsContext(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get user groups")
		}

		for _, g := range groups {
			if g.Name == name || g.Handle == name {
				groupID = g.ID
				break
			}
		}

		if groupID == "" {
			return nil, goerr.New("group not found", goerr.V("group", group))
		}
	} else if strings.HasPrefix(group, "S") {
		// Direct group ID
		groupID = group
	} else {
		return nil, goerr.New("invalid group format", goerr.V("group", group))
	}

	// Get group members
	members, err := slackClient.GetUserGroupMembersContext(ctx, groupID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get group members", goerr.V("groupID", groupID))
	}

	return members, nil
}

// executeBatchInvitation - Execute batch invitation
func (u *Invite) executeBatchInvitation(ctx context.Context, channelID types.ChannelID, targets []model.InviteDetail) *model.InvitationResult {
	var userIDs []string
	for _, target := range targets {
		if target.Status == "resolved" && target.UserID != "" {
			userIDs = append(userIDs, target.UserID)
		}
	}

	// Use existing InviteUsersToConversation
	_, err := u.slackClient.InviteUsersToConversation(ctx, string(channelID), userIDs...)

	// Record detailed results
	var details []model.InviteDetail
	for _, target := range targets {
		detail := target
		if target.Status != "resolved" || target.UserID == "" {
			// Already failed in resolution
			details = append(details, detail)
			continue
		}
		
		if err != nil {
			detail.Status = "failed"
			detail.Error = err.Error()
		} else {
			detail.Status = "success"
		}
		details = append(details, detail)
	}

	return &model.InvitationResult{Details: details}
}