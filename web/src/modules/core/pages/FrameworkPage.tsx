import { useQuery } from '@tanstack/react-query'
import { api } from '../../../core/api/client'
import { Badge, Card, CardHeader, EmptyState, LoadingState, PageContainer, PageHeader } from '../../../components/ui'

export function FrameworkPage() {
  const modules = useQuery({ queryKey: ['framework', 'modules'], queryFn: api.framework.modules })
  const entities = useQuery({ queryKey: ['framework', 'entities'], queryFn: api.framework.entities })
  const permissions = useQuery({ queryKey: ['framework', 'permissions'], queryFn: api.framework.permissions })
  const loading = modules.isLoading || entities.isLoading || permissions.isLoading
  return <PageContainer variant="workspace"><PageHeader eyebrow="ApexVoid Framework" title="Runtime discovery" description="Read-only metadata from the compiled framework runtime. Future modules will register their contracts here." actions={<Badge tone="primary">v1.0.0</Badge>} />{loading ? <Card><LoadingState label="Loading framework metadata…" /></Card> : <div className="grid gap-4 lg:grid-cols-3"><DiscoveryCard title="Installed modules" items={modules.data?.map((item) => `${item.name} · ${item.version}`) ?? []} /><DiscoveryCard title="Registered entities" items={entities.data?.map((item) => item.name) ?? []} /><DiscoveryCard title="Permissions" items={permissions.data?.map((item) => item.name) ?? []} /></div>}</PageContainer>
}

function DiscoveryCard({ title, items }: { title: string; items: string[] }) { return <Card><CardHeader title={title} action={<Badge>{items.length}</Badge>} />{items.length > 0 ? <ul className="space-y-2 p-5">{items.map((item) => <li key={item} className="rounded-lg bg-muted/60 px-3 py-2 text-sm text-muted-foreground">{item}</li>)}</ul> : <EmptyState title="Nothing registered yet" description="This runtime has not exposed any entries for this category." />}</Card> }
