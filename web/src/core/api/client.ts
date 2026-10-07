import type { EntityMetadata, HealthResponse, ModuleMetadata, PermissionMetadata, ReadinessResponse } from '../../framework/metadata/types'
import type { AuthResponse, CurrentUser, Permission } from '../../framework/auth/types'

const baseURL = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
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
    changePassword: (current_password: string, new_password: string) => request<void>('/api/v1/auth/change-password', json({ current_password, new_password })),
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
