import { describe, expect, it } from 'vitest'
import { canAccessNavigation } from './navigation'
import { Boxes } from 'lucide-react'

describe('scope-aware navigation', () => {
  const platformItem = { id: 'platform', label: 'Platform', path: '/platform', order: 1, icon: Boxes, permission: { scope: 'platform' as const, name: 'platform.read' } }
  const workspaceItem = { id: 'workspace', label: 'Workspace', path: '/workspace', order: 2, icon: Boxes, permission: { scope: 'workspace' as const, name: 'workspace.read' } }

  it('checks platform and workspace permissions through their separate evaluators', () => {
    expect(canAccessNavigation(platformItem, permission => permission === 'platform.read', () => false)).toBe(true)
    expect(canAccessNavigation(workspaceItem, () => false, permission => permission === 'workspace.read')).toBe(true)
    expect(canAccessNavigation(platformItem, () => false, () => true)).toBe(false)
  })
})
