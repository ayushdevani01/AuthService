import { ProtectedRoute } from '@/components/protected-route';
import { DashboardShell } from '@/components/shell';
import { AppDetailClient } from '@/components/app-detail-client';

export default function AppDetailPage({ params }: { params: { id: string } }) {
  return (
    <ProtectedRoute>
      <DashboardShell>
        <AppDetailClient appId={params.id} />
      </DashboardShell>
    </ProtectedRoute>
  );
}
