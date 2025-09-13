import React from 'react';
import { IncidentStatus, getStatusConfig } from '../../types/incident';

interface StatusBadgeProps {
  status: IncidentStatus | string | null | undefined;
  size?: 'sm' | 'md' | 'lg';
  showIcon?: boolean;
  showLabel?: boolean;
  className?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({
  status,
  size = 'md',
  showIcon = true,
  showLabel = true,
  className = ''
}) => {
  const sizeClasses = {
    sm: 'px-2 py-1 text-xs',
    md: 'px-3 py-1 text-sm',
    lg: 'px-4 py-2 text-base'
  };

  const iconSizes = {
    sm: 'text-xs',
    md: 'text-sm', 
    lg: 'text-base'
  };

  const config = getStatusConfig(status);

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full font-medium ${sizeClasses[size]} ${className}`}
      style={{
        backgroundColor: `${config.color}20`, // 20% opacity for background
        color: config.color,
        border: `1px solid ${config.color}40` // 40% opacity for border
      }}
      title={config.description}
    >
      {showIcon && (
        <span className={iconSizes[size]} role="img" aria-label={config.label}>
          {config.icon}
        </span>
      )}
      {showLabel && (
        <span>{config.label}</span>
      )}
    </span>
  );
};

export default StatusBadge;