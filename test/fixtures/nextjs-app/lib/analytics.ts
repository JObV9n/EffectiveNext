export async function getAnalytics() {
  return {
    totalUsers: 1234,
    activeToday: 567,
    revenue: '$12,345',
  };
}

export async function getChartData() {
  return {
    labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
    values: [100, 200, 150, 300, 250, 400],
  };
}
