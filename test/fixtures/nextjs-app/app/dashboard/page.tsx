import { getAnalytics } from '@/lib/analytics';
import StatsCard from '@/components/StatsCard';

export default async function DashboardPage() {
  const analytics = await getAnalytics();

  return (
    <div>
      <h1>Dashboard</h1>
      <div className="stats-grid">
        <StatsCard title="Total Users" value={analytics.totalUsers} />
        <StatsCard title="Active Today" value={analytics.activeToday} />
        <StatsCard title="Revenue" value={analytics.revenue} />
      </div>
    </div>
  );
}
