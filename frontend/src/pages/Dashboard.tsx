import React from 'react';
import { useQuery } from '@apollo/client/react';
import { OpenIncidentsList } from '../components/Dashboard/OpenIncidentsList';
import { SeverityTrendChart } from '../components/Dashboard/SeverityTrendChart';
import {
  GET_RECENT_OPEN_INCIDENTS,
  GET_INCIDENT_TREND_BY_SEVERITY,
} from '../graphql/queries';
import {
  RecentOpenIncidentsData,
  IncidentTrendBySeverityData,
} from '../types/dashboard';

const Dashboard: React.FC = () => {
  const weeks = 8;
  const days = 14;

  const {
    data: incidentsData,
    loading: incidentsLoading,
    error: incidentsError,
  } = useQuery<RecentOpenIncidentsData>(GET_RECENT_OPEN_INCIDENTS, {
    variables: { days },
  });

  const {
    data: trendData,
    loading: trendLoading,
    error: trendError,
  } = useQuery<IncidentTrendBySeverityData>(GET_INCIDENT_TREND_BY_SEVERITY, {
    variables: { weeks },
  });

  return (
    <div className="space-y-6">
      <SeverityTrendChart
        data={trendData?.incidentTrendBySeverity || []}
        loading={trendLoading}
        error={trendError}
        weeks={weeks}
      />

      <OpenIncidentsList
        incidents={incidentsData?.recentOpenIncidents || []}
        loading={incidentsLoading}
        error={incidentsError}
        days={days}
      />
    </div>
  );
};

export default Dashboard;
