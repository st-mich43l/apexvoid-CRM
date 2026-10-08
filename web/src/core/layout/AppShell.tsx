import { Boxes, ChevronDown, Moon, ShieldCheck, Sun } from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import type { NavigationItem } from '../../framework/module/types'
import { api } from '../api/client'
import { useAuth } from '../auth/context'
import { useWorkspace } from '../workspace/context'
import { useTheme } from '../theme/context'
import { canAccessNavigation } from './navigation'

export function AppShell({ navigation }: { navigation: NavigationItem[] }) {
  const { user, logout, platformCan } = useAuth()
  const { workspaces, activeWorkspace, selectWorkspace, can: workspaceCan } = useWorkspace()
  const { resolvedTheme, toggle } = useTheme()
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, retry: 1, refetchInterval: 30_000 })
  const healthy = health.data?.status === 'ok'
  return <div className="flex min-h-screen bg-background text-foreground">
    <aside className="hidden w-64 shrink-0 border-r border-border bg-card px-4 py-6 md:block">
      <div className="mb-12 flex items-center gap-3 px-3"><div className="grid h-9 w-9 place-items-center rounded-xl bg-primary text-primary-foreground"><Boxes size={19} /></div><div><p className="font-semibold tracking-tight">ApexVoid</p><p className="text-xs text-muted-foreground">CRM foundation</p></div></div>
      <nav className="space-y-1">{navigation.filter(item => canAccessNavigation(item, platformCan, workspaceCan)).map(({ id, label, path, icon: Icon }) => <NavLink key={id} to={path} className={({ isActive }) => `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition ${isActive ? 'bg-secondary text-secondary-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}`}><Icon size={17} />{label}</NavLink>)}</nav>
      <div className="mt-auto pt-20"><div className="rounded-xl border border-border bg-muted p-3 text-xs text-muted-foreground"><div className="mb-2 flex items-center gap-2 text-card-foreground"><ShieldCheck size={15} className="text-primary" />Environment</div><span className="rounded bg-success/10 px-2 py-1 text-success">development</span></div></div>
    </aside>
    <main className="min-w-0 flex-1"><header className="flex min-h-16 items-center justify-between gap-4 border-b border-border px-6 py-3 md:px-10"><div className="flex items-center gap-3 md:hidden"><Boxes size={20} className="text-primary" /><span className="font-semibold">ApexVoid</span></div><div className="flex min-w-0 items-center gap-3"><div className="min-w-0"><p className="truncate text-sm font-medium text-card-foreground">{activeWorkspace?.name ?? 'ApexVoid setup'}</p><p className="truncate text-xs text-muted-foreground">{activeWorkspace ? 'Current workspace' : 'Organization setup'}</p></div>{workspaces.length > 1 && <label className="relative"><span className="sr-only">Switch workspace</span><select value={activeWorkspace?.id ?? ''} onChange={event => selectWorkspace(event.target.value)} className="appearance-none rounded-lg border border-border bg-card py-2 pl-3 pr-8 text-xs text-card-foreground outline-none focus:border-primary">{workspaces.map(workspace => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}</select><ChevronDown size={14} className="pointer-events-none absolute right-2 top-2.5 text-muted-foreground" /></label>}</div><div className="ml-auto flex items-center gap-3 text-xs text-muted-foreground"><button type="button" onClick={toggle} aria-label={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`} title={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`} className="rounded-lg border border-border bg-card p-2 text-muted-foreground transition hover:bg-muted hover:text-foreground">{resolvedTheme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}</button><span>{user?.display_name}</span><button onClick={() => void logout()} className="text-muted-foreground hover:text-foreground">Sign out</button><span className={`h-2 w-2 rounded-full ${healthy ? 'bg-success' : 'bg-warning'}`} />{healthy ? 'API connected' : 'Connecting to API'}</div></header><div className="mx-auto max-w-6xl p-6 md:p-10"><Outlet /></div></main>
  </div>
}
