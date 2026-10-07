import type { EntityMetadata, HealthResponse, ModuleMetadata, PermissionMetadata, ReadinessResponse } from '../../framework/metadata/types'

const baseURL = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, { headers: { Accept: 'application/json' } })
  if (!response.ok) throw new Error(`API request failed with status ${response.status}`)
  return response.json() as Promise<T>
}

export const api = {
  health: () => request<HealthResponse>('/health'),
  readiness: () => request<ReadinessResponse>('/ready'),
  framework: {
    modules: () => request<ModuleMetadata[]>('/api/v1/framework/modules'),
    entities: () => request<EntityMetadata[]>('/api/v1/framework/entities'),
    permissions: () => request<PermissionMetadata[]>('/api/v1/framework/permissions'),
  },
}
