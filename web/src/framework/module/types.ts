import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'

export type RouteDefinition = { id: string; path?: string; index?: boolean; element: ReactNode }
export type NavigationPermission = { scope: 'platform' | 'workspace'; name: string }
export type NavigationItem = { id: string; label: string; path: string; order: number; icon: LucideIcon; parentId?: string; isGroup?: boolean; openInNewTab?: boolean; permission?: NavigationPermission; permissionsAny?: NavigationPermission[]; requiredPermission?: string }
export type FrontendApplication = { id: string; entryRoute: string; navigationID: string; apiContractVersion: string }

export interface AppModule {
  name: string
  version: string
  dependencies?: string[]
  routes?: RouteDefinition[]
  navigation?: NavigationItem[]
  application?: FrontendApplication
}
