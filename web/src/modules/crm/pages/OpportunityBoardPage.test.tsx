import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, type OpportunityBoard } from '../../../core/api/client'
import { AuthContext } from '../../../core/auth/context'
import { WorkspaceContext } from '../../../core/workspace/context'
import { OpportunityBoardPage } from './OpportunityBoardPage'

const user = { id: 'user-1', email: 'admin@example.com', username: 'admin', display_name: 'Admin', status: 'active', must_change_password: false }
const stageOne = { id: 'stage-1', pipeline_id: 'pipeline-1', key: 'new', name: 'New', description: '', color: '#7c3aed', category: 'open' as const, probability: 10, position: 0, active: true, version: 1 }
const stageTwo = { ...stageOne, id: 'stage-2', key: 'qualified', name: 'Qualified', position: 1 }
const item = { id: 'opportunity-1', title: 'Acme expansion', description: '', pipeline_id: 'pipeline-1', stage_id: stageOne.id, expected_revenue: '1250.00', currency: 'USD', outcome: 'open' as const, custom_values: {}, version: 3, created_at: '2026-10-08T00:00:00Z' }
const board: OpportunityBoard = { pipeline: { id: 'pipeline-1', workspace_id: 'workspace-1', name: 'Sales', slug: 'sales', description: '', color: '#7c3aed', status: 'active', default: true, version: 1 }, stages: [{ stage: stageOne, count: 1, totals_by_currency: { USD: '1250.00' }, items: [item], next_cursor: null }, { stage: stageTwo, count: 0, totals_by_currency: {}, items: [], next_cursor: null }], filter: { search: '', outcome: 'open', limit: 20 } }

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><AuthContext.Provider value={{ user, loading: false, platformPermissions: [], login: vi.fn(), logout: vi.fn(), platformCan: () => true, reload: vi.fn() }}><WorkspaceContext.Provider value={{ workspaces: [], activeWorkspace: null, activeWorkspaceId: 'workspace-1', setup: null, workspacePermissions: ['crm.opportunity.read', 'crm.opportunity.transition'], loading: false, selectWorkspace: vi.fn(), can: permission => ['crm.opportunity.read', 'crm.opportunity.transition'].includes(permission), reload: vi.fn() }}><MemoryRouter initialEntries={['/crm/opportunities?pipeline_id=pipeline-1&view=kanban']}><OpportunityBoardPage/></MemoryRouter></WorkspaceContext.Provider></AuthContext.Provider></QueryClientProvider>)
}

describe('OpportunityBoardPage', () => {
  afterEach(() => vi.restoreAllMocks())

  it('mounts the DB-backed board with counts and currency totals', async () => {
    vi.spyOn(api.crm, 'pipelines').mockResolvedValue([board.pipeline])
    vi.spyOn(api.crm, 'stages').mockResolvedValue([stageOne, stageTwo])
    vi.spyOn(api.crm, 'opportunityBoard').mockResolvedValue(board)
    vi.spyOn(api.customization, 'views').mockResolvedValue([])
    vi.spyOn(api.workspaces, 'candidates').mockResolvedValue([])
    vi.spyOn(api.contacts, 'list').mockResolvedValue({ items: [], page: 1, limit: 100, total: 0 })
    renderPage()
    expect(await screen.findByText('Acme expansion')).toBeInTheDocument()
    expect(screen.getAllByText('USD 1250.00').length).toBeGreaterThan(0)
    expect(screen.getByText('New')).toBeInTheDocument()
  })

  it('offers a keyboard/select movement path that submits the optimistic version', async () => {
    vi.spyOn(api.crm, 'pipelines').mockResolvedValue([board.pipeline])
    vi.spyOn(api.crm, 'stages').mockResolvedValue([stageOne, stageTwo])
    vi.spyOn(api.crm, 'opportunityBoard').mockResolvedValue(board)
    vi.spyOn(api.crm, 'moveOpportunity').mockResolvedValue({ ...item, stage_id: stageTwo.id, version: 4 })
    vi.spyOn(api.customization, 'views').mockResolvedValue([])
    vi.spyOn(api.workspaces, 'candidates').mockResolvedValue([])
    vi.spyOn(api.contacts, 'list').mockResolvedValue({ items: [], page: 1, limit: 100, total: 0 })
    renderPage()
    const select = await screen.findByRole('combobox', { name: 'Move Acme expansion' })
    fireEvent.change(select, { target: { value: stageTwo.id } })
    await waitFor(() => expect(api.crm.moveOpportunity).toHaveBeenCalledWith(item.id, { pipeline_id: item.pipeline_id, stage_id: stageTwo.id, version: item.version }))
  })
})
