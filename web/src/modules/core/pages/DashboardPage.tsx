import { Activity, Boxes, ShieldCheck } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../core/api/client'
import { useWorkspace } from '../../../core/workspace/context'

export function DashboardPage() {
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, retry: 1 })
  const readiness = useQuery({ queryKey: ['readiness'], queryFn: api.readiness, retry: 1 })
  const { activeWorkspace } = useWorkspace()
  const cards = [{ label: 'API service', value: health.data?.status === 'ok' ? 'Healthy' : 'Unavailable', icon: Activity, ok: health.data?.status === 'ok' }, { label: 'PostgreSQL', value: readiness.data?.dependencies.postgres === 'ok' ? 'Connected' : 'Unavailable', icon: ShieldCheck, ok: readiness.data?.dependencies.postgres === 'ok' }, { label: 'Workspace', value: activeWorkspace?.name ?? 'Not selected', icon: Boxes, ok: Boolean(activeWorkspace) }]
  return <div><div className="mb-10"><p className="mb-3 text-sm font-medium text-primary">ApexVoid CRM</p><h1 className="text-3xl font-semibold tracking-tight md:text-4xl">{activeWorkspace ? `Good morning, ${activeWorkspace.name}` : 'ApexVoid is ready'}</h1><p className="mt-3 max-w-xl text-muted-foreground">Your workspace is ready. Business applications will appear here as they are enabled.</p></div><div className="grid gap-4 md:grid-cols-3">{cards.map(({ label, value, icon: Icon, ok }) => <div key={label} className="rounded-2xl border border-border bg-card p-5"><div className="mb-8 flex items-center justify-between"><span className="text-sm text-muted-foreground">{label}</span><Icon size={18} className={ok ? 'text-success' : 'text-muted-foreground/70'} /></div><p className="text-xl font-medium">{value}</p><p className="mt-2 text-xs text-muted-foreground/70">Foundation dependency check</p></div>)}</div><div className="mt-8 rounded-2xl border border-border bg-card p-6"><div className="flex items-center gap-3"><div className="h-2 w-2 rounded-full bg-primary" /><h2 className="font-medium">Your workspace is ready</h2></div><p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">Start by setting up the applications your team needs. Future CRM modules will inherit this workspace’s ownership and access boundary automatically.</p></div></div>
}
