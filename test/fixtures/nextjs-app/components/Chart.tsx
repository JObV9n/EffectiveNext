'use client';

import React from 'react';

interface ChartProps {
  data: {
    labels: string[];
    values: number[];
  };
}

export default function Chart({ data }: ChartProps) {
  return (
    <div className="chart-container">
      <canvas id="chart" />
    </div>
  );
}
