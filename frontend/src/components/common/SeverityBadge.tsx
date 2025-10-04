import React from 'react';
import { getSeverityStyle } from '../../types/incident';

interface SeverityBadgeProps {
  severityLevel: number;
  severityName: string;
  size?: 'sm' | 'md';
}

/**
 * SeverityBadge component displays incident severity with appropriate styling
 * Extracts severity badge rendering logic to improve reusability and maintainability
 */
const SeverityBadge: React.FC<SeverityBadgeProps> = ({
  severityLevel,
  severityName,
  size = 'md'
}) => {
  const severityStyle = getSeverityStyle(severityLevel);

  const sizeClasses = size === 'sm'
    ? 'text-xs px-2 py-0.5'
    : 'text-sm px-2 py-1';

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full font-medium ${sizeClasses}`}
      style={{
        color: severityStyle.color,
        backgroundColor: severityStyle.backgroundColor,
      }}
    >
      <span>{severityStyle.icon}</span>
      <span>{severityName}</span>
    </span>
  );
};

export default SeverityBadge;
