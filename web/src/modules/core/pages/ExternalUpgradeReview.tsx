import { useState } from 'react'
import { CheckCircle2, Clock3, Database, Fingerprint, GitPullRequest, ShieldCheck, TriangleAlert } from 'lucide-react'
import type { ExternalIntegration, ExternalUpgrade } from '../../../core/api/client'
import { Badge, Button } from '../../../components/ui'

export function ExternalUpgradeReview({ application, plan, pending, error, onClose, onRetry, onApprove }: {
  application: ExternalIntegration
  plan: ExternalUpgrade
  pending: boolean
  error?: string
  onClose: () => void
  onRetry?: () => Promise<void>
  onApprove: (permissions: boolean, migrations: boolean) => Promise<void>
}) {
  const [approvedPermissions, setApprovedPermissions] = useState(false)
  const [approvedMigrations, setApprovedMigrations] = useState(false)
  const ready = plan.status === 'pending_approval' && approvedPermissions && approvedMigrations
  const upToDate = plan.status === 'up_to_date'
  const complete = plan.status === 'applied'
  return <div className="space-y-5">
    <div className="flex items-start gap-3 rounded-xl border border-border bg-muted/20 p-4">
      <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><GitPullRequest size={20} /></span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2"><h2 className="text-base font-semibold">{application.display_name}</h2>
          <Badge tone={complete || upToDate ? 'success' : plan.status === 'pending_approval' ? 'primary' : 'warning'}>{upToDate ? 'Up to date' : complete ? 'Upgrade applied' : plan.status.replaceAll('_', ' ')}</Badge>
        </div>
        <p className="mt-1 text-xs leading-5 text-muted-foreground">Current: v{plan.installed_version || application.version} → Available: v{plan.available_version}</p>
      </div>
    </div>
    {error && <p role="alert" className="rounded-xl border border-destructive/25 bg-destructive/5 p-3 text-sm text-destructive">{error}</p>}
    {plan.failure && <div role="alert" className="rounded-xl border border-warning/25 bg-warning/5 p-4 text-sm text-warning"><p className="font-semibold">{plan.failure.code}</p><p className="mt-1 text-xs">Migration v{plan.failure.version} · {plan.failure.path}</p><p className="mt-2 text-xs leading-5">{plan.failure.message}</p></div>}
    {(upToDate || complete) ? <div className="flex gap-3 rounded-xl border border-success/20 bg-success/5 p-4"><CheckCircle2 size={18} className="shrink-0 text-success" /><div><p className="text-sm font-semibold">{upToDate ? 'No changes to install' : 'Verified upgrade completed'}</p><p className="mt-1 text-xs leading-6 text-muted-foreground">{upToDate ? 'The registered application manifest matches the installed version. Nothing was changed.' : 'The reviewed manifest and append-only migration history were activated.'}</p></div></div> : <>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="rounded-xl border border-border p-4">
          <p className="flex items-center gap-2 text-xs font-semibold"><ShieldCheck size={15} className="text-primary" />Application permissions</p>
          <p className="mt-2 text-xl font-semibold">{plan.added_permissions?.length ?? 0}</p>
          <p className="text-xs text-muted-foreground">New permissions</p>
          <div className="mt-3 space-y-1.5">{plan.added_permissions?.map(item => <p key={item.name} className="break-all font-mono text-[11px] text-foreground">{item.name} <span className="font-sans text-muted-foreground">({item.scope})</span></p>)}</div>
        </div>
        <div className="rounded-xl border border-border p-4">
          <p className="flex items-center gap-2 text-xs font-semibold"><Database size={15} className="text-primary" />Database migrations</p>
          <p className="mt-2 text-xl font-semibold">{plan.pending_migrations?.length ?? 0}</p>
          <p className="text-xs text-muted-foreground">Append-only SQL migrations</p>
          <div className="mt-3 space-y-2">{plan.pending_migrations?.map(item => <div key={item.version} className="flex items-start justify-between gap-2"><p className="break-all font-mono text-[11px] text-foreground">v{item.version} · {item.path}</p><Badge tone={item.status === 'failed' ? 'warning' : item.status === 'applied' ? 'success' : 'neutral'}>{item.status.replaceAll('_', ' ')}</Badge></div>)}</div>
        </div>
      </div>
      <div className="space-y-3 rounded-xl border border-border bg-muted/10 p-4">
        <p className="flex items-center gap-2 text-xs font-semibold text-foreground"><Fingerprint size={15} className="text-primary" />Signed manifest fingerprint</p>
        <code className="block select-all break-all rounded-lg bg-muted/50 p-3 text-[11px] text-muted-foreground">{plan.manifest_sha256}</code>
        {plan.expires_at && <p className="flex items-center gap-1.5 text-xs text-muted-foreground"><Clock3 size={13} />Review expires {new Date(plan.expires_at).toLocaleString()}</p>}
        <p className="text-xs leading-6 text-muted-foreground">Enterprise verifies the enrolled service credential, manifest integrity, every existing migration checksum, and application database ownership. Review the deployment's SQL changes before approval. Existing permission names and scopes cannot be removed or changed here.</p>
      </div>
      {plan.status === 'pending_approval' && <fieldset className="space-y-3 rounded-xl border border-border p-4">
        <legend className="px-2 text-sm font-semibold">Administrator approval</legend>
        <label className="flex cursor-pointer items-start gap-3 text-sm leading-6"><input className="mt-1.5 accent-primary" type="checkbox" checked={approvedPermissions} onChange={event => setApprovedPermissions(event.target.checked)} /><span>I have reviewed the updated permission catalog and application entry policies.</span></label>
        <label className="flex cursor-pointer items-start gap-3 text-sm leading-6"><input className="mt-1.5 accent-primary" type="checkbox" checked={approvedMigrations} onChange={event => setApprovedMigrations(event.target.checked)} /><span>I have reviewed the SQL migration versions, paths and checksums and approve applying the pinned bundle to this application's own database.</span></label>
      </fieldset>}
      {plan.status !== 'pending_approval' && <div className="flex items-start gap-3 rounded-xl border border-warning/20 bg-warning/5 p-4 text-sm text-muted-foreground"><TriangleAlert size={18} className="shrink-0 text-warning" />This upgrade plan is no longer pending. Check for updates again to create a fresh verified review.</div>}
    </>}
    <div className="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
      <Button type="button" variant="outline" disabled={pending} onClick={onClose}>{upToDate || complete ? 'Close' : 'Cancel'}</Button>
      {onRetry && !upToDate && !complete && <Button type="button" variant="outline" disabled={pending} onClick={() => void onRetry()}>{pending ? 'Checking…' : 'Check again'}</Button>}
      {!upToDate && !complete && <Button type="button" disabled={!ready || pending} onClick={() => void onApprove(approvedPermissions, approvedMigrations)}>{pending ? 'Verifying & applying…' : 'Approve & install upgrade'}</Button>}
    </div>
  </div>
}
