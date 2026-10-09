import { createBrowserRouter } from 'react-router-dom'
import { LoginPage } from '../../modules/core/pages/LoginPage'
import { ChangePasswordPage } from '../../modules/core/pages/ChangePasswordPage'
import { SetupPage } from '../../modules/core/pages/SetupPage'
import { ProtectedLayout } from './ProtectedLayout'
import type { ModuleRegistry } from '../../framework/module/registry'

export function createAppRouter(modules: ModuleRegistry) {
  return createBrowserRouter([{ path: '/login', element: <LoginPage /> }, { element: <ProtectedLayout navigation={modules.navigation().list()} applications={modules.applications()} />, children: [{ path: '/change-password', element: <ChangePasswordPage /> }, { path: '/setup', element: <SetupPage /> }, ...modules.routes().map((route) => ({ id: route.id, path: route.path, index: route.index, element: route.element }))] }])
}
