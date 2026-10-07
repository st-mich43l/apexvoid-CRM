import { Boxes, LayoutDashboard, Settings } from 'lucide-react'
import type { AppModule } from '../../framework/module/types'
import { DashboardPage } from './pages/DashboardPage'
import { FrameworkPage } from './pages/FrameworkPage'
import { SettingsPage } from './pages/SettingsPage'
import { NotFoundPage } from './pages/NotFoundPage'

export const coreModule: AppModule = {
  name: 'core',
  version: '1.0.0',
  routes: [{ id: 'dashboard', index: true, element: <DashboardPage /> }, { id: 'framework', path: 'framework', element: <FrameworkPage /> }, { id: 'settings', path: 'settings', element: <SettingsPage /> }, { id: 'not-found', path: '*', element: <NotFoundPage /> }],
  navigation: [{ id: 'dashboard', label: 'Dashboard', path: '/', order: 100, icon: LayoutDashboard }, { id: 'framework', label: 'Framework', path: '/framework', order: 200, icon: Boxes }, { id: 'settings', label: 'Settings', path: '/settings', order: 900, icon: Settings }],
}
