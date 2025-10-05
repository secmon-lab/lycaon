package graphql

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/secmon-lab/lycaon/pkg/domain/model"
	graphql1 "github.com/secmon-lab/lycaon/pkg/domain/model/graphql"
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
