import { Boxes } from 'lucide-react'
import { describe, expect, it } from 'vitest'
import { ModuleRegistry } from './registry'

const application = { id: 'fixture', entryRoute: '/fixture', navigationID: 'fixture-home', apiContractVersion: 'v1' }
const module = { name: 'fixture', version: '1.0.0', application, routes: [{ id: 'fixture-route', path: '/fixture', element: null }], navigation: [{ id: 'fixture-home', label: 'Fixture', path: '/fixture', order: 1, icon: Boxes }] }

describe('compiled application contracts', () => {
  it('reports backend/frontend contracts and makes mismatches understandable', () => {
    const registry = new ModuleRegistry([module])
    const ready = registry.applicationContracts([{ id: 'fixture', display_name: 'Fixture', description: 'Fixture', version: '1.0.0', api_contract_version: 'v1', module_dependencies: ['fixture'], required_permissions: [], required_capabilities: [], frontend: { entry_route: '/fixture', navigation_id: 'fixture-home' }, entry_authorized: true, settings_authorized: false }])
    expect(ready[0].status).toBe('compatible')
    const mismatched = registry.applicationContracts([{ id: 'fixture', display_name: 'Fixture', description: 'Fixture', version: '1.0.0', api_contract_version: 'v1', module_dependencies: ['fixture'], required_permissions: [], required_capabilities: [], frontend: { entry_route: '/other', navigation_id: 'fixture-home' }, entry_authorized: true, settings_authorized: false }])
    expect(mismatched[0]).toMatchObject({ status: 'route-mismatch', message: expect.stringContaining('entry route') })
    const incompatible = registry.applicationContracts([{ id: 'fixture', display_name: 'Fixture', description: 'Fixture', version: '1.0.0', api_contract_version: 'v2', module_dependencies: ['fixture'], required_permissions: [], required_capabilities: [], frontend: { entry_route: '/fixture', navigation_id: 'fixture-home' }, entry_authorized: true, settings_authorized: false }])
    expect(incompatible[0]).toMatchObject({ status: 'api-version-mismatch', message: expect.stringContaining('API contract version') })
  })

  it('rejects a compiled application without its route or navigation identity', () => {
    const registry = new ModuleRegistry([{ ...module, routes: [] }])
    expect(() => registry.applications()).toThrow('entry route')
  })
})
