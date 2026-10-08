import type { NavigationItem } from '../../framework/module/types'

export function canAccessNavigation(item: NavigationItem, platformCan: (permission: string) => boolean, workspaceCan: (permission: string) => boolean) {
  if (item.permission) return item.permission.scope === 'platform' ? platformCan(item.permission.name) : workspaceCan(item.permission.name)
  return !item.requiredPermission || platformCan(item.requiredPermission)
}
