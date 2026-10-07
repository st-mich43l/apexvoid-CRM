import { Boxes, Building2, LayoutDashboard, Settings, UsersRound } from 'lucide-react'
import type { AppModule } from '../../framework/module/types'
import { DashboardPage } from './pages/DashboardPage'
import { FrameworkPage } from './pages/FrameworkPage'
import { SettingsPage } from './pages/SettingsPage'
import { OrganizationPage } from './pages/OrganizationPage'
import { MembersPage } from './pages/MembersPage'
import { NotFoundPage } from './pages/NotFoundPage'

export const coreModule: AppModule = {
  name: 'core',
  version: '1.0.0',
  routes: [{ id: 'dashboard', index: true, element: <DashboardPage /> }, { id: 'framework', path: 'framework', element: <FrameworkPage /> }, { id: 'settings', path: 'settings', element: <SettingsPage /> }, { id: 'organization', path: 'settings/organization', element: <OrganizationPage /> }, { id: 'members', path: 'settings/members', element: <MembersPage /> }, { id: 'not-found', path: '*', element: <NotFoundPage /> }],
  navigation: [{ id: 'dashboard', label: 'Dashboard', path: '/', order: 100, icon: LayoutDashboard }, { id: 'framework', label: 'Framework', path: '/framework', order: 200, icon: Boxes }, { id: 'settings', label: 'Settings', path: '/settings', order: 900, icon: Settings }, { id: 'organization', label: 'Organization', path: '/settings/organization', order: 901, icon: Building2, requiredPermission: 'organization.organization.read' }, { id: 'members', label: 'Members', path: '/settings/members', order: 902, icon: UsersRound, requiredPermission: 'workspace.member.read' }],
}
