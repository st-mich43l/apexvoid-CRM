import { Boxes, Building2, ChevronDown, ChevronRight, Command, LogOut, Moon, PanelLeftClose, ShieldCheck, Sun } from 'lucide-react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import type { NavigationItem } from '../../framework/module/types'
import { api } from '../api/client'
import { useAuth } from '../auth/context'
import { useWorkspace } from '../workspace/context'
import { useTheme } from '../theme/context'
import { canAccessNavigation, groupNavigation } from './navigation'

export function AppShell({ navigation }: { navigation: NavigationItem[] }) {
  const { user, logout, platformCan } = useAuth()
  const { workspaces, activeWorkspace, selectWorkspace, can: workspaceCan } = useWorkspace()
  const { resolvedTheme, toggle } = useTheme()
  const [collapsed, setCollapsed] = useState(false)
  useEffect(() => {
    const media = window.matchMedia('(min-width: 1280px)')
    const expandOnWideScreen = () => { if (media.matches) setCollapsed(false) }
    expandOnWideScreen()
    media.addEventListener?.('change', expandOnWideScreen)
    return () => media.removeEventListener?.('change', expandOnWideScreen)
  }, [])
  const location = useLocation()
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, retry: 1, refetchInterval: 30_000 })
  const healthy = health.data?.status === 'ok'
  const visible = navigation.filter(item => canAccessNavigation(item, platformCan, workspaceCan))
  const sections = groupNavigation(visible)

  return <div className="flex min-h-screen overflow-x-hidden bg-background text-foreground">
    <aside onClick={event => { if (collapsed && !(event.target as HTMLElement).closest('a,button,select')) setCollapsed(false) }} onKeyDown={event => { if (collapsed && (event.key === 'Enter' || event.key === ' ')) { event.preventDefault(); setCollapsed(false) } }} tabIndex={collapsed ? 0 : undefined} aria-label={collapsed ? 'Click to expand sidebar' : undefined} className={`relative sticky top-0 hidden h-screen shrink-0 flex-col border-r border-[hsl(var(--sidebar-border))] bg-[hsl(var(--sidebar))] text-[hsl(var(--sidebar-foreground))] shadow-[8px_0_30px_-24px_rgba(16,10,45,0.65)] transition-[width,padding] duration-300 md:flex ${collapsed ? 'w-[84px] cursor-e-resize px-3' : 'w-72 px-4'}`}>
      <div className={`flex h-20 shrink-0 items-center ${collapsed ? 'justify-center' : 'justify-between px-2'}`}>
        <div className="flex min-w-0 items-center gap-3">
          <div className="grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-gradient-to-br from-violet-400 via-primary to-indigo-600 text-white shadow-lg shadow-violet-950/20"><Boxes size={20} strokeWidth={2.4} /></div>
          {!collapsed && <div className="min-w-0"><p className="truncate text-[15px] font-semibold tracking-tight">ApexVoid</p><p className="truncate text-xs text-[hsl(var(--sidebar-muted))]">CRM foundation</p></div>}
        </div>
        {!collapsed && <button type="button" onClick={() => setCollapsed(true)} aria-label="Collapse sidebar" title="Collapse sidebar" className="rounded-lg p-2 text-[hsl(var(--sidebar-muted))] transition hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]"><PanelLeftClose size={16} /></button>}
      </div>

      {!collapsed && <div className="mb-5 rounded-2xl border border-[hsl(var(--sidebar-border))] bg-[hsl(var(--sidebar-card))] p-2 shadow-inner shadow-white/[0.03]"><label className="relative block"><span className="sr-only">Switch workspace</span><Building2 className="pointer-events-none absolute left-3 top-3 text-primary" size={16} /><select value={activeWorkspace?.id ?? ''} onChange={event => selectWorkspace(event.target.value)} className="w-full appearance-none rounded-xl bg-transparent py-2.5 pl-9 pr-9 text-sm font-medium text-[hsl(var(--sidebar-foreground))] outline-none"><option value="">{activeWorkspace?.name ?? 'Select workspace'}</option>{workspaces.filter(workspace => workspace.id !== activeWorkspace?.id).map(workspace => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}</select><ChevronDown className="pointer-events-none absolute right-3 top-3 text-[hsl(var(--sidebar-muted))]" size={16} /></label><div className="mt-1 flex items-center gap-2 px-3 text-[11px] text-[hsl(var(--sidebar-muted))]"><span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />Workspace active</div></div>}
      {collapsed && <div className="mb-5 flex justify-center"><div title={activeWorkspace?.name ?? 'No workspace selected'} className="grid h-10 w-10 place-items-center rounded-xl border border-[hsl(var(--sidebar-border))] bg-[hsl(var(--sidebar-card))] text-primary"><Building2 size={17} /></div></div>}

      {!collapsed && <button type="button" className="mb-5 flex w-full items-center justify-between rounded-xl border border-[hsl(var(--sidebar-border))] px-3 py-2 text-left text-xs text-[hsl(var(--sidebar-muted))] transition hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]"><span className="flex items-center gap-2"><Command size={14} />Quick find</span><kbd className="rounded-md bg-[hsl(var(--sidebar-card))] px-1.5 py-0.5 font-mono text-[10px]">⌘ K</kbd></button>}

      <nav aria-label="Primary navigation" className="min-h-0 flex-1 overflow-y-auto pr-1">
        {sections.map(section => <div key={section.id} className="mb-6 last:mb-0"><div className={`mb-2 flex items-center gap-2 px-3 text-[10px] font-semibold uppercase tracking-[0.18em] text-[hsl(var(--sidebar-muted))] ${collapsed ? 'justify-center px-0' : ''}`}><span className="h-1 w-1 rounded-full bg-primary/70" />{!collapsed && section.label}</div><div className="space-y-1">{section.items.filter(item => !item.parentId).map(item => <NavigationBranch key={item.id} item={item} childrenItems={section.items.filter(child => child.parentId === item.id)} collapsed={collapsed} activePath={location.pathname} />)}</div></div>)}
      </nav>

      <div className={`mt-5 shrink-0 border-t border-[hsl(var(--sidebar-border))] pt-4 ${collapsed ? 'flex justify-center' : ''}`}>
        <div title={collapsed ? 'System status: healthy' : undefined} className={`flex items-center ${collapsed ? 'h-9 w-9 justify-center rounded-xl bg-emerald-400/10 text-emerald-400' : 'gap-3 rounded-xl bg-[hsl(var(--sidebar-card))] px-3 py-2.5'}`}><span className="relative flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-emerald-400/10 text-emerald-400"><ShieldCheck size={15} /><span className="absolute -right-0.5 -top-0.5 h-2 w-2 rounded-full border-2 border-[hsl(var(--sidebar-card))] bg-emerald-400" /></span>{!collapsed && <div className="min-w-0"><p className="text-xs font-medium text-[hsl(var(--sidebar-foreground))]">System status</p><p className="mt-0.5 truncate text-[11px] text-[hsl(var(--sidebar-muted))]">All systems operational</p></div>}</div>
      </div>
    </aside>

    <main className="min-w-0 flex-1 overflow-x-hidden"><header className="sticky top-0 z-20 flex min-h-16 items-center justify-between gap-4 border-b border-border bg-card/90 px-6 py-3 shadow-sm backdrop-blur-md md:px-10"><div className="flex items-center gap-3 md:hidden"><Boxes size={20} className="text-primary" /><span className="font-semibold">ApexVoid</span></div><div className="flex min-w-0 items-center gap-3"><div className="min-w-0"><p className="truncate text-sm font-medium text-card-foreground">{activeWorkspace?.name ?? 'ApexVoid setup'}</p><p className="truncate text-xs text-muted-foreground">{activeWorkspace ? 'Current workspace' : 'Organization setup'}</p></div>{workspaces.length > 1 && <label className="relative hidden sm:block"><span className="sr-only">Switch workspace</span><select value={activeWorkspace?.id ?? ''} onChange={event => selectWorkspace(event.target.value)} className="appearance-none rounded-lg border border-border bg-background py-2 pl-3 pr-8 text-xs text-card-foreground outline-none focus:border-primary">{workspaces.map(workspace => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}</select><ChevronDown size={14} className="pointer-events-none absolute right-2 top-2.5 text-muted-foreground" /></label>}</div><div className="ml-auto flex items-center gap-3 text-xs text-muted-foreground"><button type="button" onClick={toggle} aria-label={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`} title={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`} className="rounded-lg border border-border bg-background p-2 text-muted-foreground transition hover:bg-muted hover:text-foreground">{resolvedTheme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}</button><span className="hidden sm:inline">{user?.display_name}</span><button onClick={() => void logout()} aria-label="Sign out" title="Sign out" className="rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground"><LogOut size={16} /></button><span className={`h-2 w-2 rounded-full ${healthy ? 'bg-success' : 'bg-warning'}`} title={healthy ? 'API connected' : 'Connecting to API'} /></div></header><div className="mx-auto w-full max-w-6xl p-6 md:p-10"><Outlet /></div></main>
  </div>
}

function NavigationBranch({ item, childrenItems, collapsed, activePath }: { item: NavigationItem; childrenItems: NavigationItem[]; collapsed: boolean; activePath: string }) {
  const hasChildren = childrenItems.length > 0
  const groupActive = childrenItems.some(child => activePath === child.path || activePath.startsWith(`${child.path}/`))
  const Icon = item.icon
  return <div>{item.isGroup ? <div title={collapsed ? item.label : undefined} className={`flex items-center gap-3 rounded-xl px-3 py-2 text-xs font-semibold ${groupActive ? 'text-[hsl(var(--sidebar-foreground))]' : 'text-[hsl(var(--sidebar-muted))]'} ${collapsed ? 'justify-center px-0' : ''}`}><Icon size={16} className={groupActive ? 'text-primary' : 'text-[hsl(var(--sidebar-muted))]'} />{!collapsed && <><span>{item.label}</span>{hasChildren && <ChevronRight size={14} className={`ml-auto transition-transform ${groupActive ? 'rotate-90 text-primary' : ''}`} />}</>}</div> : <NavLink end={item.path === '/'} title={collapsed ? item.label : undefined} to={item.path} className={({ isActive }) => `group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition ${collapsed ? 'justify-center px-0' : ''} ${isActive ? 'bg-[hsl(var(--sidebar-active))] font-medium text-[hsl(var(--sidebar-active-foreground))] shadow-sm' : 'text-[hsl(var(--sidebar-muted))] hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]'}`}><Icon size={17} className="shrink-0 transition group-hover:text-primary" />{!collapsed && <span>{item.label}</span>}</NavLink>}{hasChildren && <div className={`${collapsed ? 'mt-1 space-y-1' : 'ml-3 mt-1 space-y-1 border-l border-[hsl(var(--sidebar-border))] pl-3'}`}>{childrenItems.map(child => { const ChildIcon = child.icon; return <NavLink key={child.id} end title={collapsed ? child.label : undefined} to={child.path} className={({ isActive }) => `group flex items-center gap-3 rounded-lg px-3 py-2 text-[13px] transition ${collapsed ? 'justify-center px-0' : ''} ${isActive ? 'bg-[hsl(var(--sidebar-active))] font-medium text-[hsl(var(--sidebar-active-foreground))]' : 'text-[hsl(var(--sidebar-muted))] hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]'}`}><ChildIcon size={15} className="shrink-0 group-hover:text-primary" />{!collapsed && <span>{child.label}</span>}</NavLink> })}</div>}</div>
}
