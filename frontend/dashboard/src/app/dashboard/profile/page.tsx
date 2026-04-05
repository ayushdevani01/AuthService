import { ProtectedRoute } from '@/components/protected-route';
import { DashboardShell } from '@/components/shell';
import { ProfileClient } from '@/components/profile-client';

export default function ProfilePage() {
  return (
    <ProtectedRoute>
      <DashboardShell>
        <ProfileClient />
      </DashboardShell>
    </ProtectedRoute>
  );
}
