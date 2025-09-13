import React from 'react';

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
  return (
    <div className="space-y-6">
      {/* Welcome Section */}
      <div className="rounded-lg bg-gradient-to-r from-blue-600 to-purple-600 p-8 text-white">
        <h1 className="text-3xl font-bold">Welcome back, {user.name}!</h1>
        <p className="mt-2 text-blue-100">
          Dashboard is under construction.
        </p>
      </div>
    </div>
  );
};

export default Dashboard;