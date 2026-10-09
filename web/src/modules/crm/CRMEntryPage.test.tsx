import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { WorkspaceContext, type WorkspaceContextValue } from '../../core/workspace/context'
import { CRMEntryPage } from './index'

function workspace(permissions: string[]): WorkspaceContextValue {
  return { workspaces: [], activeWorkspace: null, activeWorkspaceId: 'workspace-1', setup: null, workspacePermissions: permissions, loading: false, selectWorkspace: () => {}, can: permission => permissions.includes(permission), reload: async () => {} }
}

function renderEntry(permissions: string[]) {
  return render(<WorkspaceContext.Provider value={workspace(permissions)}><MemoryRouter initialEntries={['/crm']}><Routes><Route path="/crm" element={<CRMEntryPage />} /><Route path="/crm/leads" element={<p>Leads destination</p>} /><Route path="/crm/opportunities" element={<p>Opportunities destination</p>} /></Routes></MemoryRouter></WorkspaceContext.Provider>)
}

describe('CRMEntryPage', () => {
  it('sends an opportunity-only user to opportunities instead of leads', () => {
    renderEntry(['crm.opportunity.read'])
    expect(screen.getByText('Opportunities destination')).toBeInTheDocument()
  })

  it('does not expose CRM when the workspace grants neither entry permission', () => {
    renderEntry([])
    expect(screen.getByText('CRM is unavailable')).toBeInTheDocument()
  })
})
