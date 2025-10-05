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

interface User {
  id: string;
  name: string;
  email: string;
  slack_user_id: string;
}

interface DashboardProps {
  user: User;
  setUser: (user: User | null) => void;
}

const Dashboard: React.FC<DashboardProps> = () => {
  const {
    data: incidentsData,
    loading: incidentsLoading,
    error: incidentsError,
  } = useQuery<RecentOpenIncidentsData>(GET_RECENT_OPEN_INCIDENTS, {
    variables: { days: 14 },
  });

  const {
    data: trendData,
    loading: trendLoading,
    error: trendError,
  } = useQuery<IncidentTrendBySeverityData>(GET_INCIDENT_TREND_BY_SEVERITY, {
    variables: { weeks: 8 },
  });

  return (
    <div className="space-y-6">
      <SeverityTrendChart
        data={trendData?.incidentTrendBySeverity || []}
        loading={trendLoading}
        error={trendError}
      />

      <OpenIncidentsList
        incidents={incidentsData?.recentOpenIncidents || []}
        loading={incidentsLoading}
        error={incidentsError}
      />
    </div>
  );
};

export default Dashboard;
