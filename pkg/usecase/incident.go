package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/utils/apperr"
	"github.com/slack-go/slack"
)

// IncidentConfig holds configuration for Incident use case
type IncidentConfig struct {
	channelPrefix string
	frontendURL   string
}

// IncidentOption is a functional option for configuring Incident
type IncidentOption func(*IncidentConfig)

// WithChannelPrefix sets the channel prefix for incident channels
func WithChannelPrefix(prefix string) IncidentOption {
	return func(c *IncidentConfig) {
		c.channelPrefix = prefix
	}
}

// WithFrontendURL sets the frontend URL for bookmark generation
func WithFrontendURL(url string) IncidentOption {
	return func(c *IncidentConfig) {
		c.frontendURL = url
	}
}

// NewIncidentConfig creates a new IncidentConfig with default values and optional settings
func NewIncidentConfig(opts ...IncidentOption) *IncidentConfig {
	config := &IncidentConfig{
		channelPrefix: "inc", // Default value
	}

	// Apply optional configurations
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// Incident implements Incident interface
type Incident struct {
	repo        interfaces.Repository
	slackClient interfaces.SlackClient
	slackSvc    *slackSvc.UIService
	modelConfig *model.Config
	invite      interfaces.Invite
	config      *IncidentConfig
}

// NewIncident creates a new Incident instance with configuration
func NewIncident(repo interfaces.Repository, slackClient interfaces.SlackClient, slackService *slackSvc.UIService, modelConfig *model.Config, invite interfaces.Invite, config *IncidentConfig) *Incident {
	return &Incident{
		repo:        repo,
		slackClient: slackClient,
		slackSvc:    slackService,
		modelConfig: modelConfig,
		invite:      invite,
		config:      config,
	}
}

// CreateIncident creates a new incident
func (u *Incident) CreateIncident(ctx context.Context, req *model.CreateIncidentRequest) (*model.Incident, error) {
	// Validate severity ID if severities config is available and severity ID is provided
	if u.modelConfig != nil && req.SeverityID != "" {
		severity := u.modelConfig.FindSeverityByID(req.SeverityID)
		if severity == nil {
			return nil, goerr.New("invalid severity ID",
				goerr.V("severityID", req.SeverityID))
		}
	}

	// Get next incident number
	incidentNumber, err := u.repo.GetNextIncidentNumber(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get next incident number")
	}

	// Get Team ID from Slack API
	var teamID types.TeamID
	authResp, err := u.slackClient.AuthTestContext(ctx)
	if err != nil {
		// Log warning but continue without TeamID
		ctxlog.From(ctx).Warn("Failed to get Team ID from Slack API",
			"error", err,
			"description", "Slack channel links will not work without Team ID")
	} else {
		teamID = types.TeamID(authResp.TeamID)
	}

	// Create incident model
	incident, err := model.NewIncident(u.config.channelPrefix, incidentNumber, req.Title, req.Description, req.CategoryID, types.SeverityID(req.SeverityID), req.AssetIDs, types.ChannelID(req.OriginChannelID), types.ChannelName(req.OriginChannelName), teamID, types.SlackUserID(req.CreatedBy), req.InitialTriage)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create incident model")
	}

	// Create Slack channel with the generated channel name (includes title if provided)
	channelName := incident.ChannelName

	channel, err := u.slackClient.CreateConversation(ctx, slack.CreateConversationParams{
		ChannelName: channelName.String(),
		IsPrivate:   false,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Slack channel")
	}

	// Set channel purpose/description if title is provided
	if req.Title != "" {
		_, err = u.slackClient.SetPurposeOfConversationContext(ctx, channel.ID, req.Title)
		if err != nil {
			// Log error but don't fail - setting purpose is nice to have but not critical
			apperr.Handle(ctx, err)
		}
	}

	// Set channel ID
	incident.ChannelID = types.ChannelID(channel.ID)

	// Invite creator to the channel
	_, err = u.slackClient.InviteUsersToConversation(ctx, channel.ID, req.CreatedBy)
	if err != nil {
		// Log error but don't fail - user might already be in channel or invitation might fail for other reasons
		apperr.Handle(ctx, err)
	}

	// Add bookmark to Web UI if frontend URL is configured
	if u.config.frontendURL != "" {
		bookmarkTitle := fmt.Sprintf("Incident #%d - Web UI", incidentNumber)
		bookmarkURL := fmt.Sprintf("%s/incidents/%d", u.config.frontendURL, incident.ID)

		err = u.slackClient.AddBookmark(ctx, channel.ID, bookmarkTitle, bookmarkURL)
		if err != nil {
			// Log error but don't fail - bookmark is nice to have but not critical
			apperr.Handle(ctx, err)
		} else {
			ctxlog.From(ctx).Info("Added incident bookmark to channel",
				"channelID", channel.ID,
				"incidentID", incident.ID,
				"bookmarkURL", bookmarkURL)
		}
	}

	// Post welcome message to the incident channel
	leadName := string(incident.Lead)
	if leadName == "" {
		leadName = "Unassigned"
	}
	welcomeTS, err := u.slackSvc.PostWelcomeMessage(
		ctx,
		incident.ChannelID,
		incident,
		req.OriginChannelName,
		leadName,
	)
	if err != nil {
		// Log error but don't fail - welcome message is nice to have but not critical
		apperr.Handle(ctx, err)
	} else {
		// Save the welcome message timestamp for later updates
		incident.WelcomeMessageTS = welcomeTS
	}

	// Save incident to repository
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to save incident")
	}

	// Save initial status history
	initialHistory := &model.StatusHistory{
		ID:         types.NewStatusHistoryID(),
		IncidentID: incident.ID,
		Status:     incident.Status,
		ChangedBy:  incident.CreatedBy,
		ChangedAt:  incident.CreatedAt,
		Note:       "Incident created",
	}
	if err := u.repo.AddStatusHistory(ctx, initialHistory); err != nil {
		// Log error and fail the operation to prevent inconsistent data
		apperr.Handle(ctx, err)
		return nil, goerr.Wrap(err, "failed to save initial status history")
	}

	// Category-based invitation process (serial execution)
	// Note: This function assumes it's already dispatched asynchronously in the Controller layer
	if incident.CategoryID != "" && u.modelConfig != nil && u.invite != nil {
		// Get invitation targets from category configuration
		category := u.modelConfig.FindCategoryByID(incident.CategoryID)
		if category == nil {
			// No category is normal - log and continue
			ctxlog.From(ctx).Info("Category not found",
				"categoryID", incident.CategoryID)
		} else if len(category.InviteUsers) == 0 && len(category.InviteGroups) == 0 {
			// No invitation settings is normal - log and continue
			ctxlog.From(ctx).Info("No invitation settings for category",
				"categoryID", incident.CategoryID)
		} else {
			// Execute invitation synchronously
			_, err := u.invite.InviteUsersByList(
				ctx,
				category.InviteUsers,
				category.InviteGroups,
				incident.ChannelID,
			)
			if err != nil {
				// Log error but continue with incident creation
				apperr.Handle(ctx, err)
			}
		}
	}

	return incident, nil
}

// GetIncident retrieves an incident by ID
func (u *Incident) GetIncident(ctx context.Context, id int) (*model.Incident, error) {
	incident, err := u.repo.GetIncident(ctx, types.IncidentID(id))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}
	return incident, nil
}

// GetIncidentByChannelID gets an incident by channel ID
func (u *Incident) GetIncidentByChannelID(ctx context.Context, channelID types.ChannelID) (*model.Incident, error) {
	incident, err := u.repo.GetIncidentByChannelID(ctx, channelID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident by channel ID")
	}
	return incident, nil
}

// HandleCreateIncidentAction handles the complete flow when a user clicks the create incident button
func (u *Incident) HandleCreateIncidentAction(ctx context.Context, requestID, userID string) (*model.Incident, error) {
	return u.handleCreateIncidentFromRequest(ctx, requestID, userID, nil)
}

// UpdateIncidentDetails updates incident title, description, and lead
func (u *Incident) UpdateIncidentDetails(ctx context.Context, incidentID types.IncidentID, title, description string, lead types.SlackUserID, severityID string, updatedBy types.SlackUserID) (*model.Incident, error) {
	// Validate incident ID
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	// Validate severity ID if provided and severities config is available
	if severityID != "" && u.modelConfig != nil {
		severity := u.modelConfig.FindSeverityByID(severityID)
		if severity == nil {
			return nil, goerr.New("invalid severity ID", goerr.V("severityID", severityID))
		}
	}

	// Get existing incident
	incident, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}

	// Track if any changes were made
	hasChanges := false

	// Update fields if they have changed
	if title != incident.Title {
		incident.Title = title
		hasChanges = true
	}
	if description != incident.Description {
		incident.Description = description
		hasChanges = true
	}
	if lead != incident.Lead {
		incident.Lead = lead
		hasChanges = true
	}
	if severityID != "" && types.SeverityID(severityID) != incident.SeverityID {
		incident.SeverityID = types.SeverityID(severityID)
		hasChanges = true
	}

	// Only update if there are changes
	if !hasChanges {
		return incident, nil
	}

	// Save updated incident using PutIncident
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to update incident")
	}

	return incident, nil
}

// UpdateIncidentDetailsWithAssets updates incident title, description, lead, severity, and assets
func (u *Incident) UpdateIncidentDetailsWithAssets(ctx context.Context, incidentID types.IncidentID, title, description string, lead types.SlackUserID, severityID string, assetIDs []types.AssetID, updatedBy types.SlackUserID) (*model.Incident, error) {
	// Validate incident ID
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	// Validate severity ID if provided and severities config is available
	if severityID != "" && u.modelConfig != nil {
		severity := u.modelConfig.FindSeverityByID(severityID)
		if severity == nil {
			return nil, goerr.New("invalid severity ID", goerr.V("severityID", severityID))
		}
	}

	// Validate asset IDs if provided and assets config is available
	if len(assetIDs) > 0 && u.modelConfig != nil {
		if err := u.modelConfig.ValidateAssetIDs(assetIDs); err != nil {
			return nil, goerr.Wrap(err, "invalid asset IDs")
		}
	}

	// Get existing incident
	incident, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}

	// Track if any changes were made
	hasChanges := false

	// Update fields if they have changed
	if title != incident.Title {
		incident.Title = title
		hasChanges = true
	}
	if description != incident.Description {
		incident.Description = description
		hasChanges = true
	}
	if lead != incident.Lead {
		incident.Lead = lead
		hasChanges = true
	}
	if severityID != "" && types.SeverityID(severityID) != incident.SeverityID {
		incident.SeverityID = types.SeverityID(severityID)
		hasChanges = true
	}

	// Update assets if they have changed
	assetIDsChanged := false
	if len(assetIDs) != len(incident.AssetIDs) {
		assetIDsChanged = true
	} else {
		currentAssets := make(map[types.AssetID]struct{})
		for _, id := range incident.AssetIDs {
			currentAssets[id] = struct{}{}
		}
		for _, id := range assetIDs {
			if _, ok := currentAssets[id]; !ok {
				assetIDsChanged = true
				break
			}
		}
	}

	if assetIDsChanged {
		incident.AssetIDs = assetIDs
		hasChanges = true
	}

	// Only update if there are changes
	if !hasChanges {
		return incident, nil
	}

	// Save updated incident using PutIncident
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to update incident")
	}

	return incident, nil
}

// UpdateIncident updates incident with UpdateIncidentRequest
func (u *Incident) UpdateIncident(ctx context.Context, incidentID types.IncidentID, req model.UpdateIncidentRequest) (*model.Incident, error) {
	// Validate incident ID
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	// Validate severity ID if provided
	if req.SeverityID != nil && u.modelConfig != nil {
		severityID := string(*req.SeverityID)
		if severityID != "" {
			severity := u.modelConfig.FindSeverityByID(severityID)
			if severity == nil {
				return nil, goerr.New("invalid severity ID",
					goerr.V("severityID", severityID))
			}
		}
	}

	// Get existing incident
	incident, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}

	// Track if any changes were made
	hasChanges := false
	var changes []string

	// Update fields if they have changed
	if req.Title != nil && *req.Title != incident.Title {
		incident.Title = *req.Title
		hasChanges = true
		changes = append(changes, "title")
	}
	if req.Description != nil && *req.Description != incident.Description {
		incident.Description = *req.Description
		hasChanges = true
		changes = append(changes, "description")
	}
	if req.Lead != nil && *req.Lead != incident.Lead {
		incident.Lead = *req.Lead
		hasChanges = true
		changes = append(changes, "lead")
	}
	if req.Status != nil && *req.Status != incident.Status {
		incident.Status = *req.Status
		hasChanges = true
		changes = append(changes, "status")
	}
	if req.SeverityID != nil && *req.SeverityID != incident.SeverityID {
		incident.SeverityID = *req.SeverityID
		hasChanges = true
		changes = append(changes, "severity")
	}

	// Update assets if provided
	if req.AssetIDs != nil {
		assetIDs := *req.AssetIDs

		// Check if asset IDs have changed
		assetIDsChanged := false
		if len(assetIDs) != len(incident.AssetIDs) {
			assetIDsChanged = true
		} else {
			currentAssets := make(map[types.AssetID]struct{})
			for _, id := range incident.AssetIDs {
				currentAssets[id] = struct{}{}
			}
			for _, id := range assetIDs {
				if _, ok := currentAssets[id]; !ok {
					assetIDsChanged = true
					break
				}
			}
		}

		if assetIDsChanged {
			incident.AssetIDs = assetIDs
			hasChanges = true
			changes = append(changes, "assets")
		}
	}

	// Only update if there are changes
	if !hasChanges {
		return incident, nil
	}

	// Save updated incident using PutIncident
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to update incident")
	}

	// Post update notification to incident channel
	if incident.ChannelID != "" {
		// Build a simple notification message
		message := fmt.Sprintf("📝 Incident details updated: %s", strings.Join(changes, ", "))
		_, _, err = u.slackClient.PostMessage(
			ctx,
			string(incident.ChannelID),
			slack.MsgOptionText(message, false),
		)
		if err != nil {
			// Log error but don't fail - notification is nice to have but not critical
			apperr.Handle(ctx, err)
		}
	}

	return incident, nil
}

// GetRecentOpenIncidents gets recent open incidents grouped by date
func (u *Incident) GetRecentOpenIncidents(ctx context.Context, days int) (map[string][]*model.Incident, error) {
	// Validate input
	if days <= 0 {
		days = 7 // Default to 7 days
	}

	// Get incidents since cutoff time
	cutoffTime := time.Now().AddDate(0, 0, -days)
	incidents, err := u.repo.ListIncidentsSince(ctx, cutoffTime)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list incidents since cutoff time")
	}

	// Filter by status and group by date
	result := make(map[string][]*model.Incident)

	for _, incident := range incidents {
		// Skip closed incidents
		if incident.Status == types.IncidentStatusClosed {
			continue
		}

		// Group by date (YYYY-MM-DD format)
		dateKey := incident.CreatedAt.Format("2006-01-02")
		result[dateKey] = append(result[dateKey], incident)
	}

	// Sort incidents within each date group (newest first)
	for _, incidents := range result {
		sort.Slice(incidents, func(i, j int) bool {
			return incidents[i].CreatedAt.After(incidents[j].CreatedAt)
		})
	}

	return result, nil
}

// GetIncidentTrendBySeverity gets incident trend by severity for specified weeks
func (u *Incident) GetIncidentTrendBySeverity(ctx context.Context, weeks int) ([]*model.WeeklySeverityCount, error) {
	// Validate input
	if weeks <= 0 {
		weeks = 4 // Default to 4 weeks
	}

	// Calculate week ranges
	now := time.Now()
	startOfCurrentWeek := getStartOfWeek(now)
	weekRanges := generateWeekRanges(startOfCurrentWeek, weeks)

	// Get incidents since the oldest week start
	oldestWeekStart := weekRanges[0].Start
	incidents, err := u.repo.ListIncidentsSince(ctx, oldestWeekStart)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list incidents since oldest week")
	}

	// Count incidents by week and severity
	result := make([]*model.WeeklySeverityCount, len(weekRanges))
	for i, weekRange := range weekRanges {
		counts := make(map[string]int)

		for _, incident := range incidents {
			// Check if incident is within this week
			if incident.CreatedAt.Before(weekRange.Start) || incident.CreatedAt.After(weekRange.End) {
				continue
			}
			// Increment count for this severity
			severityID := string(incident.SeverityID)
			counts[severityID]++
		}

		result[i] = &model.WeeklySeverityCount{
			WeekStart:      weekRange.Start,
			WeekEnd:        weekRange.End,
			WeekLabel:      weekRange.Label,
			SeverityCounts: counts,
		}
	}

	return result, nil
}

// weekRange represents a week time range
type weekRange struct {
	Start time.Time
	End   time.Time
	Label string
}

// getStartOfWeek returns the start of the week (Monday 00:00:00) for a given time
func getStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	// Convert Sunday (0) to 7, then calculate days to subtract to get to Monday
	if weekday == 0 {
		weekday = 7
	}
	daysToMonday := weekday - 1
	// Get the date of Monday, preserving timezone
	monday := t.AddDate(0, 0, -daysToMonday)
	// Return Monday at 00:00:00 in the same timezone
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

// generateWeekRanges generates week ranges from the current week backwards
func generateWeekRanges(startOfCurrentWeek time.Time, weeks int) []weekRange {
	ranges := make([]weekRange, weeks)

	for i := 0; i < weeks; i++ {
		weekOffset := weeks - 1 - i // Start from oldest week
		start := startOfCurrentWeek.AddDate(0, 0, -7*weekOffset)
		// Calculate Sunday (6 days after Monday) at 23:59:59 in the same timezone
		sunday := start.AddDate(0, 0, 6)
		end := time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 0, sunday.Location())

		// Format: "Jan 2-8" or "Dec 30-Jan 5" for cross-month weeks
		startMonth := start.Format("Jan")
		endMonth := end.Format("Jan")
		if startMonth == endMonth {
			ranges[i] = weekRange{
				Start: start,
				End:   end,
				Label: fmt.Sprintf("%s %d-%d", startMonth, start.Day(), end.Day()),
			}
		} else {
			ranges[i] = weekRange{
				Start: start,
				End:   end,
				Label: fmt.Sprintf("%s %d-%s %d", startMonth, start.Day(), endMonth, end.Day()),
			}
		}
	}

	return ranges
}
