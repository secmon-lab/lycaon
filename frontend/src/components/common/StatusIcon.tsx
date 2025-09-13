import React from 'react';
import { IncidentStatus, getStatusIcon } from '../../types/incident';

interface StatusIconProps {
  status: IncidentStatus | string | null | undefined;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export const StatusIcon: React.FC<StatusIconProps> = ({
  status,
  size = 'md',
  className = ''
}) => {
  const icon = getStatusIcon(status);
  
  const sizeClasses = {
    sm: 'text-sm',
    md: 'text-base',
    lg: 'text-lg'
  };

  return (
    <span 
      className={`${sizeClasses[size]} ${className}`}
      role="img" 
      aria-label={status || 'Not Set'}
    >
      {icon}
    </span>
  );
};

export default StatusIcon;