import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '../components/ui/Button';
import { 
  AlertCircle, 
  TrendingUp, 
  Users, 
  Clock,
  ArrowRight,
  Activity,
  CheckCircle,
  XCircle,
  AlertTriangle,
} from 'lucide-react';

interface DashboardProps {
  user: {
    id: string;
    name: string;
    email: string;
    slack_user_id: string;
  };
  setUser: (user: any) => void;
}

const Dashboard: React.FC<DashboardProps> = ({ user }) => {
  const navigate = useNavigate();

  const stats = [
    {
      title: 'Total Incidents',
      value: '142',
      change: '+12%',
      trend: 'up',
      icon: AlertCircle,
      color: 'blue',
    },
    {
      title: 'Active Incidents',
      value: '8',
      change: '-5%',
      trend: 'down',
      icon: Activity,
      color: 'orange',
    },
    {
      title: 'Resolved Today',
      value: '23',
      change: '+18%',
      trend: 'up',
      icon: CheckCircle,
      color: 'green',
    },
    {
      title: 'Avg Resolution Time',
      value: '2.4h',
      change: '-12%',
      trend: 'down',
      icon: Clock,
      color: 'purple',
    },
  ];

  const recentIncidents = [
    {
      id: '1',
      title: 'Database connection timeout',
      status: 'open',
      priority: 'high',
      time: '5 minutes ago',
    },
    {
      id: '2',
      title: 'API rate limit exceeded',
      status: 'investigating',
      priority: 'medium',
      time: '1 hour ago',
    },
    {
      id: '3',
      title: 'Deployment pipeline failure',
      status: 'resolved',
      priority: 'low',
      time: '3 hours ago',
    },
  ];

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'open':
        return <XCircle className="h-4 w-4 text-red-500" />;
      case 'investigating':
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      case 'resolved':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      default:
        return <AlertCircle className="h-4 w-4 text-gray-500" />;
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'bg-red-100 text-red-700 border-red-200';
      case 'medium':
        return 'bg-yellow-100 text-yellow-700 border-yellow-200';
      case 'low':
        return 'bg-green-100 text-green-700 border-green-200';
      default:
        return 'bg-gray-100 text-gray-700 border-gray-200';
    }
  };

  return (
    <div className="space-y-6">
      {/* Welcome Section */}
      <div className="rounded-lg bg-gradient-to-r from-blue-600 to-purple-600 p-8 text-white">
        <h1 className="text-3xl font-bold">Welcome back, {user.name}!</h1>
        <p className="mt-2 text-blue-100">
          Here's what's happening with your incidents today.
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          const colorClasses = {
            blue: 'bg-blue-100 text-blue-600',
            orange: 'bg-orange-100 text-orange-600',
            green: 'bg-green-100 text-green-600',
            purple: 'bg-purple-100 text-purple-600',
          };

          return (
            <div key={stat.title} className="rounded-lg border border-slate-200 bg-white p-6">
              <div className="flex items-center justify-between">
                <div className={`rounded-lg p-2 ${colorClasses[stat.color as keyof typeof colorClasses]}`}>
                  <Icon className="h-5 w-5" />
                </div>
                <span className={`text-sm font-medium ${
                  stat.trend === 'up' ? 'text-green-600' : 'text-red-600'
                }`}>
                  {stat.change}
                </span>
              </div>
              <div className="mt-4">
                <p className="text-2xl font-semibold text-slate-900">{stat.value}</p>
                <p className="text-sm text-slate-500">{stat.title}</p>
              </div>
            </div>
          );
        })}
      </div>

      {/* Recent Incidents and Activity */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent Incidents */}
        <div className="rounded-lg border border-slate-200 bg-white">
          <div className="flex items-center justify-between border-b border-slate-200 p-4">
            <h2 className="font-semibold text-slate-900">Recent Incidents</h2>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate('/incidents')}
              className="gap-2"
            >
              View all
              <ArrowRight className="h-4 w-4" />
            </Button>
          </div>
          <div className="divide-y divide-slate-200">
            {recentIncidents.map((incident) => (
              <div key={incident.id} className="flex items-center justify-between p-4 hover:bg-slate-50">
                <div className="flex items-start gap-3">
                  {getStatusIcon(incident.status)}
                  <div>
                    <p className="font-medium text-slate-900">{incident.title}</p>
                    <div className="mt-1 flex items-center gap-2">
                      <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${getPriorityColor(incident.priority)}`}>
                        {incident.priority}
                      </span>
                      <span className="text-xs text-slate-500">{incident.time}</span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Activity Feed */}
        <div className="rounded-lg border border-slate-200 bg-white">
          <div className="border-b border-slate-200 p-4">
            <h2 className="font-semibold text-slate-900">Recent Activity</h2>
          </div>
          <div className="p-4">
            <div className="space-y-4">
              <div className="flex gap-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-100">
                  <Users className="h-4 w-4 text-blue-600" />
                </div>
                <div className="flex-1">
                  <p className="text-sm text-slate-900">
                    <span className="font-medium">Sarah Chen</span> assigned incident #142 to <span className="font-medium">John Doe</span>
                  </p>
                  <p className="text-xs text-slate-500">2 minutes ago</p>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-green-100">
                  <CheckCircle className="h-4 w-4 text-green-600" />
                </div>
                <div className="flex-1">
                  <p className="text-sm text-slate-900">
                    <span className="font-medium">Mike Johnson</span> resolved incident #139
                  </p>
                  <p className="text-xs text-slate-500">15 minutes ago</p>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-yellow-100">
                  <AlertTriangle className="h-4 w-4 text-yellow-600" />
                </div>
                <div className="flex-1">
                  <p className="text-sm text-slate-900">
                    New incident created: <span className="font-medium">Memory leak in production</span>
                  </p>
                  <p className="text-xs text-slate-500">1 hour ago</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="rounded-lg border border-slate-200 bg-white p-6">
        <h2 className="mb-4 font-semibold text-slate-900">Quick Actions</h2>
        <div className="flex flex-wrap gap-3">
          <Button size="sm" className="gap-2">
            <AlertCircle className="h-4 w-4" />
            Create Incident
          </Button>
          <Button size="sm" variant="outline" className="gap-2">
            <TrendingUp className="h-4 w-4" />
            View Analytics
          </Button>
          <Button size="sm" variant="outline" className="gap-2">
            <Users className="h-4 w-4" />
            Team Overview
          </Button>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;