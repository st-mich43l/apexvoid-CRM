import { Boxes, ExternalLink, Settings2, TriangleAlert } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Link, useOutletContext } from 'react-router-dom'
import { api } from '../../../core/api/client'
import { Badge, Card, EmptyState, ErrorState, LoadingState, PageContainer, PageHeader } from '../../../components/ui'
import type { ApplicationMetadata } from '../../../framework/metadata/types'
import type { FrontendApplication } from '../../../framework/module/types'
import { ModuleRegistry, type ApplicationContract } from '../../../framework/module/registry'

type ShellContext = { applications: FrontendApplication[] }

export function ApplicationsPage() {
  const { applications: compiled } = useOutletContext<ShellContext>()
  const registered = useQuery({ queryKey: ['framework', 'applications'], queryFn: api.framework.applications })
  const contracts = registered.data ? contractReport(compiled, registered.data) : []

  return <PageContainer variant="workspace"><PageHeader eyebrow="Platform" title="Applications" description="Compiled applications registered by the running ApexVoid framework. Applications are deployed with the product; this page does not install arbitrary software." />
    {registered.isLoading && <Card><LoadingState label="Loading registered applications…" /></Card>}
    {registered.isError && <ErrorState message="Unable to load application discovery metadata." onRetry={() => void registered.refetch()} />}
    {!registered.isLoading && !registered.isError && contracts.length === 0 && <Card><EmptyState title="No applications registered" description="This runtime currently exposes platform services only." /></Card>}
    <div className="grid gap-4 lg:grid-cols-2">{contracts.map(contract => <ApplicationCard key={contract.id} contract={contract} />)}</div>
  </PageContainer>
}

function contractReport(compiled: FrontendApplication[], registered: ApplicationMetadata[]): ApplicationContract[] {
  const modules = compiled.map(application => ({ name: `compiled-${application.id}`, version: '1.0.0', application, routes: [{ id: `${application.id}-route`, path: application.entryRoute, element: null }], navigation: [{ id: application.navigationID, label: application.id, path: application.entryRoute, order: 0, icon: Boxes }] }))
  return new ModuleRegistry(modules).applicationContracts(registered)
}

function ApplicationCard({ contract }: { contract: ApplicationContract }) {
  const application = contract.application
  const ready = contract.status === 'ready'
  return <Card className="flex flex-col p-5"><div className="flex items-start justify-between gap-4"><div className="flex min-w-0 items-start gap-3"><span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><Boxes size={19} /></span><div className="min-w-0"><h2 className="font-semibold text-card-foreground">{application?.display_name ?? contract.id}</h2><p className="mt-1 text-xs text-muted-foreground">{application ? `v${application.version} · ${application.id}` : contract.id}</p></div></div><Badge tone={ready ? 'success' : 'warning'}>{ready ? 'Ready' : 'Contract issue'}</Badge></div>
    <p className="mt-4 text-sm leading-6 text-muted-foreground">{application?.description ?? contract.message}</p>
    {application && <div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground"><span className="rounded-md bg-muted px-2 py-1">Modules: {application.module_dependencies.join(', ')}</span>{application.required_permissions.length > 0 && <span className="rounded-md bg-muted px-2 py-1">Permissions: {application.required_permissions.join(', ')}</span>}</div>}
    {!ready && <div className="mt-4 flex gap-2 rounded-lg border border-warning/20 bg-warning/10 p-3 text-xs leading-5 text-warning"><TriangleAlert size={16} className="mt-0.5 shrink-0" /><p>{contract.message}</p></div>}
    {ready && application && <div className="mt-5 flex flex-wrap gap-2"><Link to={application.frontend.entry_route} className="inline-flex min-h-9 items-center gap-2 rounded-lg bg-primary px-3 text-sm font-medium text-primary-foreground hover:opacity-90">Open application <ExternalLink size={14} /></Link>{application.settings && <Link to={application.settings.route} className="inline-flex min-h-9 items-center gap-2 rounded-lg border border-border px-3 text-sm font-medium text-foreground hover:bg-muted"><Settings2 size={14} />Settings</Link>}</div>}
  </Card>
}
