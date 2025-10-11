package graphql

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	graphql1 "github.com/secmon-lab/lycaon/pkg/domain/model/graphql"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// convertToGroupedIncidents converts incidents map to grouped incidents slice
func convertToGroupedIncidents(ctx context.Context, incidentsMap map[string][]*model.Incident) []*graphql1.GroupedIncidents {
	if len(incidentsMap) == 0 {
		return []*graphql1.GroupedIncidents{}
	}

	// Extract dates and sort descending
	dates := make([]string, 0, len(incidentsMap))
	for date := range incidentsMap {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	// Build result
	result := make([]*graphql1.GroupedIncidents, 0, len(dates))
	for _, date := range dates {
		incidents := incidentsMap[date]
		if len(incidents) == 0 {
			continue
		}

		// Parse date string to time.Time
		dateTime, err := time.Parse("2006-01-02", date)
		if err != nil {
			// Log error and skip invalid date
			slog.WarnContext(ctx, "Failed to parse date string, skipping this group",
				"date", date,
				"error", err,
			)
			continue
		}

		result = append(result, &graphql1.GroupedIncidents{
			Date:      dateTime,
			Incidents: incidents,
		})
	}

	return result
}

// getSlackUserIDFromContext retrieves the SlackUserID from the context
func getSlackUserIDFromContext(ctx context.Context) (types.SlackUserID, bool) {
	authCtx, ok := model.GetAuthContext(ctx)
	if !ok || authCtx == nil {
		return "", false
	}
	if authCtx.SlackUserID == "" {
		return "", false
	}
	return types.SlackUserID(authCtx.SlackUserID), true
}

// filterIncidentForUser filters incident information based on user access
func filterIncidentForUser(ctx context.Context, incident *model.Incident, incidentUC interfaces.Incident, slackUserID types.SlackUserID) *model.Incident {
	if incident == nil {
		return nil
	}
	return incidentUC.FilterIncidentForUser(ctx, incident, slackUserID)
}

// filterIncidentsForUser filters multiple incidents based on user access
func filterIncidentsForUser(ctx context.Context, incidents []*model.Incident, incidentUC interfaces.Incident, slackUserID types.SlackUserID) []*model.Incident {
	if len(incidents) == 0 {
		return incidents
	}

	filtered := make([]*model.Incident, len(incidents))
	for i, incident := range incidents {
		filtered[i] = filterIncidentForUser(ctx, incident, incidentUC, slackUserID)
	}
	return filtered
}
