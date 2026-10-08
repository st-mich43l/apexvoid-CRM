import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, type OpportunityBoard, type SavedView } from '../../../core/api/client'
import { AuthContext } from '../../../core/auth/context'
import { WorkspaceContext } from '../../../core/workspace/context'
import { OpportunityBoardPage } from './OpportunityBoardPage'

const user = { id: 'user-1', email: 'admin@example.com', username: 'admin', display_name: 'Admin', status: 'active', must_change_password: false }
const stageOne = { id: 'stage-1', pipeline_id: 'pipeline-1', key: 'new', name: 'New', description: '', color: '#7c3aed', category: 'open' as const, probability: 10, position: 0, active: true, version: 1 }
const stageTwo = { ...stageOne, id: 'stage-2', key: 'qualified', name: 'Qualified', position: 1 }
const item = { id: 'opportunity-1', title: 'Acme expansion', description: '', pipeline_id: 'pipeline-1', stage_id: stageOne.id, expected_revenue: '1250.00', currency: 'USD', outcome: 'open' as const, custom_values: {}, version: 3, created_at: '2026-10-08T00:00:00Z' }
const board: OpportunityBoard = { pipeline: { id: 'pipeline-1', workspace_id: 'workspace-1', name: 'Sales', slug: 'sales', description: '', color: '#7c3aed', status: 'active', default: true, version: 1 }, stages: [{ stage: stageOne, count: 1, totals_by_currency: { USD: '1250.00' }, items: [item], next_cursor: null }, { stage: stageTwo, count: 0, totals_by_currency: {}, items: [], next_cursor: null }], filter: { search: '', outcome: 'open', limit: 20 } }

function renderPage(initialEntry = '/crm/opportunities?pipeline_id=pipeline-1&view=kanban', permissions = ['crm.opportunity.read', 'crm.opportunity.transition']) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><AuthContext.Provider value={{ user, loading: false, platformPermissions: [], login: vi.fn(), logout: vi.fn(), platformCan: () => true, reload: vi.fn() }}><WorkspaceContext.Provider value={{ workspaces: [], activeWorkspace: null, activeWorkspaceId: 'workspace-1', setup: null, workspacePermissions: permissions, loading: false, selectWorkspace: vi.fn(), can: permission => permissions.includes(permission), reload: vi.fn() }}><MemoryRouter initialEntries={[initialEntry]}><OpportunityBoardPage/></MemoryRouter></WorkspaceContext.Provider></AuthContext.Provider></QueryClientProvider>)
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
  it('moves a card dropped into a different stage rather than searching the destination cards', async () => {
    vi.spyOn(api.crm, 'pipelines').mockResolvedValue([board.pipeline])
    vi.spyOn(api.crm, 'stages').mockResolvedValue([stageOne, stageTwo])
    const boardAPI = vi.spyOn(api.crm, 'opportunityBoard').mockResolvedValue(board)
    vi.spyOn(api.crm, 'moveOpportunity').mockResolvedValue({ ...item, stage_id: stageTwo.id, version: 4 })
    vi.spyOn(api.customization, 'views').mockResolvedValue([])
    vi.spyOn(api.workspaces, 'candidates').mockResolvedValue([])
    vi.spyOn(api.contacts, 'list').mockResolvedValue({ items: [], page: 1, limit: 100, total: 0 })
    renderPage()
    expect(await screen.findByText('Acme expansion')).toBeInTheDocument()
    const destination = screen.getByRole('region', { name: 'Stage Qualified' })
    fireEvent.dragOver(destination)
    fireEvent.drop(destination, { dataTransfer: { getData: () => item.id } })
    await waitFor(() => expect(api.crm.moveOpportunity).toHaveBeenCalledWith(item.id, {
      pipeline_id: item.pipeline_id, stage_id: stageTwo.id, version: item.version,
    }))
    await waitFor(() => expect(boardAPI.mock.calls.length).toBeGreaterThan(1))
  })

  it('fetches new records when navigating from list page 1 to page 2', async () => {
    vi.spyOn(api.crm, 'pipelines').mockResolvedValue([board.pipeline])
    vi.spyOn(api.crm, 'stages').mockResolvedValue([stageOne, stageTwo])
    vi.spyOn(api.crm, 'opportunities').mockImplementation(async params => ({
      items: [{ ...item, id: Number(params?.page) === 2 ? 'page-2-id' : 'page-1-id', title: Number(params?.page) === 2 ? 'Page two deal' : 'Page one deal' }],
      page: Number(params?.page) || 1,
      limit: 25,
      total: 30,
    }))
    vi.spyOn(api.customization, 'views').mockResolvedValue([])
    vi.spyOn(api.workspaces, 'candidates').mockResolvedValue([])
    vi.spyOn(api.contacts, 'list').mockResolvedValue({ items: [], page: 1, limit: 100, total: 0 })
    renderPage('/crm/opportunities?pipeline_id=pipeline-1&view=list&page=1')
    expect(await screen.findByText('Page one deal')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }))
    expect(await screen.findByText('Page two deal')).toBeInTheDocument()
    await waitFor(() => expect(api.crm.opportunities).toHaveBeenCalledWith(expect.objectContaining({ page: 2 })))
    fireEvent.click(screen.getByRole('button', { name: 'Previous page' }))
    expect(await screen.findByText('Page one deal')).toBeInTheDocument()
  })

  it('applies a saved view and its pipeline filter atomically', async () => {
    const secondPipeline = { ...board.pipeline, id: 'pipeline-2', name: 'Second Pipeline', default: false }
    const saved: SavedView = { id: 'view-1', workspace_id: 'workspace-1', entity: 'crm.opportunity', owner_user_id: 'user-1', name: 'Second pipeline deals', shared: false, filters: [{ field: 'pipeline_id', operator: 'eq', value: 'pipeline-2' }], columns: ['title'], sort_field: 'title', sort_direction: 'asc', created_at: '', updated_at: '' }
    vi.spyOn(api.crm, 'pipelines').mockResolvedValue([board.pipeline, secondPipeline])
    vi.spyOn(api.crm, 'stages').mockResolvedValue([stageOne, stageTwo])
    const boardAPI = vi.spyOn(api.crm, 'opportunityBoard').mockResolvedValue(board)
    vi.spyOn(api.customization, 'views').mockResolvedValue([saved])
    vi.spyOn(api.workspaces, 'candidates').mockResolvedValue([])
    vi.spyOn(api.contacts, 'list').mockResolvedValue({ items: [], page: 1, limit: 100, total: 0 })
    renderPage(undefined, ['crm.opportunity.read', 'crm.opportunity.transition', 'customization.schema.read'])
    const selection = await screen.findByRole('combobox', { name: 'Saved opportunity view' })
    await screen.findByRole('option', { name: /Second pipeline deals/ })
    fireEvent.change(selection, { target: { value: saved.id } })
    await waitFor(() => expect(boardAPI).toHaveBeenCalledWith('pipeline-2', expect.objectContaining({ view_id: saved.id })))
  })

  it('shows version conflicts and refetches the authoritative board after a failed transition', async () => {
    vi.spyOn(api.crm, 'pipelines').mockResolvedValue([board.pipeline])
    vi.spyOn(api.crm, 'stages').mockResolvedValue([stageOne, stageTwo])
    const boardAPI = vi.spyOn(api.crm, 'opportunityBoard').mockResolvedValue(board)
    vi.spyOn(api.crm, 'moveOpportunity').mockRejectedValue(Object.assign(new Error('stale update'), { status: 409 }))
    vi.spyOn(api.customization, 'views').mockResolvedValue([])
    vi.spyOn(api.workspaces, 'candidates').mockResolvedValue([])
    vi.spyOn(api.contacts, 'list').mockResolvedValue({ items: [], page: 1, limit: 100, total: 0 })
    renderPage()
    const select = await screen.findByRole('combobox', { name: 'Move Acme expansion' })
    fireEvent.change(select, { target: { value: stageTwo.id } })
    expect(await screen.findByText(/This opportunity changed elsewhere/)).toBeInTheDocument()
    await waitFor(() => expect(boardAPI.mock.calls.length).toBeGreaterThan(1))
  })

})
