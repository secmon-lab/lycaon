import { useMemo } from 'react';
import { IncidentStatus, StatusHistory } from '../types/incident';

interface MinimalIncident {
  id: string;
  status: IncidentStatus;
  createdAt: string;
  statusHistories?: StatusHistory[];
}

interface IncidentStats {
  openCount: number;
  triageCount: number;
  handlingCount: number;
  longOpenCount: number;
  maxDaysOpen: number;
  newThisWeek: number;
  resolvedThisWeek: number;
  resolutionRate: number;
  averageResponseHours: string;
  averageResponseHoursLastWeek: string;
  responseTimeDiff: number;
  resolvedIncidentsCount: number;
}

const LONG_OPEN_THRESHOLD_DAYS = 2;
const WEEK_DAYS = 7;

export const useIncidentStats = (incidents: MinimalIncident[]): IncidentStats => {
  return useMemo(() => {
    const now = new Date();

    // Card 1: Open Incidents (Triage + Handling)
    const triageCount = incidents.filter(i => i.status === IncidentStatus.TRIAGE).length;
    const handlingCount = incidents.filter(i => i.status === IncidentStatus.HANDLING).length;
    const openCount = triageCount + handlingCount;

    // Card 2: Long Open Incidents (>2 days)
    const longOpenIncidents = incidents.filter(i => {
      if (i.status !== IncidentStatus.TRIAGE && i.status !== IncidentStatus.HANDLING) {
        return false;
      }
      const created = new Date(i.createdAt);
      const diffDays = (now.getTime() - created.getTime()) / (1000 * 60 * 60 * 24);
      return diffDays > LONG_OPEN_THRESHOLD_DAYS;
    });

    const longOpenCount = longOpenIncidents.length;
    const maxDaysOpen = longOpenIncidents.length > 0
      ? Math.max(...longOpenIncidents.map(i => {
          const created = new Date(i.createdAt);
          return Math.floor((now.getTime() - created.getTime()) / (1000 * 60 * 60 * 24));
        }))
      : 0;

    // Card 3: Weekly Trend
    const weekAgo = new Date();
    weekAgo.setDate(weekAgo.getDate() - WEEK_DAYS);

    const newThisWeek = incidents.filter(i => {
      const created = new Date(i.createdAt);
      return created >= weekAgo;
    }).length;

    const resolvedThisWeek = incidents.filter(i => {
      if (i.status !== IncidentStatus.CLOSED && i.status !== IncidentStatus.MONITORING) {
        return false;
      }
      const lastStatusChange = i.statusHistories
        ?.filter(h => h.status === IncidentStatus.CLOSED || h.status === IncidentStatus.MONITORING)
        .sort((a, b) => new Date(b.changedAt).getTime() - new Date(a.changedAt).getTime())[0];

      if (!lastStatusChange) return false;
      const resolvedAt = new Date(lastStatusChange.changedAt);
      return resolvedAt >= weekAgo;
    }).length;

    const resolutionRate = newThisWeek > 0
      ? Math.round((resolvedThisWeek / newThisWeek) * 100)
      : 0;

    // Card 4: Average Response Time (statusHistories-based)
    // Helper function to calculate response time for an incident
    const calculateResponseTime = (incident: MinimalIncident): number => {
      const histories = incident.statusHistories || [];
      let totalDuration = 0;
      let currentStart: Date | null = null;

      // Start from createdAt (initial state is Triage)
      currentStart = new Date(incident.createdAt);

      // Sort statusHistories by time
      const sortedHistories = [...histories].sort((a, b) =>
        new Date(a.changedAt).getTime() - new Date(b.changedAt).getTime()
      );

      for (const history of sortedHistories) {
        const historyTime = new Date(history.changedAt);

        if (history.status === IncidentStatus.MONITORING || history.status === IncidentStatus.CLOSED) {
          // Transition from Triage/Handling to Monitoring/Closed -> period ends
          if (currentStart) {
            totalDuration += historyTime.getTime() - currentStart.getTime();
            currentStart = null;
          }
        } else if (history.status === IncidentStatus.TRIAGE || history.status === IncidentStatus.HANDLING) {
          // Transition back to Triage/Handling from Monitoring/Closed -> period starts
          if (!currentStart) {
            currentStart = historyTime;
          }
        }
      }

      return totalDuration;
    };

    // This week's resolved incidents
    const twoWeeksAgo = new Date();
    twoWeeksAgo.setDate(twoWeeksAgo.getDate() - WEEK_DAYS * 2);

    const resolvedIncidents = incidents.filter(i =>
      i.status === IncidentStatus.MONITORING ||
      i.status === IncidentStatus.CLOSED
    );

    // This week: resolved within last 7 days
    const resolvedThisWeekIncidents = resolvedIncidents.filter(i => {
      const lastStatusChange = i.statusHistories
        ?.filter(h => h.status === IncidentStatus.CLOSED || h.status === IncidentStatus.MONITORING)
        .sort((a, b) => new Date(b.changedAt).getTime() - new Date(a.changedAt).getTime())[0];

      if (!lastStatusChange) return false;
      const resolvedAt = new Date(lastStatusChange.changedAt);
      return resolvedAt >= weekAgo;
    });

    // Last week: resolved between 7-14 days ago
    const resolvedLastWeekIncidents = resolvedIncidents.filter(i => {
      const lastStatusChange = i.statusHistories
        ?.filter(h => h.status === IncidentStatus.CLOSED || h.status === IncidentStatus.MONITORING)
        .sort((a, b) => new Date(b.changedAt).getTime() - new Date(a.changedAt).getTime())[0];

      if (!lastStatusChange) return false;
      const resolvedAt = new Date(lastStatusChange.changedAt);
      return resolvedAt >= twoWeeksAgo && resolvedAt < weekAgo;
    });

    const responseTimesThisWeek = resolvedThisWeekIncidents
      .map(calculateResponseTime)
      .filter(t => t > 0);

    const responseTimesLastWeek = resolvedLastWeekIncidents
      .map(calculateResponseTime)
      .filter(t => t > 0);

    const averageMsThisWeek = responseTimesThisWeek.length > 0
      ? responseTimesThisWeek.reduce((a, b) => a + b, 0) / responseTimesThisWeek.length
      : 0;

    const averageMsLastWeek = responseTimesLastWeek.length > 0
      ? responseTimesLastWeek.reduce((a, b) => a + b, 0) / responseTimesLastWeek.length
      : 0;

    const averageResponseHours = (averageMsThisWeek / (1000 * 60 * 60)).toFixed(1);
    const averageResponseHoursLastWeek = (averageMsLastWeek / (1000 * 60 * 60)).toFixed(1);

    // Calculate difference (positive means improvement, negative means worse)
    const responseTimeDiff = averageMsLastWeek > 0
      ? Number(((averageMsLastWeek - averageMsThisWeek) / (1000 * 60 * 60)).toFixed(1))
      : 0;

    const resolvedIncidentsCount = resolvedIncidents.length;

    return {
      openCount,
      triageCount,
      handlingCount,
      longOpenCount,
      maxDaysOpen,
      newThisWeek,
      resolvedThisWeek,
      resolutionRate,
      averageResponseHours,
      averageResponseHoursLastWeek,
      responseTimeDiff,
      resolvedIncidentsCount,
    };
  }, [incidents]);
};
