import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../../core/auth/context'
import { AppShell } from '../../core/layout/AppShell'
import type { NavigationItem } from '../../framework/module/types'

export function ProtectedLayout({ navigation }: { navigation: NavigationItem[] }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <div className="grid min-h-screen place-items-center bg-ink text-zinc-500">Loading…</div>
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return <AppShell navigation={navigation} />
}
