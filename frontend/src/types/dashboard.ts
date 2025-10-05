import { Incident } from './incident';

export interface GroupedIncidents {
  date: string;
  incidents: Incident[];
}

export interface SeverityCount {
  severityId: string;
  severityName: string;
  severityLevel: number;
  count: number;
}

export interface WeeklySeverityCount {
  weekStart: string;
  weekEnd: string;
  weekLabel: string;
  severityCounts: SeverityCount[];
}

export interface RecentOpenIncidentsData {
  recentOpenIncidents: GroupedIncidents[];
}

export interface IncidentTrendBySeverityData {
  incidentTrendBySeverity: WeeklySeverityCount[];
}
