package model

import "time"

// WeeklySeverityCount holds weekly severity count data
type WeeklySeverityCount struct {
	WeekStart      time.Time
	WeekEnd        time.Time
	WeekLabel      string
	SeverityCounts map[string]int // severityID -> count
}

// SeverityCountItem holds severity count for GraphQL
type SeverityCountItem struct {
	SeverityID   string
	SeverityName string
	Count        int
}
