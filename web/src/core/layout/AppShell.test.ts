import { describe, expect, it } from 'vitest'
import { canAccessNavigation, groupNavigation } from './navigation'
import { Boxes } from 'lucide-react'

describe('scope-aware navigation', () => {
  const platformItem = { id: 'platform', label: 'Platform', path: '/platform', order: 1, icon: Boxes, permission: { scope: 'platform' as const, name: 'platform.read' } }
  const workspaceItem = { id: 'workspace', label: 'Workspace', path: '/workspace', order: 2, icon: Boxes, permission: { scope: 'workspace' as const, name: 'workspace.read' } }

  it('checks platform and workspace permissions through their separate evaluators', () => {
    expect(canAccessNavigation(platformItem, permission => permission === 'platform.read', () => false)).toBe(true)
    expect(canAccessNavigation(workspaceItem, () => false, permission => permission === 'workspace.read')).toBe(true)
    expect(canAccessNavigation(platformItem, () => false, () => true)).toBe(false)
  })

  it('groups module navigation into predictable product sections', () => {
    const sections = groupNavigation([
      { id: 'dashboard', label: 'Dashboard', path: '/', order: 100, icon: Boxes },
      { id: 'people', label: 'People', path: '/contacts/people', order: 300, icon: Boxes },
      { id: 'crm', label: 'CRM', path: '/crm', order: 350, icon: Boxes, isGroup: true },
      { id: 'erp', label: 'ERP', path: '/erp', order: 400, icon: Boxes, isGroup: true },
      { id: 'products', label: 'Products', path: '/erp/products', order: 401, parentId: 'erp', icon: Boxes },
      { id: 'roles', label: 'Roles', path: '/settings/roles', order: 900, icon: Boxes },
    ])
    expect(sections.map(section => section.id)).toEqual(['platform', 'workspace', 'settings'])
    expect(sections[1].items.map(item => item.id)).toEqual(['people', 'crm', 'erp', 'products'])
  })
})
