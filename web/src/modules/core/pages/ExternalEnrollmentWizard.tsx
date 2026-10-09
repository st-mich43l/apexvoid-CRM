import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Check, ChevronLeft, Database, KeyRound, Search, ShieldCheck, Sparkles, Workflow } from 'lucide-react'
import { api, type ExternalInstallation } from '../../../core/api/client'
import { useWorkspace } from '../../../core/workspace/context'
import { Badge, Button, inputClass } from '../../../components/ui'

type Approval = 'registration' | 'permissions' | 'database' | 'schema' | 'migrations'
const requirements: Array<{ key: Approval; label: string; detail: string }> = [
  { key: 'registration', label: 'Application registration', detail: 'Register the service and its application identity' },
  { key: 'permissions', label: 'Permission catalog', detail: 'Create the application-owned RBAC permissions' },
  { key: 'database', label: 'Database provisioning', detail: 'Create an isolated database and restricted account' },
  { key: 'schema', label: 'Application schema', detail: 'Initialize its owned schema and migration ledger' },
  { key: 'migrations', label: 'SQL migrations', detail: 'Apply the reviewed, checksummed migration bundle' },
]
const blankApprovals = (): Record<Approval, boolean> => ({ registration: false, permissions: false, database: false, schema: false, migrations: false })

export function ExternalEnrollmentWizard({ onComplete, onBusyChange }: { onComplete: () => void; onBusyChange?: (busy: boolean) => void }) {
  const { workspaces, activeWorkspaceId } = useWorkspace()
  const [serviceURL, setServiceURL] = useState('')
  const [enrollmentCode, setEnrollmentCode] = useState('')
  const [plan, setPlan] = useState<ExternalInstallation | null>(null)
  const [workspaceIDs, setWorkspaceIDs] = useState<string[]>(activeWorkspaceId ? [activeWorkspaceId] : [])
  const [error, setError] = useState('')
  const [approvals, setApprovals] = useState(blankApprovals)
  const approved = requirements.every(item => approvals[item.key])
  const discover = useMutation({
    mutationFn: () => api.integrations.discover({ service_url: serviceURL.trim(), enrollment_code: enrollmentCode }),
    onSuccess: value => { setPlan(value); setError(''); setApprovals(blankApprovals()) },
    onError: value => setError(value instanceof Error ? value.message : 'The service could not be discovered.'),
  })
  const approve = useMutation({
    mutationFn: () => api.integrations.approveInstallation(plan!.id, {
      enrollment_code: enrollmentCode,
      approve_registration: approvals.registration, approve_permissions: approvals.permissions,
      approve_database: approvals.database, approve_schema: approvals.schema, approve_migrations: approvals.migrations,
      workspace_ids: workspaceIDs,
    }),
    onSuccess: onComplete,
    onError: value => setError(value instanceof Error ? value.message : 'The application could not be activated.'),
  })
  useEffect(() => { onBusyChange?.(discover.isPending || approve.isPending) }, [discover.isPending, approve.isPending, onBusyChange])
  const startDiscovery = (event: FormEvent) => {
    event.preventDefault()
    if (!serviceURL.trim() || enrollmentCode.trim().length < 32) {
      setError('Enter the service URL and the complete one-time enrollment code (at least 32 characters).')
      return
    }
    discover.mutate()
  }
  const back = () => { setPlan(null); setApprovals(blankApprovals()); setError('') }
  return <div className="space-y-6">
    <div aria-label="Registration progress" className="flex items-center gap-3 text-xs">
      <div className="flex items-center gap-2 font-semibold text-primary"><span className="grid h-7 w-7 place-items-center rounded-full bg-primary text-primary-foreground">{plan ? <Check size={15} /> : '1'}</span>Connect</div>
      <div className="h-px flex-1 bg-border" />
      <div className={plan ? 'flex items-center gap-2 font-semibold text-primary' : 'flex items-center gap-2 text-muted-foreground'}><span className={plan ? 'grid h-7 w-7 place-items-center rounded-full bg-primary text-primary-foreground' : 'grid h-7 w-7 place-items-center rounded-full border border-border bg-muted'}>2</span>Review & activate</div>
    </div>
    {error && <p role="alert" className="rounded-xl border border-destructive/25 bg-destructive/5 p-3 text-sm text-destructive">{error}</p>}

    {!plan ? <form onSubmit={startDiscovery} className="space-y-5">
      <div className="rounded-2xl border border-border bg-muted/20 p-5">
        <div className="flex items-center gap-2 text-sm font-semibold"><Workflow size={18} className="text-primary" />Automatic application discovery</div>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">Start the application's Docker container, then copy the service URL and one-time registration code from its startup output. Enterprise will read the manifest for you.</p>
      </div>
      <div><label htmlFor="external-service-url" className="block text-sm font-medium text-foreground">Application service URL</label><input id="external-service-url" autoFocus required type="url" className={inputClass} placeholder="http://apexvoid-cafe:8090" value={serviceURL} onChange={event => setServiceURL(event.target.value)} /><p className="mt-1 text-xs text-muted-foreground">Use the Docker service address accessible from Enterprise.</p></div>
      <div><label htmlFor="external-enrollment-code" className="block text-sm font-medium text-foreground">One-time enrollment code</label><input id="external-enrollment-code" required minLength={32} type="password" autoComplete="off" className={inputClass} placeholder="Paste the code printed by the application" value={enrollmentCode} onChange={event => setEnrollmentCode(event.target.value)} /><p className="mt-1 text-xs text-muted-foreground">Used to authenticate discovery and establish the secure application connection.</p></div>
      <div className="flex justify-end border-t border-border pt-5"><Button disabled={discover.isPending} type="submit" className="gap-2 px-5"><Search size={16} />{discover.isPending ? 'Discovering service…' : 'Discover application'}</Button></div>
    </form> : <div className="space-y-5">
      <div className="flex items-start gap-3 rounded-2xl border border-primary/20 bg-primary/[0.05] p-4">
        <div className="grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><Sparkles size={21} /></div>
        <div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><h3 className="font-semibold text-foreground">{plan.manifest.application.display_name}</h3><Badge tone="success">Manifest verified</Badge></div><p className="mt-1 text-xs text-muted-foreground">{plan.manifest.application.id} · {plan.manifest.application.description || 'Independent ApexVoid application'}</p></div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Metadata label="Application version" value={plan.manifest.application.version} detail="Declared by the application manifest" />
        <Metadata label="API contract" value={plan.manifest.application.api_contract_version} detail="Discovered and validated by Enterprise" />
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Review icon={<Workflow size={17} />} title="Service connection"><p className="break-all">{plan.service_url}</p><p className="mt-2">Identity: {plan.manifest.service.identity}</p><p className="mt-1">Frontend: {plan.manifest.service.frontend_route}</p></Review>
        <Review icon={<Database size={17} />} title="Database ownership"><p className="font-medium text-foreground">{plan.manifest.database.name}</p><p className="mt-2">Schema: {plan.manifest.database.schema}</p><p className="mt-1">Role: {plan.manifest.database.role}</p></Review>
      </div>
      <Review icon={<ShieldCheck size={17} />} title="Requested permissions"><div className="flex flex-wrap gap-2">{plan.manifest.permissions.map(item => <Badge key={item.name}>{item.name} · {item.scope}</Badge>)}</div></Review>
      <Review icon={<KeyRound size={17} />} title="Migration plan"><p>{plan.manifest.migrations.length} versioned SQL migration(s) · bundle {plan.manifest.database.migration_bundle_version}</p><div className="mt-2 space-y-1">{plan.manifest.migrations.map(item => <p key={item.version} className="break-all font-mono text-[11px]">v{item.version} · {item.path} · SHA-256 verified</p>)}</div></Review>
      <fieldset className="rounded-2xl border border-border p-4"><legend className="px-2 text-sm font-semibold">Workspace availability</legend><p className="mb-3 text-xs leading-5 text-muted-foreground">Choose workspaces where the application should be enabled after activation.</p><div className="grid gap-2 sm:grid-cols-2">{workspaces.map(workspace => <label key={workspace.id} className="flex min-h-11 items-center gap-3 rounded-xl border border-border bg-muted/10 px-3 text-sm"><input type="checkbox" checked={workspaceIDs.includes(workspace.id)} onChange={() => setWorkspaceIDs(current => current.includes(workspace.id) ? current.filter(value => value !== workspace.id) : [...current, workspace.id])} />{workspace.name}</label>)}</div></fieldset>
      <fieldset className="rounded-2xl border border-border p-4"><legend className="px-2 text-sm font-semibold">Approve installation operations</legend><p className="mb-3 text-xs leading-5 text-muted-foreground">Each operation requires explicit authorization. Review the service identity and migration plan before proceeding.</p><div className="space-y-2">{requirements.map(item => <label key={item.key} className="flex items-start gap-3 rounded-xl border border-border/70 bg-muted/10 p-3"><input type="checkbox" className="mt-1" checked={approvals[item.key]} onChange={event => setApprovals(current => ({ ...current, [item.key]: event.target.checked }))} /><span className="min-w-0"><span className="block text-sm font-medium">{item.label}</span><span className="mt-0.5 block text-xs leading-5 text-muted-foreground">{item.detail}</span></span></label>)}</div></fieldset>
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-5"><Button type="button" variant="outline" disabled={approve.isPending} className="gap-2" onClick={back}><ChevronLeft size={16} />Back</Button><Button type="button" disabled={approve.isPending || !approved || workspaceIDs.length === 0} onClick={() => approve.mutate()} className="gap-2 px-5"><Check size={16} />{approve.isPending ? 'Provisioning & activating…' : 'Approve & activate'}</Button></div>
    </div>}
  </div>
}

function Metadata({ label, value, detail }: { label: string; value: string; detail: string }) {
  return <div className="rounded-2xl border border-border bg-muted/15 p-4"><p className="text-xs font-medium text-muted-foreground">{label}</p><p className="mt-2 text-lg font-semibold tracking-tight text-foreground">{value}</p><p className="mt-1 text-xs text-muted-foreground">{detail}</p></div>
}
function Review({ title, icon, children }: { title: string; icon: ReactNode; children: ReactNode }) {
  return <section className="rounded-2xl border border-border p-4"><h4 className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground"><span className="text-primary">{icon}</span>{title}</h4><div className="text-xs leading-6 text-muted-foreground">{children}</div></section>
}
