export type HealthResponse = { status: 'ok'; service: string; environment: 'development' | 'production' | 'test' | string }
export type DependencyStatus = { postgres: 'ok' | 'unavailable' }
export type ReadinessResponse = { status: 'ready' | 'not_ready'; dependencies: DependencyStatus }
export type ModuleMetadata = { name: string; display_name: string; version: string; dependencies: string[] }
export type ApplicationMetadata = { id: string; display_name: string; description: string; version: string; api_contract_version: string; module_dependencies: string[]; required_permissions: string[]; required_capabilities: string[]; frontend: { entry_route: string; navigation_id: string }; settings?: { route: string }; entry_authorized: boolean; settings_authorized: boolean; deployment?: 'internal' | 'external'; frontend_external?: boolean; update_available?: boolean; available_version?: string; update_checked_at?: string; update_check_error?: string }
export type FieldMetadata = { name: string; display_name: string; type: string; required: boolean; read_only: boolean; description?: string }
export type EntityMetadata = { name: string; display_name: string; module: string; fields: FieldMetadata[]; capabilities: string[] }
export type PermissionMetadata = { name: string; module: string; scope: 'platform' | 'workspace'; display_name: string; description?: string }
