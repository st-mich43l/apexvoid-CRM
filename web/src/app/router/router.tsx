import { createBrowserRouter } from 'react-router-dom'
import { AppShell } from '../../core/layout/AppShell'
import type { ModuleRegistry } from '../../framework/module/registry'

export function createAppRouter(modules: ModuleRegistry) {
  return createBrowserRouter([{ element: <AppShell navigation={modules.navigation().list()} />, children: modules.routes().map((route) => ({ id: route.id, path: route.path, index: route.index, element: route.element })) }])
}
