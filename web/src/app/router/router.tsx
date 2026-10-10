import { createBrowserRouter } from 'react-router-dom'
import { LoginPage } from '../../modules/core/pages/LoginPage'
import { ExternalAuthContinuePage } from '../../modules/core/pages/ExternalAuthContinuePage'
import { ExternalAccessDeniedPage } from '../../modules/core/pages/ExternalAccessDeniedPage'
import { ChangePasswordPage } from '../../modules/core/pages/ChangePasswordPage'
import { SetupPage } from '../../modules/core/pages/SetupPage'
import { ProtectedLayout } from './ProtectedLayout'
import type { ModuleRegistry } from '../../framework/module/registry'

export function createAppRouter(modules: ModuleRegistry) {
  return createBrowserRouter([{ path: '/login', element: <LoginPage /> }, { path: '/auth/continue', element: <ExternalAuthContinuePage /> }, { path: '/auth/app-access-denied', element: <ExternalAccessDeniedPage /> }, { element: <ProtectedLayout navigation={modules.navigation().list()} applications={modules.applications()} />, children: [{ path: '/change-password', element: <ChangePasswordPage /> }, { path: '/setup', element: <SetupPage /> }, ...modules.routes().map((route) => ({ id: route.id, path: route.path, index: route.index, element: route.element }))] }])
}
