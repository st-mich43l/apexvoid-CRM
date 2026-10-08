import { useEffect, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type Membership, type WorkspaceRole } from '../../../core/api/client'
import { useWorkspace } from '../../../core/workspace/context'
import { workspaceQueryKey } from '../../../core/workspace/query'
import { ConfirmDialog } from '../../../components/ConfirmDialog'

function MemberRoleEditor({ member, roles, onError }: { member: Membership; roles: WorkspaceRole[]; onError: (message: string) => void }) {
  const queryClient = useQueryClient()
  const assigned = useQuery({ queryKey: workspaceQueryKey(member.workspace_id, 'member-roles', member.id), queryFn: () => api.workspaces.memberRoles(member.id) })
  const [selected, setSelected] = useState<string[]>([])
  const [saved, setSaved] = useState(true)

  useEffect(() => {
    if (assigned.data) {
      setSelected(assigned.data.role_ids)
      setSaved(true)
    }
  }, [assigned.data])

  const toggle = (id: string) => {
    setSelected(current => current.includes(id) ? current.filter(item => item !== id) : [...current, id])
    setSaved(false)
  }

  const save = async () => {
    try {
      await api.workspaces.replaceMemberRoles(member.id, selected)
      await queryClient.invalidateQueries({ queryKey: workspaceQueryKey(member.workspace_id, 'member-roles', member.id) })
      setSaved(true)
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : 'Unable to update member roles'
      onError(message)
    }
  }

  if (assigned.isLoading) return <span className="text-xs text-muted-foreground">Loading roles…</span>
  return <div className="space-y-2">
    <div className="flex flex-wrap gap-1">{selected.length === 0 && <span className="text-xs text-muted-foreground">No roles assigned</span>}{selected.map(id => <span key={id} className="rounded-full bg-secondary px-2 py-1 text-xs text-secondary-foreground">{roles.find(role => role.id === id)?.display_name ?? 'Unknown role'}</span>)}</div>
    <details className="text-xs"><summary className="cursor-pointer text-primary">Edit roles</summary><div className="mt-2 space-y-2 rounded-lg border border-border bg-background p-2">{roles.map(role => <label key={role.id} className="flex items-center gap-2 text-card-foreground"><input type="checkbox" checked={selected.includes(role.id)} onChange={() => toggle(role.id)} />{role.display_name}{role.system && <span className="text-muted-foreground">(system)</span>}</label>)}<button type="button" disabled={saved} onClick={() => void save()} className="mt-1 rounded-md bg-primary px-2 py-1 font-semibold text-primary-foreground disabled:opacity-50">Save roles</button></div></details>
  </div>
}

type PendingAction = { type: 'remove' | 'suspend'; member: Membership } | null

export function MembersPage() {
  const queryClient = useQueryClient()
  const { activeWorkspaceId } = useWorkspace()
  const members = useQuery({ queryKey: workspaceQueryKey(activeWorkspaceId, 'members'), queryFn: api.workspaces.members, enabled: Boolean(activeWorkspaceId) })
  const users = useQuery({ queryKey: workspaceQueryKey(activeWorkspaceId, 'user-candidates'), queryFn: api.workspaces.candidates, enabled: Boolean(activeWorkspaceId) })
  const roles = useQuery({ queryKey: workspaceQueryKey(activeWorkspaceId, 'roles'), queryFn: api.workspaces.roles, enabled: Boolean(activeWorkspaceId) })
  const [selectedUser, setSelectedUser] = useState('')
  const [error, setError] = useState('')
  const [pendingAction, setPendingAction] = useState<PendingAction>(null)

  const invalidateMembers = () => queryClient.invalidateQueries({ queryKey: workspaceQueryKey(activeWorkspaceId, 'members') })
  const add = async () => {
    if (!selectedUser) return
    try { await api.workspaces.addMember(selectedUser); setSelectedUser(''); await invalidateMembers() } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to add member') }
  }

  const changeStatus = async (member: Membership, status: string) => {
    if (status === 'suspended' && member.status === 'active') {
      setPendingAction({ type: 'suspend', member })
      return
    }
    try { await api.workspaces.updateMember(member.id, status); await invalidateMembers() } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to update member') }
  }

  const confirmAction = async () => {
    if (!pendingAction) return
    try {
      if (pendingAction.type === 'remove') await api.workspaces.removeMember(pendingAction.member.id)
      else await api.workspaces.updateMember(pendingAction.member.id, 'suspended')
      await invalidateMembers()
      setPendingAction(null)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to update member')
      setPendingAction(null)
    }
  }

  return <div>
    <p className="mb-3 text-sm font-medium text-primary">Settings</p>
    <h1 className="text-3xl font-semibold tracking-tight">Members</h1>
    <p className="mt-3 text-muted-foreground">Manage who can work in this workspace and what they can access.</p>
    <div className="mt-8 rounded-2xl border border-border bg-card p-6">
      <div className="flex flex-col gap-3 md:flex-row"><select aria-label="Select a user to add" value={selectedUser} onChange={event => setSelectedUser(event.target.value)} className="flex-1 rounded-lg border border-border bg-background px-3 py-3 text-sm text-card-foreground outline-none focus:border-primary"><option value="">Select an existing user…</option>{(users.data ?? []).filter(user => !(members.data ?? []).some(member => member.user_id === user.id)).map(user => <option key={user.id} value={user.id}>{user.display_name} · {user.email}</option>)}</select><button onClick={() => void add()} disabled={!selectedUser} className="rounded-lg bg-primary px-5 py-3 text-sm font-semibold text-primary-foreground disabled:cursor-not-allowed disabled:opacity-50">Add member</button></div>
      {error && <p className="mt-3 text-sm text-destructive">{error}</p>}
    </div>
    <div className="mt-4 overflow-hidden rounded-2xl border border-border bg-card">
      <div className="hidden grid-cols-[1.1fr_1.2fr_0.8fr_1.6fr_0.5fr] gap-4 border-b border-border px-6 py-4 text-xs uppercase tracking-wider text-muted-foreground/70 md:grid"><span>Name</span><span>Email</span><span>Status</span><span>Roles</span><span /></div>
      {members.isLoading && <div className="px-6 py-12 text-center text-sm text-muted-foreground">Loading members…</div>}
      {(members.data ?? []).map(member => <div key={member.id} className="grid gap-3 border-b border-border px-6 py-5 last:border-0 md:grid-cols-[1.1fr_1.2fr_0.8fr_1.6fr_0.5fr] md:items-center md:gap-4"><div><p className="font-medium text-card-foreground">{member.display_name}</p><p className="text-xs text-muted-foreground/70 md:hidden">{member.email}</p></div><span className="hidden text-sm text-muted-foreground md:block">{member.email}</span><select value={member.status} onChange={event => void changeStatus(member, event.target.value)} className="w-fit rounded-lg border border-border bg-background px-2 py-2 text-xs text-card-foreground"><option value="active">Active</option><option value="suspended">Suspended</option></select><MemberRoleEditor member={member} roles={roles.data ?? []} onError={setError} /><button onClick={() => setPendingAction({ type: 'remove', member })} className="text-left text-xs text-destructive hover:text-destructive">Remove</button></div>)}
      {members.data?.length === 0 && <div className="px-6 py-12 text-center"><p className="font-medium text-foreground">No team members yet</p><p className="mt-2 text-sm text-muted-foreground">Select an existing user above to give them access to this workspace.</p></div>}
    </div>
    <ConfirmDialog open={pendingAction?.type === 'remove'} title="Remove workspace member?" description="This removes the member’s access to the current workspace. Their account and access to other workspaces are not affected." confirmLabel="Remove member" destructive onCancel={() => setPendingAction(null)} onConfirm={confirmAction} />
    <ConfirmDialog open={pendingAction?.type === 'suspend'} title="Suspend workspace member?" description="The member will lose access to this workspace until they are activated again. This action is blocked when it would remove the last workspace administrator." confirmLabel="Suspend member" destructive onCancel={() => setPendingAction(null)} onConfirm={confirmAction} />
  </div>
}
