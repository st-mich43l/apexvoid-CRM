import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, cleanup } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, afterEach, vi } from 'vitest'
import { api, type ERPProduct, type CafeBooking } from '../../core/api/client'
import { AuthContext } from '../../core/auth/context'
import { WorkspaceContext } from '../../core/workspace/context'
import { cafeModule } from './index'

const latte: ERPProduct = {
  id: 'product-1', workspace_id: 'workspace-1', sku: 'CAFE-LATTE', name: 'Iced Latte',
  description: '', kind: 'good', unit: 'unit', unit_price: '49000.0000', currency: 'VND',
  status: 'active', version: 1, created_at: '', updated_at: '',
}
const photo: ERPProduct = { ...latte, id: 'service-1', sku: 'PHOTO-20M', name: 'Photo Session 20m', kind: 'service', unit_price: '80000.0000' }
function setup(id: 'cafe-counter' | 'cafe-booths', can = (_permission: string) => true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const element = cafeModule.routes?.find(route => route.id === id)?.element
  if (!element) throw new Error('missing cafe route')
  return render(
    <QueryClientProvider client={client}>
      <AuthContext.Provider value={{
        user: { id: 'user-1', email: 'a@coffee.test', display_name: 'Cashier', status: 'active', must_change_password: false },
        loading: false, platformPermissions: [], platformCan: () => false, login: async () => {}, logout: async () => {}, reload: async () => {},
      }}>
        <WorkspaceContext.Provider value={{
          workspaces: [], activeWorkspace: null, activeWorkspaceId: 'workspace-1', setup: null,
          workspacePermissions: [], loading: false, can, reload: async () => {}, selectWorkspace: () => {},
        }}>
          <MemoryRouter>{element}</MemoryRouter>
        </WorkspaceContext.Provider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  )
}
afterEach(() => { cleanup();vi.restoreAllMocks() })

describe('Café & Photo Booth application', () => {
  it('builds a café cart from ERP goods and submits an exact product/quantity order', async () => {
    vi.spyOn(api.erp, 'products').mockResolvedValue({ items: [latte, photo], page: 1, limit: 100, total: 2 })
    vi.spyOn(api.cafe, 'orders').mockResolvedValue([])
    const create = vi.spyOn(api.cafe, 'createOrder').mockResolvedValue({
      id: 'order-1', workspace_id: 'workspace-1', status: 'open',
      currency: 'VND', total: '98000.0000', note: '', line_count: 1, created_at: '',
    })
    setup('cafe-counter')
    const tile = await screen.findByText('Iced Latte')
    fireEvent.click(tile.closest('button')!)
    fireEvent.click(screen.getByRole('button', { name: 'Add one Iced Latte' }))
    fireEvent.click(screen.getByRole('button', { name: 'Create order' }))
    await waitFor(() => expect(create).toHaveBeenCalledWith({
      note: '', lines: [{ product_id: 'product-1', quantity: 2 }],
    }))
    expect(screen.queryByText('Photo Session 20m')).not.toBeInTheDocument()
  })

  it('schedules an ERP photo service for a named physical booth', async () => {
    vi.spyOn(api.erp, 'products').mockResolvedValue({ items: [latte, photo], page: 1, limit: 100, total: 2 })
    vi.spyOn(api.cafe, 'booths').mockResolvedValue([
      { id: 'booth-1', workspace_id: 'workspace-1', name: 'Booth A', active: true, created_at: '' },
    ])
    vi.spyOn(api.cafe, 'bookings').mockResolvedValue([])
    const reserve = vi.spyOn(api.cafe, 'reserve').mockImplementation(async input => ({
      id: 'booking-1', workspace_id: 'workspace-1', booth_name: 'Booth A',
      status: 'reserved', price: '80000.0000', currency: 'VND', created_at: '', ...input,
    } satisfies CafeBooking))
    setup('cafe-booths')
    await screen.findByRole('option', { name: 'Booth A' })
    fireEvent.change(screen.getByLabelText('Booth'), { target: { value: 'booth-1' } })
    fireEvent.change(screen.getByLabelText('Photo package'), { target: { value: 'service-1' } })
    fireEvent.change(screen.getByLabelText('Guest name'), { target: { value: 'Group One' } })
    const future = new Date(Date.now() + 60 * 60_000)
    const local = new Date(future.getTime() - future.getTimezoneOffset() * 60_000).toISOString().slice(0, 16)
    fireEvent.change(screen.getByLabelText('Start time'), { target: { value: local } })
    fireEvent.click(screen.getByRole('button', { name: 'Reserve booth' }))
    await waitFor(() => expect(reserve).toHaveBeenCalledWith(expect.objectContaining({
      booth_id: 'booth-1', package_product_id: 'service-1', guest_name: 'Group One',
    })))
  })

  it('does not show the counter to unauthorized workspace users', () => {
    setup('cafe-counter', permission => permission !== 'cafe.order.read')
    expect(screen.getByText('Counter unavailable')).toBeInTheDocument()
  })
})
