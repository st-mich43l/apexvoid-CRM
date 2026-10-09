import { Boxes, ExternalLink, Settings2, TriangleAlert } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useOutletContext } from 'react-router-dom'
import { useState } from 'react'
import { api } from '../../../core/api/client'
import { Badge, Button, Card, EmptyState, ErrorState, LoadingState, PageContainer, PageHeader } from '../../../components/ui'
import { ConfirmDialog } from '../../../components/ConfirmDialog'
import type { ApplicationMetadata } from '../../../framework/metadata/types'
import type { FrontendApplication } from '../../../framework/module/types'
import { ModuleRegistry, type ApplicationContract } from '../../../framework/module/registry'
import { useAuth } from '../../../core/auth/context'
import { useWorkspace } from '../../../core/workspace/context'
import { userWorkspaceQueryKey } from '../../../core/workspace/query'
import type { ExternalIntegration, ExternalIntegrationInput } from '../../../core/api/client'
import { ExternalIntegrationEditor } from './ExternalIntegrationEditor'
import { ExternalEnrollmentWizard } from './ExternalEnrollmentWizard'
import { ApplicationDialog } from './ApplicationDialog'

type ShellContext = { applications: FrontendApplication[] }

export function ApplicationsPage() {
  const { applications: compiled } = useOutletContext<ShellContext>()
  const { user, platformCan } = useAuth()
  const { activeWorkspaceId } = useWorkspace()
  const canManage = platformCan('core.application.manage')
  const registered = useQuery({ queryKey: userWorkspaceQueryKey(user?.id, activeWorkspaceId, 'framework', 'applications'), queryFn: api.framework.applications, enabled: Boolean(user && activeWorkspaceId), retry: false })
  const management = useQuery({ queryKey: userWorkspaceQueryKey(user?.id, activeWorkspaceId, 'external-integrations'), queryFn: () => api.integrations.external(activeWorkspaceId ?? undefined), enabled: Boolean(canManage && user && activeWorkspaceId), retry: false })
  const contracts = registered.data ? contractReport(compiled, registered.data.filter(item => item.deployment !== 'external')) : []
  const external = registered.data?.filter(item => item.deployment === 'external') ?? []

  return <PageContainer variant="workspace"><PageHeader eyebrow="Platform" title="Applications" description="Internal modules and trusted external services registered with ApexVoid. The platform controls availability and access; it never installs arbitrary software." />
    {registered.isLoading && <Card><LoadingState label="Loading registered applications…" /></Card>}
    {registered.isError && <ErrorState message="Unable to load application discovery metadata." onRetry={() => void registered.refetch()} />}
    {!registered.isLoading && !registered.isError && contracts.length + external.length === 0 && <Card><EmptyState title="No applications registered" description="This runtime currently exposes platform services only." /></Card>}
    <div className="grid gap-4 lg:grid-cols-2">{contracts.map(contract => <ApplicationCard key={contract.id} contract={contract} />)}</div>
    {external.length > 0 && <div className="mt-4 grid gap-4 lg:grid-cols-2">{external.map(application => <ExternalApplicationCard key={application.id} application={application} />)}</div>}
    {canManage && <IntegrationManagement integrations={management.data ?? []} loading={management.isLoading} workspaceID={activeWorkspaceId ?? ''} />}
  </PageContainer>
}

function IntegrationManagement({ integrations, loading, workspaceID }: { integrations: ExternalIntegration[]; loading: boolean; workspaceID: string }) {
  const client = useQueryClient()
  const [confirmation, setConfirmation] = useState<null | { title: string; description: string; label: string; destructive?: boolean; confirm: () => Promise<void> }>(null)
  const [rotatedCredential, setRotatedCredential] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [editor, setEditor] = useState<ExternalIntegration | null>(null)
  const [enrollment, setEnrollment] = useState(false)
  const refresh = () => client.invalidateQueries({ predicate: query => query.queryKey.includes('applications') || query.queryKey.includes('external-integrations') })
  const availability = useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => api.integrations.setWorkspaceAvailability(id, workspaceID, enabled), onSuccess: refresh })
  const revoke = useMutation({ mutationFn: api.integrations.revokeCredential, onSuccess: refresh })
  const rotate = useMutation({ mutationFn: api.integrations.rotateCredential, onSuccess: refresh })
  const update = useMutation({ mutationFn: ({ id, input }: { id: string; input: ExternalIntegrationInput }) => api.integrations.update(id, input), onSuccess: refresh })
  const retire = useMutation({ mutationFn: api.integrations.retire, onSuccess: refresh })
  const execute = async (action: () => Promise<unknown>) => {
    setActionError(null)
    try { await action(); return true } catch (error) { setActionError(error instanceof Error ? error.message : 'The integration action could not be completed.'); return false }
  }
  const requestAvailability = (item: ExternalIntegration, enabled: boolean) => {
    const apply = async () => { if (await execute(() => availability.mutateAsync({ id: item.id, enabled }))) setConfirmation(null) }
    if (enabled) { void apply(); return }
    setConfirmation({ title: `Disable ${item.display_name}?`, description: 'Users in this workspace will immediately lose gateway access to this service. You can enable it again later.', label: 'Disable service', destructive: true, confirm: apply })
  }
  const requestRotate = (item: ExternalIntegration) => setConfirmation({ title: `Rotate ${item.display_name} credential?`, description: 'The current service credential will stop working immediately. Update the service secret with the new value shown once after rotation.', label: 'Rotate credential', destructive: true, confirm: async () => { let result: { service_credential: string } | undefined; if (await execute(async () => { result = await rotate.mutateAsync(item.id) })) { setRotatedCredential(result?.service_credential ?? null); setConfirmation(null) } } })
  const requestRevoke = (item: ExternalIntegration) => setConfirmation({ title: `Revoke ${item.display_name} credential?`, description: 'This permanently disables service-to-platform authentication. The credential cannot be restored; register a replacement service if needed.', label: 'Revoke credential', destructive: true, confirm: async () => { if (await execute(() => revoke.mutateAsync(item.id))) setConfirmation(null) } })
  const requestRetire = (item: ExternalIntegration) => setConfirmation({ title: `Retire ${item.display_name}?`, description: 'The application will be disabled, its credential revoked, and its owned permissions retired permanently. Existing grants will not be rebound to another application.', label: 'Retire application', destructive: true, confirm: async () => { if (await execute(() => retire.mutateAsync(item.id))) setConfirmation(null) } })
  const saveEditor = async (input: ExternalIntegrationInput) => {
    if (editor && await execute(() => update.mutateAsync({ id: editor.id, input }))) setEditor(null)
  }
  return <section className="mt-7"><PageHeader eyebrow="Administrator" title="External integration management" description="Discover a service from its manifest, review its requested access and database plan, then activate it only in selected workspaces." actions={<Button onClick={() => { setActionError(null); setEnrollment(true); setEditor(null) }}>Register application</Button>} />
    {enrollment && <ApplicationDialog title="Register an application" description="Connect a trusted Docker application and discover its identity, version, API contract and permissions automatically." onClose={() => setEnrollment(false)}><ExternalEnrollmentWizard onComplete={() => { setEnrollment(false); refresh() }} /></ApplicationDialog>}
    {editor && <ApplicationDialog title={`Edit ${editor.display_name}`} description="Update connection details. Application version and API contract are owned by the external service." onClose={() => { if (!update.isPending) setEditor(null) }} busy={update.isPending}><ExternalIntegrationEditor key={editor.id} initial={editor} pending={update.isPending} error={actionError ?? undefined} onCancel={() => setEditor(null)} onSubmit={saveEditor} /></ApplicationDialog>}
    {actionError && <p role="alert" className="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{actionError}</p>}
    {rotatedCredential && <Card className="mb-4 border-warning/30 bg-warning/5 p-4"><p className="font-medium">Copy the replacement credential now</p><p className="mt-1 text-sm text-muted-foreground">It is shown only for this lifecycle action. Store it in the service secret before closing this message.</p><code className="mt-3 block select-all overflow-x-auto rounded-md bg-muted p-3 text-xs">{rotatedCredential}</code><div className="mt-3 flex gap-2"><Button variant="outline" onClick={() => void navigator.clipboard?.writeText(rotatedCredential)}>Copy credential</Button><Button variant="outline" onClick={() => setRotatedCredential(null)}>I stored this credential</Button></div></Card>}
    {loading ? <Card><LoadingState label="Loading external integrations…" /></Card> : integrations.length === 0 ? <Card><EmptyState title="No external services registered" description="Trusted services can be registered through the platform integration API." /></Card> : <div className="grid gap-4 lg:grid-cols-2">{integrations.map(item => { const enabled = item.workspace_enabled ?? item.enabled; return <Card key={item.id} className="p-5"><div className="flex items-start justify-between gap-4"><div><h2 className="font-semibold">{item.display_name}</h2><p className="mt-1 text-xs text-muted-foreground">{item.service_identity} · {item.version}</p></div><Badge tone={item.health === 'healthy' ? 'success' : 'warning'}>{item.health}</Badge></div><div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground"><span className="rounded-md bg-muted px-2 py-1">{item.permissions.length} permissions</span><span className="rounded-md bg-muted px-2 py-1">API {item.api_contract_version}</span><span className="rounded-md bg-muted px-2 py-1">{enabled ? 'Enabled here' : 'Disabled here'}</span>{item.credential_revoked && <span className="rounded-md bg-destructive/10 px-2 py-1 text-destructive">Credential revoked</span>}</div><p className="mt-3 text-xs text-muted-foreground">{item.service_endpoint}</p><div className="mt-5 flex flex-wrap gap-2"><Button variant="outline" disabled={!workspaceID || availability.isPending} onClick={() => requestAvailability(item, !enabled)}>{enabled ? 'Disable in workspace' : 'Enable in workspace'}</Button><Button variant="outline" onClick={() => setEditor(item)}>Edit metadata</Button><Button variant="outline" disabled={item.credential_revoked || rotate.isPending} onClick={() => requestRotate(item)}>Rotate credential</Button><Button variant="outline" disabled={item.credential_revoked || revoke.isPending} onClick={() => requestRevoke(item)}>Revoke credential</Button><Button variant="destructive" disabled={retire.isPending} onClick={() => requestRetire(item)}>Retire</Button></div></Card> })}</div>}
    {confirmation && <ConfirmDialog open title={confirmation.title} description={confirmation.description} confirmLabel={confirmation.label} destructive={confirmation.destructive} onCancel={() => setConfirmation(null)} onConfirm={confirmation.confirm} />}
  </section>
}

function ExternalApplicationCard({ application }: { application: ApplicationMetadata }) {
  const available = application.entry_authorized
  return <Card className="flex flex-col p-5"><div className="flex items-start justify-between gap-4"><div className="flex min-w-0 items-start gap-3"><span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><Boxes size={19} /></span><div className="min-w-0"><h2 className="font-semibold text-card-foreground">{application.display_name}</h2><p className="mt-1 text-xs text-muted-foreground">External service · v{application.version}</p></div></div><Badge tone={available ? 'success' : 'neutral'}>{available ? 'Available' : 'Not authorized'}</Badge></div><p className="mt-4 text-sm leading-6 text-muted-foreground">{application.description}</p><div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground"><span className="rounded-md bg-muted px-2 py-1">API contract: {application.api_contract_version}</span><span className="rounded-md bg-muted px-2 py-1">Gateway managed</span></div>{available && application.frontend.entry_route && <div className="mt-5"><Link to={application.frontend.entry_route} className="inline-flex min-h-9 items-center gap-2 rounded-lg bg-primary px-3 text-sm font-medium text-primary-foreground hover:opacity-90">Open application <ExternalLink size={14} /></Link></div>}</Card>
}

function contractReport(compiled: FrontendApplication[], registered: ApplicationMetadata[]): ApplicationContract[] {
  const modules = compiled.map(application => ({ name: `compiled-${application.id}`, version: '1.0.0', application, routes: [{ id: `${application.id}-route`, path: application.entryRoute, element: null }], navigation: [{ id: application.navigationID, label: application.id, path: application.entryRoute, order: 0, icon: Boxes }] }))
  return new ModuleRegistry(modules).applicationContracts(registered)
}

function ApplicationCard({ contract }: { contract: ApplicationContract }) {
  const application = contract.application
  const compatible = contract.status === 'compatible'
  const entryAuthorized = application?.entry_authorized === true
  const settingsAuthorized = application?.settings_authorized === true
  const status = !compatible ? { tone: 'warning' as const, label: 'Contract issue' } : entryAuthorized ? { tone: 'success' as const, label: 'Available' } : { tone: 'neutral' as const, label: 'Not authorized' }
  return <Card className="flex flex-col p-5"><div className="flex items-start justify-between gap-4"><div className="flex min-w-0 items-start gap-3"><span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><Boxes size={19} /></span><div className="min-w-0"><h2 className="font-semibold text-card-foreground">{application?.display_name ?? contract.id}</h2><p className="mt-1 text-xs text-muted-foreground">{application ? `v${application.version} · ${application.id}` : contract.id}</p></div></div><Badge tone={status.tone}>{status.label}</Badge></div>
    <p className="mt-4 text-sm leading-6 text-muted-foreground">{application?.description ?? contract.message}</p>
    {application && <div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground"><span className="rounded-md bg-muted px-2 py-1">Modules: {application.module_dependencies.join(', ')}</span><span className="rounded-md bg-muted px-2 py-1">API contract: {application.api_contract_version}</span></div>}
    {!compatible && <div className="mt-4 flex gap-2 rounded-lg border border-warning/20 bg-warning/10 p-3 text-xs leading-5 text-warning"><TriangleAlert size={16} className="mt-0.5 shrink-0" /><p>{contract.message}</p></div>}
    {compatible && !entryAuthorized && <div className="mt-4 rounded-lg border border-border bg-muted/40 p-3 text-xs leading-5 text-muted-foreground">You do not have the required permission in this workspace to open this application.</div>}
    {compatible && application && <div className="mt-5 flex flex-wrap gap-2">{entryAuthorized && <Link to={application.frontend.entry_route} className="inline-flex min-h-9 items-center gap-2 rounded-lg bg-primary px-3 text-sm font-medium text-primary-foreground hover:opacity-90">Open application <ExternalLink size={14} /></Link>}{application.settings && settingsAuthorized && <Link to={application.settings.route} className="inline-flex min-h-9 items-center gap-2 rounded-lg border border-border px-3 text-sm font-medium text-foreground hover:bg-muted"><Settings2 size={14} />Settings</Link>}</div>}
  </Card>
}
