import { Boxes, ExternalLink, Settings2, TriangleAlert } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useContext } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { api } from '../../../core/api/client'
import { Badge, Button, Card, EmptyState, ErrorState, LoadingState, PageContainer, PageHeader } from '../../../components/ui'
import type { ApplicationMetadata } from '../../../framework/metadata/types'
import type { FrontendApplication } from '../../../framework/module/types'
import { ModuleRegistry, type ApplicationContract } from '../../../framework/module/registry'
import { AuthContext } from '../../../core/auth/context'
import type { ExternalIntegration } from '../../../core/api/client'

type ShellContext = { applications: FrontendApplication[] }

export function ApplicationsPage() {
  const { applications: compiled } = useOutletContext<ShellContext>()
  // Application discovery is workspace-specific. Including the selected ID
  // prevents a response from one tenant being reused after a workspace switch.
  const workspaceID = window.localStorage.getItem('apexvoid.active_workspace') ?? ''
  const auth = useContext(AuthContext)
  const canManage = auth?.platformCan('core.application.manage') ?? false
  const registered = useQuery({ queryKey: ['framework', 'applications', workspaceID], queryFn: api.framework.applications })
  const management = useQuery({ queryKey: ['external-integrations', workspaceID], queryFn: () => api.integrations.external(workspaceID), enabled: canManage, retry: false })
  const contracts = registered.data ? contractReport(compiled, registered.data.filter(item => item.deployment !== 'external')) : []
  const external = registered.data?.filter(item => item.deployment === 'external') ?? []

  return <PageContainer variant="workspace"><PageHeader eyebrow="Platform" title="Applications" description="Internal modules and trusted external services registered with ApexVoid. The platform controls availability and access; it never installs arbitrary software." />
    {registered.isLoading && <Card><LoadingState label="Loading registered applications…" /></Card>}
    {registered.isError && <ErrorState message="Unable to load application discovery metadata." onRetry={() => void registered.refetch()} />}
    {!registered.isLoading && !registered.isError && contracts.length + external.length === 0 && <Card><EmptyState title="No applications registered" description="This runtime currently exposes platform services only." /></Card>}
    <div className="grid gap-4 lg:grid-cols-2">{contracts.map(contract => <ApplicationCard key={contract.id} contract={contract} />)}</div>
    {external.length > 0 && <div className="mt-4 grid gap-4 lg:grid-cols-2">{external.map(application => <ExternalApplicationCard key={application.id} application={application} />)}</div>}
    {canManage && <IntegrationManagement integrations={management.data ?? []} loading={management.isLoading} workspaceID={workspaceID} />}
  </PageContainer>
}

function IntegrationManagement({ integrations, loading, workspaceID }: { integrations: ExternalIntegration[]; loading: boolean; workspaceID: string }) {
  const client = useQueryClient()
  const availability = useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => api.integrations.setWorkspaceAvailability(id, workspaceID, enabled), onSuccess: () => { void client.invalidateQueries({ queryKey: ['framework', 'applications'] }); void client.invalidateQueries({ queryKey: ['external-integrations'] }) } })
  const revoke = useMutation({ mutationFn: api.integrations.revokeCredential, onSuccess: () => void client.invalidateQueries({ queryKey: ['external-integrations'] }) })
  return <section className="mt-7"><PageHeader eyebrow="Administrator" title="External integration management" description="Availability is scoped to the selected workspace. Service credentials are never displayed after registration." />{loading ? <Card><LoadingState label="Loading external integrations…" /></Card> : integrations.length === 0 ? <Card><EmptyState title="No external services registered" description="Trusted services can be registered through the platform integration API." /></Card> : <div className="grid gap-4 lg:grid-cols-2">{integrations.map(item => { const enabled = item.workspace_enabled ?? item.enabled; return <Card key={item.id} className="p-5"><div className="flex items-start justify-between gap-4"><div><h2 className="font-semibold">{item.display_name}</h2><p className="mt-1 text-xs text-muted-foreground">{item.service_identity} · {item.version}</p></div><Badge tone={item.health === 'healthy' ? 'success' : 'warning'}>{item.health}</Badge></div><div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground"><span className="rounded-md bg-muted px-2 py-1">{item.permissions.length} permissions</span><span className="rounded-md bg-muted px-2 py-1">API {item.api_contract_version}</span>{item.credential_revoked && <span className="rounded-md bg-destructive/10 px-2 py-1 text-destructive">Credential revoked</span>}</div><div className="mt-5 flex flex-wrap gap-2"><Button variant="outline" disabled={!workspaceID || availability.isPending} onClick={() => availability.mutate({ id: item.id, enabled: !enabled })}>{enabled ? 'Disable in workspace' : 'Enable in workspace'}</Button><Button variant="outline" disabled={item.credential_revoked || revoke.isPending} onClick={() => revoke.mutate(item.id)}>Revoke credential</Button></div></Card> })}</div>}</section>
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
