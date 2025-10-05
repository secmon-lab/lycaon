import React, { useMemo } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ChartOptions,
} from 'chart.js';
import { Bar } from 'react-chartjs-2';

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
);

interface SeverityCount {
  severityId: string;
  severityName: string;
  severityLevel: number;
  count: number;
}

interface WeeklySeverityData {
  weekLabel: string;
  weekStart: string;
  weekEnd: string;
  severityCounts: SeverityCount[];
}

interface SeverityTrendChartProps {
  data: WeeklySeverityData[];
  loading?: boolean;
  error?: Error;
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#dc2626',
  high: '#ea580c',
  medium: '#ca8a04',
  low: '#16a34a',
  ignorable: '#6b7280',
  unknown: '#9ca3af',
};

export const SeverityTrendChart: React.FC<SeverityTrendChartProps> = ({
  data,
  loading,
  error,
}) => {
  // Get all unique severity IDs sorted by level (ascending - low to high for stacking order)
  // Chart.js stacks from bottom to top, so we need low severity first
  const severityList = useMemo(() => {
    if (!data || data.length === 0) return [];

    const severityMap = new Map<string, { id: string; name: string; level: number }>();
    data.forEach((week) => {
      week.severityCounts.forEach((sc) => {
        if (!severityMap.has(sc.severityId)) {
          severityMap.set(sc.severityId, {
            id: sc.severityId,
            name: sc.severityName,
            level: sc.severityLevel,
          });
        }
      });
    });

    return Array.from(severityMap.values()).sort((a, b) => a.level - b.level);
  }, [data]);

  const chartData = useMemo(() => {
    if (!data || data.length === 0 || severityList.length === 0) {
      return {
        labels: [],
        datasets: [],
      };
    }

    const labels = data.map((week) => week.weekLabel);

    const datasets = severityList.map((severity, index) => {
      const isTopDataset = index === severityList.length - 1;

      return {
        label: severity.name,
        data: data.map((week) => {
          const count = week.severityCounts.find(
            (sc) => sc.severityId === severity.id
          );
          return count ? count.count : 0;
        }),
        backgroundColor: SEVERITY_COLORS[severity.id] || '#9ca3af',
        // Only apply border radius to the top dataset
        borderRadius: isTopDataset ? 4 : 0,
        // Skip bottom border for all datasets except the bottom one
        borderSkipped: (isTopDataset ? 'bottom' : false) as 'bottom' | false,
        barPercentage: 0.8,
      };
    });

    return {
      labels,
      datasets,
    };
  }, [data, severityList]);

  const options: ChartOptions<'bar'> = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index' as const,
      intersect: false,
    },
    plugins: {
      legend: {
        display: true,
        position: 'bottom' as const,
        labels: {
          usePointStyle: true,
          pointStyle: 'circle',
          padding: 15,
          font: {
            size: 12,
            family: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
          },
          color: '#6b7280',
        },
      },
      tooltip: {
        backgroundColor: '#ffffff',
        titleColor: '#1f2937',
        bodyColor: '#6b7280',
        borderColor: '#e5e7eb',
        borderWidth: 1,
        padding: 12,
        boxPadding: 6,
        usePointStyle: true,
        callbacks: {
          label: function (context) {
            const label = context.dataset.label || '';
            const value = context.parsed.y;
            return `${label}: ${value}`;
          },
        },
      },
    },
    scales: {
      x: {
        stacked: true,
        grid: {
          display: false,
        },
        ticks: {
          color: '#6b7280',
          font: {
            size: 12,
          },
        },
        border: {
          display: false,
        },
      },
      y: {
        stacked: true,
        grid: {
          color: '#f3f4f6',
          drawTicks: false,
        },
        ticks: {
          color: '#6b7280',
          font: {
            size: 12,
          },
          padding: 8,
          stepSize: 1,
          precision: 0,
        },
        border: {
          display: false,
        },
      },
    },
  };

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h2 className="text-base font-semibold text-gray-900 mb-6">
          Incident Trend by Severity (Last 8 Weeks)
        </h2>
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h2 className="text-base font-semibold text-gray-900 mb-6">
          Incident Trend by Severity (Last 8 Weeks)
        </h2>
        <div className="text-red-600">
          <p className="font-medium">Error loading trend data</p>
          <p className="text-sm mt-1">{error.message}</p>
        </div>
      </div>
    );
  }

  if (!chartData.labels || chartData.labels.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h2 className="text-base font-semibold text-gray-900 mb-6">
          Incident Trend by Severity (Last 8 Weeks)
        </h2>
        <p className="text-gray-500 text-center py-8">No incident data available</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-sm p-6">
      <h2 className="text-base font-semibold text-gray-900 mb-6">
        Incident Trend by Severity (Last 8 Weeks)
      </h2>
      <div style={{ height: '280px' }}>
        <Bar options={options} data={chartData} />
      </div>
    </div>
  );
};
