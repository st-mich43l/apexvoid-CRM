import { useEffect, useState, type FormEvent } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../../core/api/client'
import { useAuth } from '../../../core/auth/context'
import { useWorkspace } from '../../../core/workspace/context'
import { workspaceQueryKey } from '../../../core/workspace/query'

export function OrganizationPage() {
  const queryClient = useQueryClient()
  const { platformCan } = useAuth()
  const { activeWorkspaceId, activeWorkspace, workspaces, can: workspaceCan, reload, selectWorkspace } = useWorkspace()
  const current = useQuery({ queryKey: workspaceQueryKey(activeWorkspaceId, 'current'), queryFn: api.workspaces.current, enabled: Boolean(activeWorkspaceId) })
  const [organizationName, setOrganizationName] = useState('')
  const [workspaceName, setWorkspaceName] = useState('')
  const [timezone, setTimezone] = useState('UTC')
  const [newWorkspaceName, setNewWorkspaceName] = useState('')
  const [newWorkspaceTimezone, setNewWorkspaceTimezone] = useState('UTC')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (!current.data) return
    setOrganizationName(current.data.organization.name)
    setWorkspaceName(current.data.workspace.name)
    setTimezone(current.data.workspace.timezone)
  }, [current.data])

  const save = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    try {
      if (platformCan('organization.organization.update')) await api.workspaces.organization({ name: organizationName })
      if (workspaceCan('workspace.workspace.update')) await api.workspaces.update({ name: workspaceName, timezone })
      await queryClient.invalidateQueries({ queryKey: workspaceQueryKey(activeWorkspaceId) })
      setMessage('Saved')
      window.setTimeout(() => setMessage(''), 2200)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to save organization settings')
    }
  }

  const createWorkspace = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    try {
      const created = await api.workspaces.create({ name: newWorkspaceName, timezone: newWorkspaceTimezone })
      setNewWorkspaceName('')
      await reload()
      selectWorkspace(created.id)
      setMessage('Workspace created')
      window.setTimeout(() => setMessage(''), 2200)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to create workspace')
    }
  }

  if (current.isLoading) return <p className="text-muted-foreground">Loading organization…</p>
  if (current.isError) return <p className="text-destructive">Unable to load the current workspace.</p>

  return <div className="max-w-4xl">
    <p className="mb-3 text-sm font-medium text-primary">Settings</p>
    <h1 className="text-3xl font-semibold tracking-tight">Organization</h1>
    <p className="mt-3 text-muted-foreground">Manage the company identity and the workspaces your team uses every day.</p>
    <form onSubmit={save} className="mt-8 rounded-2xl border border-border bg-card p-6 md:p-8">
      <div className="mb-6 rounded-lg border border-border bg-muted/50 p-4 text-sm text-muted-foreground">Organization changes apply to every workspace and require platform administrator access. Workspace administrators can update their current workspace below.</div>
      <label className="mb-5 block text-sm text-muted-foreground">Organization name<input required value={organizationName} disabled={!platformCan('organization.organization.update')} onChange={event => setOrganizationName(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary disabled:cursor-not-allowed disabled:opacity-60" /></label>
      <label className="mb-5 block text-sm text-muted-foreground">Current workspace<input required value={workspaceName} disabled={!workspaceCan('workspace.workspace.update')} onChange={event => setWorkspaceName(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary disabled:cursor-not-allowed disabled:opacity-60" /></label>
      <label className="mb-6 block text-sm text-muted-foreground">Timezone<select value={timezone} disabled={!workspaceCan('workspace.workspace.update')} onChange={event => setTimezone(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary disabled:cursor-not-allowed disabled:opacity-60"><option>UTC</option><option>Asia/Ho_Chi_Minh</option><option>Asia/Singapore</option><option>America/New_York</option><option>Europe/London</option></select></label>
      {error && <p className="mb-4 text-sm text-destructive">{error}</p>}
      <div className="flex items-center gap-4"><button disabled={!platformCan('organization.organization.update') && !workspaceCan('workspace.workspace.update')} className="rounded-lg bg-primary px-5 py-3 font-semibold text-primary-foreground disabled:cursor-not-allowed disabled:opacity-50">Save changes</button>{message && <span className="text-sm text-success">{message}</span>}</div>
    </form>
    <section className="mt-6 rounded-2xl border border-border bg-card p-6 md:p-8">
      <div className="mb-5"><h2 className="text-xl font-semibold">Workspaces</h2><p className="mt-1 text-sm text-muted-foreground">Each workspace has its own members, roles, and tenant data.</p></div>
      <div className="divide-y divide-border rounded-lg border border-border">{workspaces.map(workspace => <button type="button" key={workspace.id} onClick={() => selectWorkspace(workspace.id)} className={`flex w-full items-center justify-between px-4 py-3 text-left first:rounded-t-lg last:rounded-b-lg hover:bg-muted ${workspace.id === activeWorkspaceId ? 'bg-muted' : ''}`}><span><span className="block font-medium text-foreground">{workspace.name}</span><span className="text-xs text-muted-foreground">{workspace.timezone}{workspace.id === activeWorkspaceId ? ' · Current workspace' : ''}</span></span><span className="text-xs text-muted-foreground">{workspace.status}</span></button>)}</div>
      {workspaceCan('workspace.workspace.create') && <form onSubmit={createWorkspace} className="mt-6 grid gap-3 rounded-lg border border-dashed border-border p-4 md:grid-cols-[1fr_0.7fr_auto] md:items-end"><label className="text-sm text-muted-foreground">New workspace name<input required value={newWorkspaceName} onChange={event => setNewWorkspaceName(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-2 text-foreground outline-none focus:border-primary" /></label><label className="text-sm text-muted-foreground">Timezone<select value={newWorkspaceTimezone} onChange={event => setNewWorkspaceTimezone(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-2 text-foreground outline-none focus:border-primary"><option>UTC</option><option>Asia/Ho_Chi_Minh</option><option>Asia/Singapore</option><option>America/New_York</option><option>Europe/London</option></select></label><button className="rounded-lg bg-secondary px-4 py-2 font-semibold text-secondary-foreground">Create workspace</button></form>}
      {!workspaceCan('workspace.workspace.create') && <p className="mt-4 text-sm text-muted-foreground">Only workspace administrators can create additional workspaces.</p>}
    </section>
    {activeWorkspace && <p className="mt-4 text-xs text-muted-foreground">Workspace ID: {activeWorkspace.id}</p>}
  </div>
}
