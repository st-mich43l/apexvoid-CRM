import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
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

  it('gives platform administrators a structured external application registration flow', async () => {
    vi.spyOn(api.framework, 'applications').mockResolvedValue([registered()])
    vi.spyOn(api.integrations, 'external').mockResolvedValue([])
    renderPage(true)
    fireEvent.click(await screen.findByRole('button', { name: /register application/i }))
    expect(screen.getByText(/register trusted external application/i)).toBeInTheDocument()
    expect(screen.getByLabelText('Application ID')).toBeInTheDocument()
    expect(screen.getByLabelText('Service endpoint')).toBeInTheDocument()
    expect(screen.getByText('Application permissions')).toBeInTheDocument()
  })
})
