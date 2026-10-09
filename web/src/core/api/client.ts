import type { ApplicationMetadata, EntityMetadata, HealthResponse, ModuleMetadata, PermissionMetadata, ReadinessResponse } from '../../framework/metadata/types'
import type { AuthResponse, CurrentUser, Permission } from '../../framework/auth/types'

const baseURL = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
const sessionExpiryReloadKey = 'apexvoid.session_expiry_reload'

function reloadAfterSessionExpiry(path: string) {
  const bootstrapAuthPaths = ['/api/v1/auth/login', '/api/v1/auth/me', '/api/v1/auth/refresh']
  if (bootstrapAuthPaths.includes(path) || window.sessionStorage.getItem(sessionExpiryReloadKey)) return
  window.sessionStorage.setItem(sessionExpiryReloadKey, '1')
  window.location.reload()
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const workspaceID = window.localStorage.getItem('apexvoid.active_workspace')
  if (workspaceID) headers.set('X-ApexVoid-Workspace', workspaceID)
  headers.set('Accept', 'application/json')
  if (options.body && !(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  const response = await fetch(`${baseURL}${path}`, { ...options, headers, credentials: 'include' })
  if (!response.ok) {
    if (response.status === 401) reloadAfterSessionExpiry(path)
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

export type ExternalIntegration = { id: string; deployment: 'external'; display_name: string; description: string; version: string; api_contract_version: string; service_identity: string; service_endpoint: string; health_endpoint: string; frontend_route: string; settings_route: string; enabled: boolean; workspace_enabled?: boolean; credential_revoked: boolean; status: 'active'; access_match: string; access_permissions: string[]; permissions: Array<{ name: string; display_name: string; description: string; scope: 'platform' | 'workspace' }>; health: 'healthy' | 'unhealthy' | 'unavailable' | 'invalid' | 'unknown' }
export type ExternalIntegrationInput = { id: string; display_name: string; description: string; version: string; api_contract_version: 'v1'; service_identity: string; service_endpoint: string; health_endpoint: string; frontend_route: string; settings_route?: string; access_match: 'all' | 'any'; access_permissions: string[]; permissions: Array<{ name: string; display_name: string; description?: string; scope: 'platform' | 'workspace' }> }

export const api = {
  health: () => request<HealthResponse>('/health'),
  readiness: () => request<ReadinessResponse>('/ready'),
  framework: {
    applications: () => request<ApplicationMetadata[]>('/api/v1/framework/applications'),
    modules: () => request<ModuleMetadata[]>('/api/v1/framework/modules'),
    entities: () => request<EntityMetadata[]>('/api/v1/framework/entities'),
    permissions: () => request<PermissionMetadata[]>('/api/v1/framework/permissions'),
  },
  integrations: {
    external: (workspaceID?: string) => request<ExternalIntegration[]>(`/api/v1/applications/external${workspaceID ? `?workspace_id=${encodeURIComponent(workspaceID)}` : ''}`),
    register: (input: ExternalIntegrationInput) => request<ExternalIntegration & { service_credential: string }>('/api/v1/applications/external', json(input)),
    update: (id: string, input: ExternalIntegrationInput) => request<ExternalIntegration>(`/api/v1/applications/external/${id}`, { method: 'PATCH', body: JSON.stringify(input) }),
    retire: (id: string) => request<void>(`/api/v1/applications/external/${id}`, { method: 'DELETE' }),
    setWorkspaceAvailability: (id: string, workspaceID: string, enabled: boolean) => request<{ enabled: boolean }>(`/api/v1/applications/external/${id}/workspaces/${workspaceID}`, { method: 'PUT', body: JSON.stringify({ enabled }) }),
    revokeCredential: (id: string) => request<void>(`/api/v1/applications/external/${id}/credentials/revoke`, { method: 'POST' }),
    rotateCredential: (id: string) => request<{ service_credential: string }>(`/api/v1/applications/external/${id}/credentials/rotate`, { method: 'POST' }),
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
  contacts: {
    list: (params: { search?: string; kind?: string; status?: string; tag_id?: string; page?: number; limit?: number; sort?: string; desc?: boolean }) => { const query = new URLSearchParams(); Object.entries(params).forEach(([key, value]) => { if (value !== undefined && value !== '') query.set(key, String(value)) }); return request<ContactList>(`/api/v1/contacts?${query}`) },
    get: (id: string) => request<Contact>(`/api/v1/contacts/${id}`),
    create: (body: ContactInput) => request<Contact>('/api/v1/contacts', json(body)),
    update: (id: string, body: ContactInput) => request<Contact>(`/api/v1/contacts/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    archive: (id: string) => request<void>(`/api/v1/contacts/${id}/archive`, { method: 'POST' }),
    restore: (id: string) => request<void>(`/api/v1/contacts/${id}/restore`, { method: 'POST' }),
    tags: () => request<Tag[]>('/api/v1/contacts/tags'),
    createTag: (body: { name: string; color: string }) => request<Tag>('/api/v1/contacts/tags', json(body)),
    updateTag: (id: string, body: { name: string; color: string; active: boolean }) => request<Tag>(`/api/v1/contacts/tags/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    fields: () => request<CustomField[]>('/api/v1/contacts/fields'),
    createField: (body: Omit<CustomField, 'id' | 'workspace_id' | 'created_at' | 'updated_at' | 'active'>) => request<CustomField>('/api/v1/contacts/fields', json(body)),
    updateField: (id: string, body: Partial<CustomField>) => request<CustomField>(`/api/v1/contacts/fields/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    notes: (id: string) => request<Note[]>(`/api/v1/contacts/${id}/notes`),
    createNote: (id: string, content: string) => request<Note>(`/api/v1/contacts/${id}/notes`, json({ content })),
    updateNote: (contactID: string, noteID: string, content: string) => request<Note>(`/api/v1/contacts/${contactID}/notes/${noteID}`, { method: 'PATCH', body: JSON.stringify({ content }) }),
    activities: (id?: string, params: { status?: string; page?: number; limit?: number; assigned_user_id?: string } = {}) => { const query = new URLSearchParams(); Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) }); return request<Activity[] | ActivityList>(id ? `/api/v1/contacts/${id}/activities?${query}` : `/api/v1/activities?${query}`) },
    createActivity: (body: ActivityInput) => request<Activity>('/api/v1/activities', json(body)),
    completeActivity: (id: string) => request<Activity>(`/api/v1/activities/${id}/complete`, { method: 'POST' }),
    cancelActivity: (id: string) => request<Activity>(`/api/v1/activities/${id}/cancel`, { method: 'POST' }),
    relationships: (id: string) => request<Relationship[]>(`/api/v1/contacts/${id}/relationships`),
    tagsFor: (id: string) => request<string[]>(`/api/v1/contacts/${id}/tags`),
    replaceTags: (id: string, tagIDs: string[]) => request<void>(`/api/v1/contacts/${id}/tags`, { method: 'PUT', body: JSON.stringify(tagIDs) }),
    attachments: (id: string) => request<Attachment[]>(`/api/v1/contacts/${id}/attachments`),
    upload: (id: string, file: File) => { const body = new FormData(); body.append('file', file); return request<Attachment>(`/api/v1/contacts/${id}/attachments`, { method: 'POST', body }) },
    download: async (id: string, attachmentID: string) => { const workspaceID = window.localStorage.getItem('apexvoid.active_workspace'); const headers = new Headers(); if (workspaceID) headers.set('X-ApexVoid-Workspace', workspaceID); const response = await fetch(`${baseURL}/api/v1/contacts/${id}/attachments/${attachmentID}`, { headers, credentials: 'include' }); if (!response.ok) throw new Error(`Download failed with status ${response.status}`); return response.blob() },
    removeAttachment: (id: string, attachmentID: string) => request<void>(`/api/v1/contacts/${id}/attachments/${attachmentID}`, { method: 'DELETE' }),
  },
  crm: {
    pipelines: () => request<Pipeline[]>('/api/v1/crm/pipelines'),
    allPipelines: () => request<Pipeline[]>('/api/v1/crm/pipelines?include_archived=true'),
    stages: (pipelineID: string) => request<Stage[]>(`/api/v1/crm/pipelines/${pipelineID}/stages`),
    templates: () => request<PipelineTemplate[]>('/api/v1/crm/pipeline-templates'),
    initializePipeline: (template: string) => request<Pipeline>('/api/v1/crm/pipelines/initialize', json({ template })),
    createPipeline: (body: { name: string; description: string; color: string }) => request<Pipeline>('/api/v1/crm/pipelines', json(body)),
    updatePipeline: (id: string, body: { name: string; description: string; color: string; version: number }) => request<Pipeline>(`/api/v1/crm/pipelines/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    clonePipeline: (id: string, name: string) => request<Pipeline>(`/api/v1/crm/pipelines/${id}/clone`, json({ name })),
    archivePipeline: (id: string, version: number) => request<void>(`/api/v1/crm/pipelines/${id}/archive`, json({ version })),
    restorePipeline: (id: string, version: number) => request<void>(`/api/v1/crm/pipelines/${id}/restore`, json({ version })),
    setDefaultPipeline: (id: string) => request<void>(`/api/v1/crm/pipelines/${id}/set-default`, { method: 'POST' }),
    addStage: (pipelineID: string, body: { key: string; name: string; probability: number }) => request<Stage>(`/api/v1/crm/pipelines/${pipelineID}/stages`, json(body)),
    updateStage: (pipelineID: string, id: string, body: { name: string; description: string; color: string; probability: number; version: number }) => request<Stage>(`/api/v1/crm/pipelines/${pipelineID}/stages/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    archiveStage: (pipelineID: string, id: string) => request<void>(`/api/v1/crm/pipelines/${pipelineID}/stages/${id}/archive`, { method: 'POST' }),
    restoreStage: (pipelineID: string, id: string) => request<void>(`/api/v1/crm/pipelines/${pipelineID}/stages/${id}/restore`, { method: 'POST' }),
    reorderStages: (pipelineID: string, stage_ids: string[]) => request<void>(`/api/v1/crm/pipelines/${pipelineID}/stages/reorder`, json({ stage_ids })),
    leads: (params: Record<string, string | number | boolean | undefined> = {}) => request<CRMList<Lead>>(`/api/v1/crm/leads?${crmQuery(params)}`),
    lead: (id: string) => request<Lead>(`/api/v1/crm/leads/${id}`),
    createLead: (body: LeadInput) => request<Lead>('/api/v1/crm/leads', json(body)),
    updateLead: (id: string, body: LeadPatch) => request<Lead>(`/api/v1/crm/leads/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    leadTransition: (id: string, action: 'contact' | 'qualify' | 'disqualify' | 'reopen', body: { version: number; reason?: string }) => request<Lead>(`/api/v1/crm/leads/${id}/${action}`, json(body)),
    convertLead: (id: string, body: { pipeline_id: string; stage_id: string; contact_id?: string; create_contact: boolean; expected_revenue: string; currency: string; version: number }) => request<Opportunity>(`/api/v1/crm/leads/${id}/convert`, json(body)),
    leadHistory: (id: string) => request<LifecycleHistory[]>(`/api/v1/crm/leads/${id}/history`),
    opportunities: (params: Record<string, string | number | boolean | undefined> = {}) => request<CRMList<Opportunity>>(`/api/v1/crm/opportunities?${crmQuery(params)}`),
    opportunityBoard: (pipelineID: string, params: Record<string, string | number | boolean | undefined> = {}) => request<OpportunityBoard>(`/api/v1/crm/pipelines/${pipelineID}/board?${crmQuery(params)}`),
    opportunity: (id: string) => request<Opportunity>(`/api/v1/crm/opportunities/${id}`),
    createOpportunity: (body: OpportunityCreateInput) => request<OpportunityResponse>('/api/v1/crm/opportunities', json(body)),
    updateOpportunity: (id: string, body: OpportunityPatchInput) => request<OpportunityResponse>(`/api/v1/crm/opportunities/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    moveOpportunity: (id: string, body: { pipeline_id: string; stage_id: string; version: number }) => request<Opportunity>(`/api/v1/crm/opportunities/${id}/move-stage`, json(body)),
    closeOpportunity: (id: string, won: boolean, body: { version: number; reason?: string }) => request<Opportunity>(`/api/v1/crm/opportunities/${id}/${won ? 'mark-won' : 'mark-lost'}`, json(body)),
    reopenOpportunity: (id: string, body: { stage_id: string; version: number }) => request<Opportunity>(`/api/v1/crm/opportunities/${id}/reopen`, json(body)),
    opportunityHistory: (id: string) => request<LifecycleHistory[]>(`/api/v1/crm/opportunities/${id}/history`),
  },
  customization: {
    schema: (entity: string) => request<EffectiveSchema>(`/api/v1/customization/schema/${entity}`),
    fields: (entity: string) => request<RuntimeField[]>(`/api/v1/customization/fields/${entity}`),
    createField: (body: Omit<RuntimeField, 'id' | 'workspace_id' | 'created_at' | 'updated_at'>) => request<RuntimeField>('/api/v1/customization/fields', json(body)),
    updateField: (id: string, body: Partial<RuntimeField> & { section_id?: string | null }) => request<RuntimeField>(`/api/v1/customization/fields/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    sections: (entity: string) => request<FormSection[]>(`/api/v1/customization/sections/${entity}`),
    createSection: (body: Omit<FormSection, 'id' | 'workspace_id' | 'created_at' | 'updated_at'>) => request<FormSection>('/api/v1/customization/sections', json(body)),
    updateSection: (id: string, body: Omit<FormSection, 'id' | 'workspace_id' | 'created_at' | 'updated_at'>) => request<FormSection>(`/api/v1/customization/sections/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    views: (entity: string) => request<SavedView[]>(`/api/v1/customization/views/${entity}`),
    createView: (body: SavedViewInput) => request<SavedView>('/api/v1/customization/views', json(body)),
    updateView: (entity: string, id: string, body: SavedViewInput) => request<SavedView>(`/api/v1/customization/views/${entity}/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    deleteView: (entity: string, id: string) => request<void>(`/api/v1/customization/views/${entity}/${id}`, { method: 'DELETE' }),
  },
}

const crmQuery = (params: Record<string, string | number | boolean | undefined>) => { const query = new URLSearchParams(); Object.entries(params).forEach(([key, value]) => { if (value !== undefined && value !== '') query.set(key, String(value)) }); return query }

export type Role = { id: string; name: string; display_name: string; description: string; system: boolean; permissions: string[] }
export type Organization = { id: string; name: string; slug: string; status: string; created_at: string; updated_at: string }
export type Workspace = { id: string; organization_id: string; name: string; slug: string; timezone: string; status: string; created_at: string; updated_at: string }
export type Membership = { id: string; workspace_id: string; user_id: string; email: string; display_name: string; status: string; created_at: string; updated_at: string }
export type WorkspaceRole = { id: string; workspace_id?: string; name: string; display_name: string; description: string; system: boolean; permissions: string[] }
export type SetupStatus = { required: boolean; organization: boolean; workspace: boolean }
export type SetupResponse = { organization: Organization; workspace: Workspace; membership: Membership }
export type WorkspaceContextResponse = { organization: Organization; workspace: Workspace; permissions: string[] }
export type Contact = { id: string; workspace_id: string; kind: 'person' | 'company'; display_name: string; email: string; phone: string; website: string; description: string; status: 'active' | 'archived'; custom_values: Record<string, unknown>; created_at: string; updated_at: string }
export type ContactInput = Pick<Contact, 'kind' | 'display_name' | 'email' | 'phone' | 'website' | 'description' | 'custom_values'>
export type ContactList = { items: Contact[]; page: number; limit: number; total: number }
export type Tag = { id: string; workspace_id: string; name: string; color: string; active: boolean; created_at: string; updated_at: string }
export type Note = { id: string; workspace_id: string; contact_id: string; author_user_id: string; content: string; created_at: string; updated_at: string }
export type Activity = { id: string; workspace_id: string; title: string; description: string; activity_type: string; related_contact_id?: string; assigned_user_id: string; due_at?: string; status: string; completed_at?: string; created_at: string; updated_at: string }
export type ActivityList = { items: Activity[]; page: number; limit: number; total: number }
export type ActivityInput = Pick<Activity, 'title' | 'description' | 'activity_type' | 'assigned_user_id'> & { related_contact_id?: string; due_at?: string; status?: string }
export type Relationship = { id: string; person_id: string; company_id: string; relationship_type: string; job_title: string; is_primary: boolean; person_name?: string; company_name?: string }
export type Attachment = { id: string; contact_id: string; file_name: string; size: number; content_type: string; uploaded_by: string; created_at: string }
export type CustomField = { id: string; workspace_id: string; entity: string; key: string; label: string; type: string; description: string; required: boolean; options: string[]; display_order: number; active: boolean; created_at: string; updated_at: string }
export type Pipeline = { id: string; workspace_id: string; name: string; slug: string; description: string; color: string; status: 'active' | 'archived'; default: boolean; version: number }
export type Stage = { id: string; pipeline_id: string; key: string; name: string; description: string; color: string; category: 'open' | 'won' | 'lost'; probability: number; position: number; active: boolean; version: number }
export type PipelineTemplate = { key: string; name: string; description: string }
export type CRMList<T> = { items: T[]; page: number; limit: number; total: number }
export type Lead = { id: string; title: string; description: string; contact_name: string; company_name: string; email: string; phone: string; source: string; assigned_user_id?: string; contact_id?: string; status: 'new' | 'contacted' | 'qualified' | 'converted' | 'disqualified'; disqualification_reason?: string; custom_values: Record<string, unknown>; converted_opportunity_id?: string; version: number; created_at: string }
export type LeadInput = Omit<Lead, 'id' | 'status' | 'converted_opportunity_id' | 'version' | 'created_at'>
export type LeadPatch = Partial<LeadInput> & { version: number }
export type Opportunity = { id: string; title: string; description: string; pipeline_id: string; stage_id: string; contact_id?: string; company_id?: string; assigned_user_id?: string; expected_revenue: string; currency: string; expected_close_date?: string; outcome: 'open' | 'won' | 'lost'; loss_reason?: string; custom_values: Record<string, unknown>; original_lead_id?: string; version: number; created_at: string; closed_at?: string }
export type OpportunityBoardStage = { stage: Stage; count: number; totals_by_currency: Record<string, string>; items: Opportunity[]; next_cursor?: string | null }
export type OpportunityBoard = { pipeline: Pipeline; stages: OpportunityBoardStage[]; filter: { search: string; view_id?: string; owner_id?: string; outcome: string; stage_id?: string; limit: number } }
export type OpportunityResponse = Opportunity
export type OpportunityCreateInput = Omit<Opportunity, 'id' | 'outcome' | 'loss_reason' | 'original_lead_id' | 'version' | 'created_at' | 'closed_at'>
export type OpportunityPatchInput = Partial<Omit<OpportunityCreateInput, 'pipeline_id' | 'stage_id' | 'expected_close_date'>> & { expected_close_date?: string | null; version: number }
export type OpportunityInput = OpportunityCreateInput
export type OpportunityPatch = OpportunityPatchInput
export type LifecycleHistory = { id: string; workspace_id: string; lead_id?: string; opportunity_id?: string; event_type: string; from_stage_id?: string; to_stage_id?: string; actor_id: string; created_at: string }
export type EffectiveField = { key: string; label: string; type: 'string' | 'text' | 'boolean' | 'integer' | 'decimal' | 'date' | 'enum'; description: string; required: boolean; read_only: boolean; source: 'built_in' | 'custom'; default_value?: unknown; options: string[]; visible: boolean; display_order: number; section_id?: string }
export type EffectiveSchema = { entity: string; fields: EffectiveField[]; sections: { id: string; name: string; description: string; display_order: number }[] }
export type RuntimeField = { id: string; workspace_id: string; entity: string; key: string; label: string; type: EffectiveField['type']; description: string; required: boolean; default_value?: unknown; options: string[]; visible: boolean; display_order: number; section_id?: string; active: boolean; created_at: string; updated_at: string }
export type FormSection = { id: string; workspace_id: string; entity: string; name: string; description: string; display_order: number; created_at: string; updated_at: string }
export type SavedView = { id: string; workspace_id: string; entity: string; owner_user_id: string; name: string; shared: boolean; filters: { field: string; operator: string; value?: unknown }[]; columns: string[]; sort_field: string; sort_direction: string; created_at: string; updated_at: string }
export type SavedViewInput = Omit<SavedView, 'id' | 'workspace_id' | 'owner_user_id' | 'created_at' | 'updated_at'>
