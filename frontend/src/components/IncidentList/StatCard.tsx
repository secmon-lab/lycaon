import React from 'react';
import { LucideIcon } from 'lucide-react';

interface StatCardProps {
  icon: LucideIcon;
  iconColor: string;
  iconBgColor: string;
  mainValue: string | number;
  label: string;
  subInfo?: string | React.ReactNode;
}

export const StatCard: React.FC<StatCardProps> = ({
  icon: Icon,
  iconColor,
  iconBgColor,
  mainValue,
  label,
  subInfo,
}) => {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <div className="flex items-center gap-3">
        <div className={`rounded-lg ${iconBgColor} p-2`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
        <div className="flex-1">
          <p className="text-2xl font-semibold text-slate-900">{mainValue}</p>
          <p className="text-sm text-slate-500">{label}</p>
          {subInfo && (
            <p className="text-xs text-slate-400 mt-1">{subInfo}</p>
          )}
        </div>
      </div>
    </div>
  );
};
