import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../../core/auth/context'
import { AppShell } from '../../core/layout/AppShell'
import { useWorkspace } from '../../core/workspace/context'
import { WorkspaceProvider } from '../../core/workspace/provider'
import type { FrontendApplication, NavigationItem } from '../../framework/module/types'

export function ProtectedLayout({ navigation, applications }: { navigation: NavigationItem[]; applications: FrontendApplication[] }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <div className="grid min-h-screen place-items-center bg-background text-muted-foreground">Loading…</div>
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  if (user.must_change_password && location.pathname !== '/change-password') return <Navigate to="/change-password" replace />
  if (user.must_change_password && location.pathname === '/change-password') return <Outlet />
  return <WorkspaceProvider><WorkspaceGate navigation={navigation} applications={applications} /></WorkspaceProvider>
}

function WorkspaceGate({ navigation, applications }: { navigation: NavigationItem[]; applications: FrontendApplication[] }) {
  const location = useLocation()
  const { setup, loading, activeWorkspace } = useWorkspace()
  if (loading) return <div className="grid min-h-screen place-items-center bg-background text-muted-foreground">Loading workspace…</div>
  if (setup?.required && location.pathname !== '/setup') return <Navigate to="/setup" replace />
  if (!setup?.required && location.pathname === '/setup') return <Navigate to="/" replace />
  if (!setup?.required && !activeWorkspace) return <div className="grid min-h-screen place-items-center bg-background text-muted-foreground">No accessible workspace</div>
  return <AppShell navigation={navigation} applications={applications} />
}
