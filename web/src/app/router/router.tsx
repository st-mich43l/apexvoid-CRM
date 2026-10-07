import { createBrowserRouter } from 'react-router-dom'
import { LoginPage } from '../../modules/core/pages/LoginPage'
import { ProtectedLayout } from './ProtectedLayout'
import type { ModuleRegistry } from '../../framework/module/registry'

export function createAppRouter(modules: ModuleRegistry) {
  return createBrowserRouter([{ path: '/login', element: <LoginPage /> }, { element: <ProtectedLayout navigation={modules.navigation().list()} />, children: modules.routes().map((route) => ({ id: route.id, path: route.path, index: route.index, element: route.element })) }])
}
