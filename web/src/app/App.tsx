import { Activity, Boxes, LayoutDashboard, Settings, ShieldCheck } from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../services/api/client'

const navigation = [{ label: 'Dashboard', to: '/', icon: LayoutDashboard }, { label: 'Framework', to: '/framework', icon: Boxes }, { label: 'Settings', to: '/settings', icon: Settings }]

export function App() {
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, retry: 1, refetchInterval: 30_000 })
  const readiness = useQuery({ queryKey: ['readiness'], queryFn: api.readiness, retry: 1, refetchInterval: 30_000 })
  const healthy = health.data?.status === 'ok'
  const ready = readiness.data?.status === 'ready'

  return <div className="flex min-h-screen bg-ink text-zinc-100">
    <aside className="hidden w-64 shrink-0 border-r border-line bg-panel px-4 py-6 md:block">
      <div className="mb-12 flex items-center gap-3 px-3"><div className="grid h-9 w-9 place-items-center rounded-xl bg-accent text-ink"><Boxes size={19} /></div><div><p className="font-semibold tracking-tight">ApexVoid</p><p className="text-xs text-zinc-500">CRM foundation</p></div></div>
      <nav className="space-y-1">{navigation.map(({ label, to, icon: Icon }) => <NavLink key={to} to={to} className={({ isActive }) => `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition ${isActive ? 'bg-zinc-800 text-white' : 'text-zinc-500 hover:bg-zinc-900 hover:text-zinc-200'}`}><Icon size={17} />{label}</NavLink>)}</nav>
      <div className="mt-auto pt-20"><div className="rounded-xl border border-line bg-zinc-950/50 p-3 text-xs text-zinc-500"><div className="mb-2 flex items-center gap-2 text-zinc-300"><ShieldCheck size={15} className="text-accent" />Environment</div><span className="rounded bg-emerald-400/10 px-2 py-1 text-emerald-300">development</span></div></div>
    </aside>
    <main className="min-w-0 flex-1"><header className="flex h-16 items-center justify-between border-b border-line px-6 md:px-10"><div className="flex items-center gap-3 md:hidden"><Boxes size={20} className="text-accent" /><span className="font-semibold">ApexVoid</span></div><div className="ml-auto flex items-center gap-2 text-xs text-zinc-500"><span className={`h-2 w-2 rounded-full ${healthy ? 'bg-emerald-400' : 'bg-amber-400'}`} />{healthy ? 'API connected' : 'Connecting to API'}</div></header><div className="mx-auto max-w-6xl p-6 md:p-10"><Outlet context={{ healthy, ready }} /></div></main>
  </div>
}

export function Dashboard() {
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, retry: 1 })
  const readiness = useQuery({ queryKey: ['readiness'], queryFn: api.readiness, retry: 1 })
  const cards = [{ label: 'API service', value: health.data?.status === 'ok' ? 'Healthy' : 'Unavailable', icon: Activity, ok: health.data?.status === 'ok' }, { label: 'PostgreSQL', value: readiness.data?.dependencies.postgres === 'ok' ? 'Connected' : 'Unavailable', icon: ShieldCheck, ok: readiness.data?.dependencies.postgres === 'ok' }, { label: 'Framework modules', value: '1 registered', icon: Boxes, ok: true }]
  return <div><div className="mb-10"><p className="mb-3 text-sm font-medium text-accent">ApexVoid CRM</p><h1 className="text-3xl font-semibold tracking-tight md:text-4xl">Framework initialized</h1><p className="mt-3 max-w-xl text-zinc-500">The modular application foundation is running. Your next module can be built on explicit contracts and a predictable platform.</p></div><div className="grid gap-4 md:grid-cols-3">{cards.map(({ label, value, icon: Icon, ok }) => <div key={label} className="rounded-2xl border border-line bg-panel p-5"><div className="mb-8 flex items-center justify-between"><span className="text-sm text-zinc-500">{label}</span><Icon size={18} className={ok ? 'text-emerald-400' : 'text-zinc-600'} /></div><p className="text-xl font-medium">{value}</p><p className="mt-2 text-xs text-zinc-600">Foundation dependency check</p></div>)}</div><div className="mt-8 rounded-2xl border border-line bg-panel p-6"><div className="flex items-center gap-3"><div className="h-2 w-2 rounded-full bg-accent" /><h2 className="font-medium">Ready for the next layer</h2></div><p className="mt-3 max-w-2xl text-sm leading-6 text-zinc-500">ApexVoid is intentionally starting as a modular monolith. Domain capabilities can be added as bounded modules without introducing distributed-system complexity too early.</p></div></div>
}

export function SettingsPage() { return <div><p className="mb-3 text-sm font-medium text-accent">Platform</p><h1 className="text-3xl font-semibold tracking-tight">Settings</h1><p className="mt-3 text-zinc-500">Configuration is managed by the server environment and application YAML.</p></div> }

export function FrameworkPage() {
  const modules = useQuery({ queryKey: ['framework', 'modules'], queryFn: api.framework.modules })
  const entities = useQuery({ queryKey: ['framework', 'entities'], queryFn: api.framework.entities })
  const permissions = useQuery({ queryKey: ['framework', 'permissions'], queryFn: api.framework.permissions })
  const loading = modules.isLoading || entities.isLoading || permissions.isLoading
  return <div><div className="mb-10"><div className="mb-3 flex items-center gap-3"><p className="text-sm font-medium text-accent">ApexVoid Framework</p><span className="rounded-full border border-line px-2 py-1 text-xs text-zinc-500">v1.0.0</span></div><h1 className="text-3xl font-semibold tracking-tight">Runtime discovery</h1><p className="mt-3 max-w-2xl text-zinc-500">Read-only metadata from the compiled framework runtime. Future modules will register their contracts here.</p></div>{loading ? <div className="rounded-2xl border border-line bg-panel p-6 text-sm text-zinc-500">Loading framework metadata…</div> : <div className="grid gap-4 lg:grid-cols-3"><DiscoveryCard title="Installed modules" items={modules.data?.map((item) => `${item.name} · ${item.version}`) ?? []} /><DiscoveryCard title="Registered entities" items={entities.data?.map((item) => item.name) ?? []} /><DiscoveryCard title="Permissions" items={permissions.data?.map((item) => item.name) ?? []} /></div>}</div>
}

function DiscoveryCard({ title, items }: { title: string; items: string[] }) { return <section className="rounded-2xl border border-line bg-panel p-5"><div className="mb-5 flex items-center justify-between"><h2 className="font-medium">{title}</h2><span className="rounded-full bg-zinc-800 px-2 py-1 text-xs text-zinc-400">{items.length}</span></div><ul className="space-y-3">{items.map((item) => <li key={item} className="rounded-lg bg-zinc-950/60 px-3 py-2 text-sm text-zinc-400">{item}</li>)}</ul></section> }
