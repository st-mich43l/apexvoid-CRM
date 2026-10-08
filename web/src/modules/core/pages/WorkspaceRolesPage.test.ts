import { describe, expect, it } from 'vitest'
import { groupWorkspacePermissions } from './workspaceRolePermissions'

describe('workspace role permissions', () => {
  it('groups only workspace-scoped permissions for role editing', () => {
    const groups = groupWorkspacePermissions([
      { name: 'organization.organization.read', display_name: 'Read organization', description: '', module: 'organization', scope: 'workspace' },
      { name: 'workspace.member.read', display_name: 'Read members', description: '', module: 'organization', scope: 'workspace' },
      { name: 'users.user.read', display_name: 'Read users', description: '', module: 'users', scope: 'platform' },
    ])
    expect(groups.Organization).toHaveLength(1)
    expect(groups.Workspace).toHaveLength(1)
    expect(Object.values(groups).flat()).toHaveLength(2)
  })
})
