import { getChartData } from '@/lib/analytics';
import Chart from '@/components/Chart';

export default async function AnalyticsPage() {
  const data = await getChartData();

  return (
    <div>
      <h1>Analytics</h1>
      <Chart data={data} />
    </div>
  );
}
