import { ProtectedRoute } from '@/components/protected-route';
import { DashboardShell } from '@/components/shell';
import { DashboardClient } from '@/components/dashboard-client';

export default function DashboardPage() {
  return (
    <ProtectedRoute>
      <DashboardShell>
        <DashboardClient />
      </DashboardShell>
    </ProtectedRoute>
  );
}
