import React from 'react';

export const BrandLogo: React.FC<{ className?: string }> = ({ className = 'w-6 h-6' }) => {
  return (
    <svg
      className={className}
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Outer Shield */}
      <path
        d="M24 4L8 10V22C8 32 15 40 24 44C33 40 40 32 40 22V10L24 4Z"
        fill="currentColor"
        fillOpacity="0.15"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinejoin="round"
      />
      {/* Portal Archway */}
      <path
        d="M18 32V20C18 16.6863 20.6863 14 24 14C27.3137 14 30 16.6863 30 20V32"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
      {/* Central Portal Pulse */}
      <circle cx="24" cy="22" r="2.5" fill="currentColor" />
    </svg>
  );
};
