import React from 'react';

interface TestBadgeProps {
  size?: 'sm' | 'md' | 'lg';
}

export const TestBadge: React.FC<TestBadgeProps> = ({ size = 'md' }) => {
  const sizeClasses = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm',
    lg: 'px-3 py-1.5 text-base',
  };

  return (
    <span
      className={`inline-flex items-center gap-1 ${sizeClasses[size]} font-medium text-yellow-800 bg-yellow-100 border border-yellow-300 rounded whitespace-nowrap`}
    >
      🧪 TEST
    </span>
  );
};

export default TestBadge;
