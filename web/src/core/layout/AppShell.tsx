import { Boxes, ShieldCheck } from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import type { NavigationItem } from '../../framework/module/types'
import { api } from '../api/client'
import { useAuth } from '../auth/context'

export function AppShell({ navigation }: { navigation: NavigationItem[] }) {
  const { user, logout, can } = useAuth()
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, retry: 1, refetchInterval: 30_000 })
  const healthy = health.data?.status === 'ok'
  return <div className="flex min-h-screen bg-ink text-zinc-100">
    <aside className="hidden w-64 shrink-0 border-r border-line bg-panel px-4 py-6 md:block">
      <div className="mb-12 flex items-center gap-3 px-3"><div className="grid h-9 w-9 place-items-center rounded-xl bg-accent text-ink"><Boxes size={19} /></div><div><p className="font-semibold tracking-tight">ApexVoid</p><p className="text-xs text-zinc-500">CRM foundation</p></div></div>
      <nav className="space-y-1">{navigation.filter(item => !item.requiredPermission || can(item.requiredPermission)).map(({ id, label, path, icon: Icon }) => <NavLink key={id} to={path} className={({ isActive }) => `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition ${isActive ? 'bg-zinc-800 text-white' : 'text-zinc-500 hover:bg-zinc-900 hover:text-zinc-200'}`}><Icon size={17} />{label}</NavLink>)}</nav>
      <div className="mt-auto pt-20"><div className="rounded-xl border border-line bg-zinc-950/50 p-3 text-xs text-zinc-500"><div className="mb-2 flex items-center gap-2 text-zinc-300"><ShieldCheck size={15} className="text-accent" />Environment</div><span className="rounded bg-emerald-400/10 px-2 py-1 text-emerald-300">development</span></div></div>
    </aside>
    <main className="min-w-0 flex-1"><header className="flex h-16 items-center justify-between border-b border-line px-6 md:px-10"><div className="flex items-center gap-3 md:hidden"><Boxes size={20} className="text-accent" /><span className="font-semibold">ApexVoid</span></div><div className="ml-auto flex items-center gap-4 text-xs text-zinc-500"><span>{user?.display_name}</span><button onClick={() => void logout()} className="text-zinc-400 hover:text-white">Sign out</button><span className={`h-2 w-2 rounded-full ${healthy ? 'bg-emerald-400' : 'bg-amber-400'}`} />{healthy ? 'API connected' : 'Connecting to API'}</div></header><div className="mx-auto max-w-6xl p-6 md:p-10"><Outlet /></div></main>
  </div>
}
