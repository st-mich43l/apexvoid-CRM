import { useEffect, useState, type FormEvent } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../../core/api/client'
import { useAuth } from '../../../core/auth/context'
import { useWorkspace } from '../../../core/workspace/context'
import { userWorkspaceQueryKey } from '../../../core/workspace/query'

export function OrganizationPage() {
  const queryClient = useQueryClient()
  const { platformCan, user } = useAuth()
  const { activeWorkspaceId, workspaces, can: workspaceCan, reload, selectWorkspace } = useWorkspace()
  const current = useQuery({ queryKey: userWorkspaceQueryKey(user?.id, activeWorkspaceId, 'current'), queryFn: api.workspaces.current, enabled: Boolean(activeWorkspaceId) })
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
      await queryClient.invalidateQueries({ queryKey: userWorkspaceQueryKey(user?.id, activeWorkspaceId) })
      setMessage('Changes saved')
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

  if (current.isLoading) return <section className="max-w-5xl"><div className="h-5 w-24 animate-pulse rounded bg-muted" /><div className="mt-3 h-10 w-64 animate-pulse rounded bg-muted" /><div className="mt-8 h-96 animate-pulse rounded-2xl bg-muted" /></section>
  if (current.isError) return <section className="max-w-5xl"><p className="text-destructive">Unable to load the current workspace.</p></section>

  return <section className="w-full max-w-5xl">
    <header className="mb-8"><p className="text-sm font-medium uppercase tracking-[0.18em] text-primary">Settings</p><h1 className="mt-2 text-3xl font-semibold tracking-tight sm:text-4xl">Organization</h1><p className="mt-3 max-w-2xl text-muted-foreground">Manage the company identity and the workspaces your team uses every day.</p></header>
    <form onSubmit={save} className="rounded-2xl border border-border bg-card p-5 shadow-sm sm:p-7">
      <div className="mb-7 rounded-xl border border-border bg-muted/50 p-4 text-sm leading-6 text-muted-foreground">Organization changes apply to every workspace and require platform administrator access. Workspace administrators can update their current workspace below.</div>
      <div className="grid gap-7 lg:grid-cols-2">
        <div><h2 className="text-lg font-semibold">Organization details</h2><p className="mt-1 text-sm text-muted-foreground">The identity your team sees across ApexVoid.</p><label className="mt-5 block text-sm font-medium text-foreground">Organization name<input required value={organizationName} disabled={!platformCan('organization.organization.update')} onChange={event => setOrganizationName(event.target.value)} className="mt-2 w-full rounded-xl border border-border bg-background px-3 py-3 text-foreground outline-none transition focus:border-primary disabled:cursor-not-allowed disabled:opacity-60" /></label></div>
        <div><h2 className="text-lg font-semibold">Workspace settings</h2><p className="mt-1 text-sm text-muted-foreground">These settings apply to the workspace you are viewing.</p><label className="mt-5 block text-sm font-medium text-foreground">Workspace name<input required value={workspaceName} disabled={!workspaceCan('workspace.workspace.update')} onChange={event => setWorkspaceName(event.target.value)} className="mt-2 w-full rounded-xl border border-border bg-background px-3 py-3 text-foreground outline-none transition focus:border-primary disabled:cursor-not-allowed disabled:opacity-60" /></label><label className="mt-5 block text-sm font-medium text-foreground">Timezone<select value={timezone} disabled={!workspaceCan('workspace.workspace.update')} onChange={event => setTimezone(event.target.value)} className="mt-2 w-full rounded-xl border border-border bg-background px-3 py-3 text-foreground outline-none transition focus:border-primary disabled:cursor-not-allowed disabled:opacity-60"><option>UTC</option><option>Asia/Ho_Chi_Minh</option><option>Asia/Singapore</option><option>America/New_York</option><option>Europe/London</option></select></label></div>
      </div>
      {error && <p className="mt-6 rounded-xl border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{error}</p>}
      <div className="mt-8 flex flex-wrap items-center gap-4 border-t border-border pt-6"><button disabled={!platformCan('organization.organization.update') && !workspaceCan('workspace.workspace.update')} className="rounded-xl bg-primary px-5 py-3 font-semibold text-primary-foreground shadow-sm transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50">Save changes</button>{message && <span className="text-sm text-success">{message}</span>}</div>
    </form>
    <section className="mt-6 rounded-2xl border border-border bg-card p-5 shadow-sm sm:p-7"><div className="mb-5"><h2 className="text-xl font-semibold">Available workspaces</h2><p className="mt-1 text-sm text-muted-foreground">Each workspace has its own members, roles, and tenant data.</p></div><div className="divide-y divide-border overflow-hidden rounded-xl border border-border">{workspaces.map(workspace => <button type="button" key={workspace.id} onClick={() => selectWorkspace(workspace.id)} className={`flex w-full items-center justify-between gap-4 px-4 py-4 text-left transition hover:bg-muted ${workspace.id === activeWorkspaceId ? 'bg-muted' : 'bg-card'}`}><span className="min-w-0"><span className="block truncate font-medium text-foreground">{workspace.name}</span><span className="text-xs text-muted-foreground">{workspace.timezone}{workspace.id === activeWorkspaceId ? ' · Current workspace' : ''}</span></span><span className="shrink-0 rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground">{workspace.status}</span></button>)}</div>{workspaceCan('workspace.workspace.create') && <form onSubmit={createWorkspace} className="mt-6 grid gap-4 rounded-xl border border-dashed border-border p-4 sm:grid-cols-[1fr_0.7fr_auto] sm:items-end"><label className="text-sm font-medium text-foreground">New workspace name<input required value={newWorkspaceName} onChange={event => setNewWorkspaceName(event.target.value)} className="mt-2 w-full rounded-xl border border-border bg-background px-3 py-2.5 text-foreground outline-none focus:border-primary" /></label><label className="text-sm font-medium text-foreground">Timezone<select value={newWorkspaceTimezone} onChange={event => setNewWorkspaceTimezone(event.target.value)} className="mt-2 w-full rounded-xl border border-border bg-background px-3 py-2.5 text-foreground outline-none focus:border-primary"><option>UTC</option><option>Asia/Ho_Chi_Minh</option><option>Asia/Singapore</option><option>America/New_York</option><option>Europe/London</option></select></label><button className="rounded-xl bg-secondary px-4 py-2.5 font-semibold text-secondary-foreground transition hover:bg-muted">Create workspace</button></form>}{!workspaceCan('workspace.workspace.create') && <p className="mt-4 text-sm text-muted-foreground">Only workspace administrators can create additional workspaces.</p>}</section>
  </section>
}
