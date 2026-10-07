import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'

export type RouteDefinition = { id: string; path?: string; index?: boolean; element: ReactNode }
export type NavigationItem = { id: string; label: string; path: string; order: number; icon: LucideIcon; parentId?: string }

export interface AppModule {
  name: string
  version: string
  dependencies?: string[]
  routes?: RouteDefinition[]
  navigation?: NavigationItem[]
}
