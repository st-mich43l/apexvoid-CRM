import { useQuery } from '@tanstack/react-query'
import { api } from '../../../core/api/client'

export function FrameworkPage() {
  const modules = useQuery({ queryKey: ['framework', 'modules'], queryFn: api.framework.modules })
  const entities = useQuery({ queryKey: ['framework', 'entities'], queryFn: api.framework.entities })
  const permissions = useQuery({ queryKey: ['framework', 'permissions'], queryFn: api.framework.permissions })
  const loading = modules.isLoading || entities.isLoading || permissions.isLoading
  return <div><div className="mb-10"><div className="mb-3 flex items-center gap-3"><p className="text-sm font-medium text-primary">ApexVoid Framework</p><span className="rounded-full border border-border px-2 py-1 text-xs text-muted-foreground">v1.0.0</span></div><h1 className="text-3xl font-semibold tracking-tight">Runtime discovery</h1><p className="mt-3 max-w-2xl text-muted-foreground">Read-only metadata from the compiled framework runtime. Future modules will register their contracts here.</p></div>{loading ? <div className="rounded-2xl border border-border bg-card p-6 text-sm text-muted-foreground">Loading framework metadata…</div> : <div className="grid gap-4 lg:grid-cols-3"><DiscoveryCard title="Installed modules" items={modules.data?.map((item) => `${item.name} · ${item.version}`) ?? []} /><DiscoveryCard title="Registered entities" items={entities.data?.map((item) => item.name) ?? []} /><DiscoveryCard title="Permissions" items={permissions.data?.map((item) => item.name) ?? []} /></div>}</div>
}

function DiscoveryCard({ title, items }: { title: string; items: string[] }) { return <section className="rounded-2xl border border-border bg-card p-5"><div className="mb-5 flex items-center justify-between"><h2 className="font-medium">{title}</h2><span className="rounded-full bg-secondary px-2 py-1 text-xs text-muted-foreground">{items.length}</span></div><ul className="space-y-3">{items.map((item) => <li key={item} className="rounded-lg bg-muted/60 px-3 py-2 text-sm text-muted-foreground">{item}</li>)}</ul></section> }
