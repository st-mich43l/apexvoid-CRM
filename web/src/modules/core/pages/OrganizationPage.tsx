import { useEffect, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../../core/api/client'
import { useWorkspace } from '../../../core/workspace/context'

export function OrganizationPage() {
  const queryClient = useQueryClient()
  const { activeWorkspaceId } = useWorkspace()
  const current = useQuery({ queryKey: ['workspace', activeWorkspaceId, 'current'], queryFn: api.workspaces.current, enabled: Boolean(activeWorkspaceId) })
  const [organizationName, setOrganizationName] = useState('')
  const [workspaceName, setWorkspaceName] = useState('')
  const [timezone, setTimezone] = useState('UTC')
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (!current.data) return
    setOrganizationName(current.data.organization.name)
    setWorkspaceName(current.data.workspace.name)
    setTimezone(current.data.workspace.timezone)
  }, [current.data])

  const save = async (event: React.FormEvent) => {
    event.preventDefault()
    await api.workspaces.organization({ name: organizationName })
    await api.workspaces.update({ name: workspaceName, timezone })
    await queryClient.invalidateQueries({ queryKey: ['workspace', activeWorkspaceId] })
    await queryClient.invalidateQueries({ queryKey: ['workspaces'] })
    setMessage('Saved')
    window.setTimeout(() => setMessage(''), 2200)
  }

  if (current.isLoading) return <p className="text-zinc-500">Loading organization…</p>
  return <div className="max-w-3xl"><p className="mb-3 text-sm font-medium text-accent">Settings</p><h1 className="text-3xl font-semibold tracking-tight">Organization</h1><p className="mt-3 text-zinc-500">Manage the company identity and the workspace your team uses every day.</p><form onSubmit={save} className="mt-8 rounded-2xl border border-line bg-panel p-6 md:p-8"><label className="mb-5 block text-sm text-zinc-400">Organization name<input required value={organizationName} onChange={event => setOrganizationName(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent" /></label><label className="mb-5 block text-sm text-zinc-400">Workspace name<input required value={workspaceName} onChange={event => setWorkspaceName(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent" /></label><label className="mb-6 block text-sm text-zinc-400">Timezone<select value={timezone} onChange={event => setTimezone(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent"><option>UTC</option><option>Asia/Ho_Chi_Minh</option><option>Asia/Singapore</option><option>America/New_York</option><option>Europe/London</option></select></label><div className="flex items-center gap-4"><button className="rounded-lg bg-accent px-5 py-3 font-semibold text-ink">Save changes</button>{message && <span className="text-sm text-emerald-300">{message}</span>}</div></form></div>
}
