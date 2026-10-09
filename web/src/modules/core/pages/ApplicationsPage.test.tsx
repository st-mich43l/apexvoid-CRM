import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Outlet, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../../core/api/client'
import type { ApplicationMetadata } from '../../../framework/metadata/types'
import type { FrontendApplication } from '../../../framework/module/types'
import { ApplicationsPage } from './ApplicationsPage'
import { AuthContext } from '../../../core/auth/context'
import { WorkspaceContext } from '../../../core/workspace/context'

const compiled: FrontendApplication[] = [{ id: 'crm', entryRoute: '/crm', navigationID: 'crm', apiContractVersion: 'v1' }]
const registered = (overrides: Partial<ApplicationMetadata> = {}): ApplicationMetadata => ({
  id: 'crm', display_name: 'CRM', description: 'Sales workspace', version: '1.0.0', api_contract_version: 'v1', module_dependencies: ['crm'], required_permissions: ['crm.lead.read'], required_capabilities: [], frontend: { entry_route: '/crm', navigation_id: 'crm' }, entry_authorized: true, settings_authorized: false, ...overrides,
})

function renderPage(canManage = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const auth = { user: { id: 'user-a', email: 'a@example.test', display_name: 'A', status: 'active', must_change_password: false }, loading: false, platformPermissions: [], platformCan: () => canManage, login: async () => {}, logout: async () => {}, reload: async () => {} }
  const workspace = { workspaces: [], activeWorkspace: null, activeWorkspaceId: 'workspace-a', setup: null, workspacePermissions: [], loading: false, selectWorkspace: () => {}, can: () => false, reload: async () => {} }
  return render(<QueryClientProvider client={client}><AuthContext.Provider value={auth}><WorkspaceContext.Provider value={workspace}><MemoryRouter><Routes><Route element={<Outlet context={{ applications: compiled }} />}><Route index element={<ApplicationsPage />} /></Route></Routes></MemoryRouter></WorkspaceContext.Provider></AuthContext.Provider></QueryClientProvider>)
}

afterEach(() => vi.restoreAllMocks())

describe('ApplicationsPage', () => {
  it('shows an open action only for a compatible application authorized for the workspace', async () => {
    vi.spyOn(api.framework, 'applications').mockResolvedValue([registered({ settings: { route: '/crm/settings/pipelines' }, settings_authorized: true })])
    renderPage()
    expect(await screen.findByText('Available')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /open application/i })).toHaveAttribute('href', '/crm')
    expect(screen.getByRole('link', { name: /settings/i })).toHaveAttribute('href', '/crm/settings/pipelines')
  })

  it('explains unavailable access without rendering privileged actions', async () => {
    vi.spyOn(api.framework, 'applications').mockResolvedValue([registered({ entry_authorized: false, settings_authorized: false })])
    renderPage()
    expect(await screen.findByText('Not authorized')).toBeInTheDocument()
    expect(screen.getByText(/do not have the required permission/i)).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /open application/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /settings/i })).not.toBeInTheDocument()
  })

  it('renders a clear contract warning instead of an action for an incompatible build', async () => {
    vi.spyOn(api.framework, 'applications').mockResolvedValue([registered({ api_contract_version: 'v2' })])
    renderPage()
    expect(await screen.findByText('Contract issue')).toBeInTheDocument()
    expect(screen.getByText(/API contract version does not match/i)).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /open application/i })).not.toBeInTheDocument()
  })

  it('opens an accessible registration popup and discovers app-owned version and API contract', async () => {
    vi.spyOn(api.framework, 'applications').mockResolvedValue([registered()])
    vi.spyOn(api.integrations, 'external').mockResolvedValue([])
    const discover = vi.spyOn(api.integrations, 'discover').mockResolvedValue({
      id: 'install-1', application_id: 'cafe', service_url: 'http://apexvoid-cafe:8090',
      manifest: {
        manifest_version: 'v1',
        application: { id: 'cafe', display_name: 'ApexVoid Café', description: 'Café operations', version: '2.7.3', api_contract_version: 'v1' },
        service: { identity: 'cafe-service', health_path: '/health', enrollment_path: '/.well-known/apexvoid/enroll', frontend_route: '/apps/cafe', api_route: '/api' },
        database: { name: 'apexvoid_cafe', schema: 'cafe', role: 'apexvoid_cafe', migration_bundle_version: '2.7.3' },
        permissions: [{ name: 'cafe.order.read', display_name: 'Read orders', description: '', scope: 'workspace' }],
        access: { match: 'all', permissions: ['cafe.order.read'] }, migrations: [],
      },
      status: 'pending_approval', expires_at: '', last_step: 'discovered', selected_workspace_ids: [], created_at: '', updated_at: '',
    })
    renderPage(true)
    const launch = await screen.findByRole('button', { name: /register application/i })
    launch.focus()
    fireEvent.click(launch)
    expect(screen.getByRole('dialog', { name: /register an application/i })).toBeInTheDocument()
    expect(screen.getByLabelText('Application service URL')).toBeInTheDocument()
    expect(screen.queryByLabelText('Application version')).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Application service URL'), { target: { value: 'http://apexvoid-cafe:8090' } })
    fireEvent.change(screen.getByLabelText('One-time enrollment code'), { target: { value: 'correct-32-character-secret-for-cafe-bootstrap' } })
    fireEvent.click(screen.getByRole('button', { name: 'Discover application' }))
    await waitFor(() => expect(discover).toHaveBeenCalled())
    expect(await screen.findByText('2.7.3')).toBeInTheDocument()
    expect(screen.getByText('v1')).toBeInTheDocument()
    expect(screen.getByText('Declared by the application manifest')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /approve & activate/i })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Close application dialog' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(launch).toHaveFocus()
  })
})
