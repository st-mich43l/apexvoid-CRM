import { Shield, Users } from 'lucide-react'
import { UsersPage } from './pages/UsersPage'
import { RolesPage } from './pages/RolesPage'
import type { AppModule } from '../../framework/module/types'

export const adminModule: AppModule = {
  name: 'admin', version: '1.0.0', dependencies: ['core'],
  routes: [{ id: 'admin-users', path: '/admin/users', element: <UsersPage /> }, { id: 'admin-roles', path: '/admin/roles', element: <RolesPage /> }],
  navigation: [{ id: 'admin-users', label: 'Users', path: '/admin/users', order: 30, icon: Users, permission: { scope: 'platform', name: 'users.user.read' } }, { id: 'admin-roles', label: 'Platform Roles', path: '/admin/roles', order: 31, icon: Shield, permission: { scope: 'platform', name: 'access.role.read' } }],
}
