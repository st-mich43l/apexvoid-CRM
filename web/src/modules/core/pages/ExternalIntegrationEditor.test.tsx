import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { ExternalIntegration } from '../../../core/api/client'
import { ExternalIntegrationEditor } from './ExternalIntegrationEditor'

const installed: ExternalIntegration = {
  id: 'cafe',
  deployment: 'external',
  display_name: 'ApexVoid Café',
  description: 'Orders and photo booth management',
  version: '2.3.4',
  api_contract_version: 'v1',
  service_identity: 'cafe-service',
  service_endpoint: 'http://apexvoid-cafe:8090',
  health_endpoint: 'http://apexvoid-cafe:8090/health',
  frontend_route: '/apps/cafe',
  settings_route: '',
  enabled: true,
  credential_revoked: false,
  status: 'active',
  access_match: 'all',
  access_permissions: ['cafe.order.read'],
  permissions: [{ name: 'cafe.order.read', display_name: 'View orders', description: '', scope: 'workspace' }],
  health: 'healthy',
}

describe('Existing application metadata editor', () => {
  it('displays the actual app-owned version and contract as read-only and preserves them on save', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    render(<ExternalIntegrationEditor initial={installed} onCancel={() => {}} onSubmit={save} />)
    expect(screen.getByText('Application version · app-owned')).toBeInTheDocument()
    expect(screen.getByText('2.3.4')).toBeInTheDocument()
    expect(screen.getByText('API contract · app-owned')).toBeInTheDocument()
    expect(screen.getByText('v1')).toBeInTheDocument()
    expect(screen.queryByLabelText('Application version')).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: 'API contract' })).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Display name'), { target: { value: 'New Café Display Name' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save metadata' }))
    await waitFor(() => expect(save).toHaveBeenCalledWith(expect.objectContaining({
      id: 'cafe', display_name: 'New Café Display Name', version: '2.3.4', api_contract_version: 'v1',
    })))
  })
})
