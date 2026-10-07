import type { EntityMetadata, HealthResponse, ModuleMetadata, PermissionMetadata, ReadinessResponse } from '../../framework/metadata/types'
import type { AuthResponse, CurrentUser, Permission } from '../../framework/auth/types'

const baseURL = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const workspaceID = window.localStorage.getItem('apexvoid.active_workspace')
  if (workspaceID) headers.set('X-ApexVoid-Workspace', workspaceID)
  headers.set('Accept', 'application/json')
  if (options.body) headers.set('Content-Type', 'application/json')
  const response = await fetch(`${baseURL}${path}`, { ...options, headers, credentials: 'include' })
  if (!response.ok) {
    let message = `API request failed with status ${response.status}`
    try { const body = await response.json() as { error?: { message?: string } }; message = body.error?.message ?? message } catch { /* keep status message */ }
    const error = new Error(message) as Error & { status?: number }
    error.status = response.status
    throw error
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

const json = (body: unknown): RequestInit => ({ method: 'POST', body: JSON.stringify(body) })

export const api = {
  health: () => request<HealthResponse>('/health'),
  readiness: () => request<ReadinessResponse>('/ready'),
  framework: {
    modules: () => request<ModuleMetadata[]>('/api/v1/framework/modules'),
    entities: () => request<EntityMetadata[]>('/api/v1/framework/entities'),
    permissions: () => request<PermissionMetadata[]>('/api/v1/framework/permissions'),
  },
  auth: {
    login: (email: string, password: string) => request<AuthResponse>('/api/v1/auth/login', json({ email, password })),
    refresh: () => request<AuthResponse>('/api/v1/auth/refresh', { method: 'POST' }),
    logout: () => request<void>('/api/v1/auth/logout', { method: 'POST' }),
    me: () => request<AuthResponse>('/api/v1/auth/me'),
    changePassword: (current_password: string, new_password: string) => request<AuthResponse>('/api/v1/auth/change-password', json({ current_password, new_password })),
  },
  setup: {
    status: () => request<SetupStatus>('/api/v1/setup/status'),
    createOrganization: (body: { organization_name: string; workspace_name?: string; timezone?: string }) => request<SetupResponse>('/api/v1/setup/organization', json(body)),
  },
  workspaces: {
    list: () => request<Workspace[]>('/api/v1/workspaces'),
    current: () => request<WorkspaceContextResponse>('/api/v1/workspace'),
    create: (body: { name: string; timezone?: string }) => request<Workspace>('/api/v1/workspaces', json(body)),
    update: (body: { name?: string; timezone?: string }) => request<Workspace>('/api/v1/workspace', { method: 'PATCH', body: JSON.stringify(body) }),
    members: () => request<Membership[]>('/api/v1/workspace/members'),
    candidates: () => request<CurrentUser[]>('/api/v1/workspace/user-candidates'),
    addMember: (user_id: string) => request<Membership>('/api/v1/workspace/members', json({ user_id })),
    updateMember: (id: string, status: string) => request<Membership>(`/api/v1/workspace/members/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) }),
    removeMember: (id: string) => request<void>(`/api/v1/workspace/members/${id}`, { method: 'DELETE' }),
    memberRoles: (id: string) => request<{ role_ids: string[] }>(`/api/v1/workspace/members/${id}/roles`),
    replaceMemberRoles: (id: string, role_ids: string[]) => request<{ role_ids: string[] }>(`/api/v1/workspace/members/${id}/roles`, { method: 'PUT', body: JSON.stringify({ role_ids }) }),
    roles: () => request<WorkspaceRole[]>('/api/v1/workspace/roles'),
    createRole: (body: { name: string; display_name: string; description?: string }) => request<WorkspaceRole>('/api/v1/workspace/roles', json(body)),
    replaceRolePermissions: (id: string, permissions: string[]) => request<{ status: string }>(`/api/v1/workspace/roles/${id}/permissions`, { method: 'PUT', body: JSON.stringify({ permissions }) }),
    organization: (body: { name?: string }) => request<Organization>('/api/v1/organization', { method: 'PATCH', body: JSON.stringify(body) }),
  },
  users: {
    list: () => request<CurrentUser[]>('/api/v1/users'),
    create: (body: { email: string; username?: string; display_name: string; password: string }) => request<CurrentUser>('/api/v1/users', { method: 'POST', body: JSON.stringify(body) }),
    update: (id: string, body: { username?: string; display_name?: string }) => request<CurrentUser>(`/api/v1/users/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    setStatus: (id: string, status: 'enable' | 'disable') => request<CurrentUser>(`/api/v1/users/${id}/${status}`, { method: 'POST' }),
    roles: (id: string) => request<{ role_ids: string[] }>(`/api/v1/users/${id}/roles`),
    replaceRoles: (id: string, role_ids: string[]) => request<{ role_ids: string[] }>(`/api/v1/users/${id}/roles`, { method: 'PUT', body: JSON.stringify({ role_ids }) }),
  },
  access: {
    roles: () => request<Role[]>('/api/v1/access/roles'),
    createRole: (body: { name: string; display_name: string; description?: string }) => request<Role>('/api/v1/access/roles', { method: 'POST', body: JSON.stringify(body) }),
    updateRole: (id: string, body: { display_name?: string; description?: string }) => request<Role>(`/api/v1/access/roles/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    deleteRole: (id: string) => request<void>(`/api/v1/access/roles/${id}`, { method: 'DELETE' }),
    replacePermissions: (id: string, permissions: string[]) => request<Role>(`/api/v1/access/roles/${id}/permissions`, { method: 'PUT', body: JSON.stringify({ permissions }) }),
    permissions: () => request<Permission[]>('/api/v1/access/permissions'),
  },
}

export type Role = { id: string; name: string; display_name: string; description: string; system: boolean; permissions: string[] }
export type Organization = { id: string; name: string; slug: string; status: string; created_at: string; updated_at: string }
export type Workspace = { id: string; organization_id: string; name: string; slug: string; timezone: string; status: string; created_at: string; updated_at: string }
export type Membership = { id: string; workspace_id: string; user_id: string; email: string; display_name: string; status: string; created_at: string; updated_at: string }
export type WorkspaceRole = { id: string; workspace_id?: string; name: string; display_name: string; description: string; system: boolean }
export type SetupStatus = { required: boolean; organization: boolean; workspace: boolean }
export type SetupResponse = { organization: Organization; workspace: Workspace; membership: Membership }
export type WorkspaceContextResponse = { organization: Organization; workspace: Workspace; permissions: string[] }
