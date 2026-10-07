import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../../core/api/client'
import { useWorkspace } from '../../../core/workspace/context'

export function MembersPage() {
  const queryClient = useQueryClient()
  const { activeWorkspaceId } = useWorkspace()
  const members = useQuery({ queryKey: ['workspace', activeWorkspaceId, 'members'], queryFn: api.workspaces.members, enabled: Boolean(activeWorkspaceId) })
  const users = useQuery({ queryKey: ['workspace', activeWorkspaceId, 'user-candidates'], queryFn: api.workspaces.candidates, enabled: Boolean(activeWorkspaceId) })
  const roles = useQuery({ queryKey: ['workspace', activeWorkspaceId, 'roles'], queryFn: api.workspaces.roles, enabled: Boolean(activeWorkspaceId) })
  const [selectedUser, setSelectedUser] = useState('')
  const [error, setError] = useState('')

  const add = async () => {
    if (!selectedUser) return
    try { await api.workspaces.addMember(selectedUser); setSelectedUser(''); await queryClient.invalidateQueries({ queryKey: ['workspace', activeWorkspaceId, 'members'] }) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to add member') }
  }

  const changeStatus = async (id: string, status: string) => { await api.workspaces.updateMember(id, status); await queryClient.invalidateQueries({ queryKey: ['workspace', activeWorkspaceId, 'members'] }) }
  const setRole = async (memberID: string, roleID: string) => { await api.workspaces.replaceMemberRoles(memberID, roleID ? [roleID] : []); setError(''); }
  const remove = async (id: string) => { if (!window.confirm('Remove this member from the workspace?')) return; await api.workspaces.removeMember(id); await queryClient.invalidateQueries({ queryKey: ['workspace', activeWorkspaceId, 'members'] }) }

  return <div><p className="mb-3 text-sm font-medium text-primary">Settings</p><h1 className="text-3xl font-semibold tracking-tight">Members</h1><p className="mt-3 text-muted-foreground">Manage who can work in this workspace and what they can access.</p><div className="mt-8 rounded-2xl border border-border bg-card p-6"><div className="flex flex-col gap-3 md:flex-row"><select value={selectedUser} onChange={event => setSelectedUser(event.target.value)} className="flex-1 rounded-lg border border-border bg-background px-3 py-3 text-sm text-card-foreground outline-none focus:border-primary"><option value="">Select an existing user…</option>{(users.data ?? []).filter(user => !(members.data ?? []).some(member => member.user_id === user.id)).map(user => <option key={user.id} value={user.id}>{user.display_name} · {user.email}</option>)}</select><button onClick={() => void add()} className="rounded-lg bg-primary px-5 py-3 text-sm font-semibold text-primary-foreground">Add member</button></div>{error && <p className="mt-3 text-sm text-destructive">{error}</p>}</div><div className="mt-4 overflow-hidden rounded-2xl border border-border bg-card"><div className="hidden grid-cols-[1.3fr_1.3fr_1fr_1fr_0.7fr] gap-4 border-b border-border px-6 py-4 text-xs uppercase tracking-wider text-muted-foreground/70 md:grid"><span>Name</span><span>Email</span><span>Status</span><span>Role</span><span /></div>{(members.data ?? []).map(member => <div key={member.id} className="grid gap-3 border-b border-border px-6 py-5 last:border-0 md:grid-cols-[1.3fr_1.3fr_1fr_1fr_0.7fr] md:items-center md:gap-4"><div><p className="font-medium text-card-foreground">{member.display_name}</p><p className="text-xs text-muted-foreground/70 md:hidden">{member.email}</p></div><span className="hidden text-sm text-muted-foreground md:block">{member.email}</span><select value={member.status} onChange={event => void changeStatus(member.id, event.target.value)} className="w-fit rounded-lg border border-border bg-background px-2 py-2 text-xs text-card-foreground"><option value="active">Active</option><option value="suspended">Suspended</option></select><select defaultValue="" onChange={event => void setRole(member.id, event.target.value)} className="w-fit rounded-lg border border-border bg-background px-2 py-2 text-xs text-card-foreground"><option value="">Assign role…</option>{(roles.data ?? []).map(role => <option key={role.id} value={role.id}>{role.display_name}</option>)}</select><button onClick={() => void remove(member.id)} className="text-left text-xs text-destructive hover:text-destructive">Remove</button></div>)}{members.data?.length === 0 && <div className="px-6 py-12 text-center text-sm text-muted-foreground">No members yet.</div>}</div></div>
}
