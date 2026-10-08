import { useEffect, useState, type FormEvent } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, CheckCircle2, ChevronRight, Globe2, Layers3, LockKeyhole, Plus, ShieldCheck, X } from 'lucide-react'
import { api } from '../../../core/api/client'
import { useAuth } from '../../../core/auth/context'
import { useWorkspace } from '../../../core/workspace/context'
import { userWorkspaceQueryKey } from '../../../core/workspace/query'

type Feedback = { kind: 'success' | 'error'; message: string } | null

const timezones = [
  { value: 'UTC', label: 'UTC' },
  { value: 'Asia/Ho_Chi_Minh', label: 'Ho Chi Minh (GMT+7)' },
  { value: 'Asia/Singapore', label: 'Singapore (GMT+8)' },
  { value: 'America/New_York', label: 'New York' },
  { value: 'Europe/London', label: 'London' },
]

const inputClass = 'mt-2 block h-11 w-full rounded-xl border border-border bg-background px-3.5 text-sm text-foreground shadow-sm shadow-black/[0.02] outline-none transition placeholder:text-muted-foreground/70 focus:border-primary focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:bg-muted/50 disabled:text-muted-foreground'
const labelClass = 'block text-sm font-medium text-foreground'
const primaryButton = 'inline-flex min-h-10 items-center justify-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:brightness-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary disabled:cursor-not-allowed disabled:opacity-50'

function FeedbackMessage({ value }: { value: Feedback }) {
  if (!value) return null
  return (
    <p role={value.kind === 'error' ? 'alert' : 'status'} className={'flex items-center gap-2 text-sm ' + (value.kind === 'error' ? 'text-destructive' : 'text-success')}>
      {value.kind === 'success' && <CheckCircle2 size={16} aria-hidden="true" />}
      {value.message}
    </p>
  )
}

export function OrganizationPage() {
  const queryClient = useQueryClient()
  const { platformCan, user } = useAuth()
  const { activeWorkspaceId, workspaces, can: workspaceCan, reload, selectWorkspace } = useWorkspace()

  const contextKey = userWorkspaceQueryKey(user?.id, activeWorkspaceId, 'current')
  const current = useQuery({
    queryKey: contextKey,
    queryFn: api.workspaces.current,
    enabled: Boolean(activeWorkspaceId),
  })

  const canEditOrganization = platformCan('organization.organization.update')
  const canEditWorkspace = workspaceCan('workspace.workspace.update')
  const canCreateWorkspace = workspaceCan('workspace.workspace.create')

  const [organizationName, setOrganizationName] = useState('')
  const [workspaceName, setWorkspaceName] = useState('')
  const [timezone, setTimezone] = useState('UTC')
  const [newWorkspaceName, setNewWorkspaceName] = useState('')
  const [newWorkspaceTimezone, setNewWorkspaceTimezone] = useState('UTC')
  const [showCreate, setShowCreate] = useState(false)
  const [savingOrganization, setSavingOrganization] = useState(false)
  const [savingWorkspace, setSavingWorkspace] = useState(false)
  const [creatingWorkspace, setCreatingWorkspace] = useState(false)
  const [organizationFeedback, setOrganizationFeedback] = useState<Feedback>(null)
  const [workspaceFeedback, setWorkspaceFeedback] = useState<Feedback>(null)
  const [createFeedback, setCreateFeedback] = useState<Feedback>(null)

  useEffect(() => {
    if (!current.data) return
    setOrganizationName(current.data.organization.name)
    setWorkspaceName(current.data.workspace.name)
    setTimezone(current.data.workspace.timezone)
    setOrganizationFeedback(null)
    setWorkspaceFeedback(null)
  }, [current.data])

  const organizationDirty = organizationName.trim() !== (current.data?.organization.name ?? '')
  const workspaceDirty = workspaceName.trim() !== (current.data?.workspace.name ?? '') ||
    timezone !== (current.data?.workspace.timezone ?? 'UTC')

  const saveOrganization = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!canEditOrganization || !organizationName.trim() || !organizationDirty) return
    setSavingOrganization(true)
    setOrganizationFeedback(null)
    try {
      await api.workspaces.organization({ name: organizationName.trim() })
      await queryClient.invalidateQueries({ queryKey: contextKey })
      setOrganizationFeedback({ kind: 'success', message: 'Organization updated' })
    } catch (reason) {
      setOrganizationFeedback({ kind: 'error', message: reason instanceof Error ? reason.message : 'Unable to update organization' })
    } finally {
      setSavingOrganization(false)
    }
  }

  const saveWorkspace = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!canEditWorkspace || !workspaceName.trim() || !workspaceDirty) return
    setSavingWorkspace(true)
    setWorkspaceFeedback(null)
    try {
      await api.workspaces.update({ name: workspaceName.trim(), timezone })
      await Promise.all([queryClient.invalidateQueries({ queryKey: contextKey }), reload()])
      setWorkspaceFeedback({ kind: 'success', message: 'Workspace settings saved' })
    } catch (reason) {
      setWorkspaceFeedback({ kind: 'error', message: reason instanceof Error ? reason.message : 'Unable to update workspace' })
    } finally {
      setSavingWorkspace(false)
    }
  }

  const createWorkspace = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!canCreateWorkspace || !newWorkspaceName.trim()) return
    setCreatingWorkspace(true)
    setCreateFeedback(null)
    try {
      const created = await api.workspaces.create({ name: newWorkspaceName.trim(), timezone: newWorkspaceTimezone })
      await reload()
      setNewWorkspaceName('')
      setNewWorkspaceTimezone('UTC')
      setShowCreate(false)
      selectWorkspace(created.id)
    } catch (reason) {
      setCreateFeedback({ kind: 'error', message: reason instanceof Error ? reason.message : 'Unable to create workspace' })
    } finally {
      setCreatingWorkspace(false)
    }
  }

  if (current.isLoading) {
    return (
      <section className="mx-auto w-full max-w-[1180px] space-y-6" aria-label="Loading organization settings">
        <div className="h-20 max-w-lg animate-pulse rounded-2xl bg-muted" />
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.55fr)_minmax(320px,0.95fr)]">
          <div className="space-y-6"><div className="h-64 animate-pulse rounded-2xl bg-muted" /><div className="h-72 animate-pulse rounded-2xl bg-muted" /></div>
          <div className="h-80 animate-pulse rounded-2xl bg-muted" />
        </div>
      </section>
    )
  }

  if (current.isError || !current.data) {
    return (
      <section className="mx-auto max-w-[1180px] rounded-2xl border border-border bg-card p-8">
        <h1 className="text-xl font-semibold text-foreground">Unable to load organization</h1>
        <p className="mt-2 text-sm text-muted-foreground">Check your connection and try again.</p>
        <button type="button" onClick={() => void current.refetch()} className={primaryButton + ' mt-5'}>Retry</button>
      </section>
    )
  }

  return (
    <section className="mx-auto w-full max-w-[1180px] pb-8">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="mb-3 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
            <span>Settings</span><ChevronRight size={13} aria-hidden="true" /><span className="text-primary">Organization</span>
          </div>
          <h1 className="text-3xl font-semibold tracking-tight text-foreground sm:text-[34px]">Organization</h1>
          <p className="mt-2 max-w-xl text-sm leading-6 text-muted-foreground">Manage your company identity, workspace preferences, and team spaces.</p>
        </div>
        <div className="flex items-center gap-2 rounded-full border border-border bg-card px-3.5 py-2 text-xs font-medium text-muted-foreground shadow-sm">
          <Layers3 size={15} className="text-primary" aria-hidden="true" />
          {workspaces.length} {workspaces.length === 1 ? 'workspace' : 'workspaces'}
        </div>
      </div>

      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1.55fr)_minmax(320px,0.95fr)]">
        <div className="min-w-0 space-y-6">
          <form onSubmit={saveOrganization} className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="flex items-start gap-3 border-b border-border/80 px-5 py-5 sm:px-6">
              <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><Building2 size={19} aria-hidden="true" /></div>
              <div className="min-w-0">
                <h2 className="text-base font-semibold text-foreground">Company profile</h2>
                <p className="mt-1 text-xs leading-5 text-muted-foreground">The organization identity shared by your workspaces.</p>
              </div>
              {!canEditOrganization && <LockKeyhole size={16} className="ml-auto shrink-0 text-muted-foreground" aria-label="Read-only" />}
            </div>
            <div className="px-5 py-6 sm:px-6">
              <label htmlFor="organization-name" className={labelClass}>Organization name</label>
              <input id="organization-name" required maxLength={180} autoComplete="organization" value={organizationName} disabled={!canEditOrganization || savingOrganization} onChange={event => { setOrganizationName(event.target.value); setOrganizationFeedback(null) }} className={inputClass} />
              <p className="mt-3 text-xs leading-5 text-muted-foreground">This name is visible across all workspaces in your organization.</p>
            </div>
            {canEditOrganization ? (
              <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border/80 bg-muted/20 px-5 py-4 sm:px-6">
                <FeedbackMessage value={organizationFeedback} />
                <button type="submit" disabled={!organizationDirty || !organizationName.trim() || savingOrganization} className={primaryButton + ' ml-auto'}>{savingOrganization ? 'Saving…' : 'Save organization'}</button>
              </div>
            ) : (
              <div className="flex items-center gap-2 border-t border-border/80 bg-muted/20 px-5 py-3.5 text-xs text-muted-foreground sm:px-6">
                <LockKeyhole size={14} aria-hidden="true" /> Only platform administrators can change the organization name.
              </div>
            )}
          </form>

          <form onSubmit={saveWorkspace} className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="flex items-start gap-3 border-b border-border/80 px-5 py-5 sm:px-6">
              <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><Globe2 size={19} aria-hidden="true" /></div>
              <div className="min-w-0">
                <h2 className="text-base font-semibold text-foreground">Workspace preferences</h2>
                <p className="mt-1 text-xs leading-5 text-muted-foreground">Configure the workspace you're currently using.</p>
              </div>
              <span className="ml-auto shrink-0 rounded-full bg-primary/10 px-2.5 py-1 text-[11px] font-semibold text-primary">Current</span>
            </div>
            <div className="grid gap-5 px-5 py-6 sm:grid-cols-2 sm:px-6">
              <div className="min-w-0">
                <label htmlFor="workspace-name" className={labelClass}>Workspace name</label>
                <input id="workspace-name" required maxLength={180} value={workspaceName} disabled={!canEditWorkspace || savingWorkspace} onChange={event => { setWorkspaceName(event.target.value); setWorkspaceFeedback(null) }} className={inputClass} />
              </div>
              <div className="min-w-0">
                <label htmlFor="workspace-timezone" className={labelClass}>Timezone</label>
                <select id="workspace-timezone" value={timezone} disabled={!canEditWorkspace || savingWorkspace} onChange={event => { setTimezone(event.target.value); setWorkspaceFeedback(null) }} className={inputClass}>
                  {timezones.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}
                </select>
              </div>
              <p className="text-xs leading-5 text-muted-foreground sm:col-span-2">Workspace settings only affect this workspace, not your other teams.</p>
            </div>
            {canEditWorkspace ? (
              <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border/80 bg-muted/20 px-5 py-4 sm:px-6">
                <FeedbackMessage value={workspaceFeedback} />
                <button type="submit" disabled={!workspaceDirty || !workspaceName.trim() || savingWorkspace} className={primaryButton + ' ml-auto'}>{savingWorkspace ? 'Saving…' : 'Save workspace'}</button>
              </div>
            ) : (
              <div className="flex items-center gap-2 border-t border-border/80 bg-muted/20 px-5 py-3.5 text-xs text-muted-foreground sm:px-6">
                <LockKeyhole size={14} aria-hidden="true" /> Your role has read-only access to these settings.
              </div>
            )}
          </form>
        </div>

        <aside className="min-w-0 space-y-4" aria-label="Workspace management">
          <section className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/80 px-5 py-5">
              <div>
                <h2 className="text-base font-semibold text-foreground">Workspaces</h2>
                <p className="mt-1 text-xs text-muted-foreground">Your team's separate working spaces.</p>
              </div>
              {canCreateWorkspace && !showCreate && <button type="button" onClick={() => { setShowCreate(true); setCreateFeedback(null) }} className="inline-flex items-center gap-1.5 rounded-lg border border-primary/20 bg-primary/10 px-3 py-2 text-xs font-semibold text-primary transition hover:bg-primary/15"><Plus size={14} aria-hidden="true" /> New</button>}
            </div>
            <div className="space-y-2 p-3">
              {workspaces.map(workspace => {
                const selected = workspace.id === activeWorkspaceId
                return (
                  <button type="button" key={workspace.id} disabled={selected} onClick={() => selectWorkspace(workspace.id)} aria-current={selected ? 'true' : undefined} className={'flex w-full items-center gap-3 rounded-xl border p-3 text-left transition ' + (selected ? 'border-primary/25 bg-primary/[0.06]' : 'border-transparent bg-muted/35 hover:border-border hover:bg-muted/65')}>
                    <span className={'grid h-9 w-9 shrink-0 place-items-center rounded-lg ' + (selected ? 'bg-primary/10 text-primary' : 'bg-card text-muted-foreground')}><Layers3 size={17} aria-hidden="true" /></span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-semibold text-foreground">{workspace.name}</span>
                      <span className="mt-0.5 block truncate text-[11px] text-muted-foreground">{workspace.timezone}</span>
                    </span>
                    {selected ? <span className="shrink-0 rounded-full bg-primary/10 px-2 py-1 text-[10px] font-semibold text-primary">Current</span> : <ChevronRight size={16} className="shrink-0 text-muted-foreground" aria-label="Switch workspace" />}
                  </button>
                )
              })}
              {workspaces.length === 0 && <p className="px-3 py-5 text-sm text-muted-foreground">No workspaces available.</p>}
            </div>
            {canCreateWorkspace && showCreate && (
              <form onSubmit={createWorkspace} className="space-y-4 border-t border-border bg-muted/20 p-5">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-semibold text-foreground">Create workspace</h3>
                  <button type="button" onClick={() => { setShowCreate(false); setCreateFeedback(null) }} aria-label="Cancel workspace creation" className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"><X size={16} /></button>
                </div>
                <div><label htmlFor="new-workspace-name" className={labelClass}>Name</label><input id="new-workspace-name" required maxLength={180} autoFocus placeholder="e.g. Sales team" value={newWorkspaceName} disabled={creatingWorkspace} onChange={event => setNewWorkspaceName(event.target.value)} className={inputClass} /></div>
                <div><label htmlFor="new-workspace-timezone" className={labelClass}>Timezone</label><select id="new-workspace-timezone" value={newWorkspaceTimezone} disabled={creatingWorkspace} onChange={event => setNewWorkspaceTimezone(event.target.value)} className={inputClass}>{timezones.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select></div>
                <FeedbackMessage value={createFeedback} />
                <button type="submit" disabled={creatingWorkspace || !newWorkspaceName.trim()} className={primaryButton + ' w-full'}><Plus size={15} aria-hidden="true" /> {creatingWorkspace ? 'Creating…' : 'Create workspace'}</button>
              </form>
            )}
          </section>
          <div className="flex items-start gap-3 rounded-xl border border-primary/15 bg-primary/[0.045] p-4">
            <ShieldCheck size={18} className="mt-0.5 shrink-0 text-primary" aria-hidden="true" />
            <div>
              <p className="text-sm font-semibold text-foreground">Separate space, focused teams</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">Each workspace has its own members, roles, and business data. Switch spaces without changing your account.</p>
            </div>
          </div>
        </aside>
      </div>
    </section>
  )
}
