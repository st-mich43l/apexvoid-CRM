import type { PermissionMetadata } from '../../../framework/metadata/types'

export function groupWorkspacePermissions(permissions: PermissionMetadata[]) {
  return permissions.filter(item => item.scope === 'workspace').reduce<Record<string, PermissionMetadata[]>>((groups, item) => {
    const group = item.name.startsWith('workspace.') ? 'Workspace' : item.module === 'organization' ? 'Organization' : item.module
    groups[group] = [...(groups[group] ?? []), item]
    return groups
  }, {})
}
