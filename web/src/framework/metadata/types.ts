export type HealthResponse = { status: 'ok'; service: string }
export type DependencyStatus = { postgres: 'ok' | 'unavailable' }
export type ReadinessResponse = { status: 'ready' | 'not_ready'; dependencies: DependencyStatus }
export type ModuleMetadata = { name: string; display_name: string; version: string; dependencies: string[] }
export type FieldMetadata = { name: string; display_name: string; type: string; required: boolean; read_only: boolean; description?: string }
export type EntityMetadata = { name: string; display_name: string; module: string; fields: FieldMetadata[]; capabilities: string[] }
export type PermissionMetadata = { name: string; module: string; scope: 'platform' | 'workspace'; display_name: string; description?: string }
