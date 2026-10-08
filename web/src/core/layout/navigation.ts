import type { NavigationItem } from '../../framework/module/types'

export type NavigationSection = { id: string; label: string; items: NavigationItem[] }

export function canAccessNavigation(item: NavigationItem, platformCan: (permission: string) => boolean, workspaceCan: (permission: string) => boolean) {
  if (item.permission) return item.permission.scope === 'platform' ? platformCan(item.permission.name) : workspaceCan(item.permission.name)
  return !item.requiredPermission || platformCan(item.requiredPermission)
}

const sectionOrder = [
  { id: 'overview', label: 'Overview' },
  { id: 'workspace', label: 'Workspace' },
  { id: 'administration', label: 'Administration' },
  { id: 'settings', label: 'Settings' },
]

function sectionFor(item: NavigationItem): string {
  if (item.path.startsWith('/contacts')) return item.path.includes('/settings') ? 'settings' : 'workspace'
  if (item.path.startsWith('/admin') || item.path.startsWith('/framework')) return 'administration'
  if (item.path.startsWith('/settings')) return 'settings'
  return 'overview'
}

export function groupNavigation(items: NavigationItem[]): NavigationSection[] {
  return sectionOrder.map(section => ({ id: section.id, label: section.label, items: items.filter(item => sectionFor(item) === section.id).sort((a, b) => a.order - b.order) })).filter(section => section.items.length > 0)
}
