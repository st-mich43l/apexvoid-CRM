import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Boxes, BriefcaseBusiness, FileText, Package, Settings2 } from 'lucide-react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '../api/client'
import { AuthContext } from '../auth/context'
import { WorkspaceContext } from '../workspace/context'
import { ThemeContext } from '../theme/context'
import type { NavigationItem } from '../../framework/module/types'
import { AppShell } from './AppShell'

const navigation: NavigationItem[] = [
  { id: 'dashboard', label: 'Dashboard', path: '/', order: 100, icon: Boxes },
  { id: 'crm', label: 'CRM', path: '/crm', order: 350, icon: BriefcaseBusiness, isGroup: true },
  { id: 'crm-leads', label: 'Leads', path: '/crm/leads', order: 351, parentId: 'crm', icon: FileText },
  { id: 'erp', label: 'ERP', path: '/erp', order: 400, icon: Boxes, isGroup: true },
  { id: 'erp-products', label: 'Products & Services', path: '/erp/products', order: 401, parentId: 'erp', icon: Package },
  { id: 'settings', label: 'Settings', path: '/settings/profile', order: 900, icon: Settings2 },
]

function renderShell(start = '/crm/leads', allowed = navigation) {
  vi.spyOn(api, 'health').mockResolvedValue({
    status: 'ok',
    environment: 'test',
  } as Awaited<ReturnType<typeof api.health>>)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <AuthContext.Provider value={{
        user: { id: 'user-1', email: 'user@example.test', display_name: 'Example User', status: 'active', must_change_password: false },
        loading: false, platformPermissions: [],
        login: async () => {}, logout: async () => {}, reload: async () => {}, platformCan: () => true,
      }}>
        <WorkspaceContext.Provider value={{
          workspaces: [], activeWorkspace: null, activeWorkspaceId: 'workspace-1', setup: null,
          workspacePermissions: [], loading: false, selectWorkspace: () => {}, can: () => true, reload: async () => {},
        }}>
          <ThemeContext.Provider value={{
            preference: 'light', resolvedTheme: 'light', setPreference: () => {}, toggle: () => {},
          }}>
            <MemoryRouter initialEntries={[start]}>
              <Routes>
                <Route path="*" element={<AppShell navigation={allowed} applications={[]} />} />
              </Routes>
            </MemoryRouter>
          </ThemeContext.Provider>
        </WorkspaceContext.Provider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  )
}

afterEach(() => {
  cleanup()
  window.localStorage.clear()
  vi.restoreAllMocks()
})

describe('collapsible sidebar navigation', () => {
  it('collapses and expands a section without affecting other sections', () => {
    renderShell()
    const workspace = screen.getByRole('button', { name: 'Workspace section' })
    const platform = screen.getByRole('button', { name: 'Platform section' })
    expect(workspace).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Leads' })).toBeVisible()

    fireEvent.click(workspace)
    expect(workspace).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText('Leads')).not.toBeVisible()
    expect(platform).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Dashboard' })).toBeVisible()

    fireEvent.click(workspace)
    expect(workspace).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Leads' })).toBeVisible()
  })

  it('independently toggles CRM and ERP submenus', () => {
    renderShell()
    const crm = screen.getByRole('button', { name: 'CRM menu' })
    const erp = screen.getByRole('button', { name: 'ERP menu' })
    expect(crm).toHaveAttribute('aria-expanded', 'true')
    expect(erp).toHaveAttribute('aria-expanded', 'true')

    fireEvent.click(crm)
    expect(crm).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText('Leads')).not.toBeVisible()
    expect(erp).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Products & Services' })).toBeVisible()

    fireEvent.click(crm)
    expect(screen.getByRole('link', { name: 'Leads' })).toBeVisible()
  })

  it('remembers collapsed sections and groups after remount', () => {
    renderShell()
    fireEvent.click(screen.getByRole('button', { name: 'Settings section' }))
    fireEvent.click(screen.getByRole('button', { name: 'ERP menu' }))
    expect(JSON.parse(window.localStorage.getItem('apexvoid.sidebar.closed-sections') ?? '{}')).toMatchObject({ settings: true })
    expect(JSON.parse(window.localStorage.getItem('apexvoid.sidebar.closed-groups') ?? '{}')).toMatchObject({ erp: true })

    cleanup()
    renderShell()
    expect(screen.getByRole('button', { name: 'Settings section' })).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByRole('button', { name: 'ERP menu' })).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByRole('link', { name: 'Leads' })).toBeVisible()
  })

  it('opens the sidebar when a compact CRM group is selected', () => {
    renderShell()
    fireEvent.click(screen.getByRole('button', { name: 'CRM menu' }))
    fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }))
    fireEvent.click(screen.getByRole('button', { name: 'CRM menu' }))
    expect(screen.getByRole('button', { name: 'Collapse sidebar' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'CRM menu' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Leads' })).toBeVisible()
  })

  it('reopens a hidden Workspace section from a compact CRM icon', () => {
    renderShell()
    fireEvent.click(screen.getByRole('button', { name: 'Workspace section' }))
    fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }))
    fireEvent.click(screen.getByRole('button', { name: 'CRM menu' }))
    expect(screen.getByRole('button', { name: 'Workspace section' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Leads' })).toBeVisible()
  })

  it('recovers from invalid saved preferences', () => {
    window.localStorage.setItem('apexvoid.sidebar.closed-sections', '{invalid')
    window.localStorage.setItem('apexvoid.sidebar.closed-groups', 'null')
    renderShell()
    expect(screen.getByRole('button', { name: 'Workspace section' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('button', { name: 'CRM menu' })).toHaveAttribute('aria-expanded', 'true')
  })

  it('opens installed applications from the sidebar in a new tab', async () => {
    vi.spyOn(api.framework, 'applications').mockResolvedValue([{
      id: 'cafe',
      display_name: 'ApexVoid Café',
      description: 'Coffee counter and photo booth booking',
      version: '0.1.0',
      api_contract_version: 'v1',
      module_dependencies: [],
      required_permissions: [],
      required_capabilities: [],
      frontend: { entry_route: '/apps/cafe', navigation_id: 'cafe' },
      entry_authorized: true,
      settings_authorized: false,
      deployment: 'external',
      frontend_external: true,
    }])

    renderShell()

    await waitFor(() => expect(screen.getByRole('link', { name: 'ApexVoid Café' })).toBeInTheDocument())
    const link = screen.getByRole('link', { name: 'ApexVoid Café' })
    expect(link).toHaveAttribute('href', '/apps/cafe')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })
})
