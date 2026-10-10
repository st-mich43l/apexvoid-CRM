import { Boxes, Building2, ChevronDown, ChevronRight, Command, LogOut, Menu, Moon, PanelLeftClose, Search, ShieldCheck, Sun, X } from 'lucide-react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useId, useMemo, useState } from 'react'
import type { FrontendApplication, NavigationItem } from '../../framework/module/types'
import { api } from '../api/client'
import { useAuth } from '../auth/context'
import { useWorkspace } from '../workspace/context'
import { useTheme } from '../theme/context'
import { canAccessNavigation, groupNavigation } from './navigation'

const sectionsStorageKey = 'apexvoid.sidebar.closed-sections'
const groupsStorageKey = 'apexvoid.sidebar.closed-groups'

function readSidebarPreferences(key: string): Record<string, boolean> {
  try {
    const stored: unknown = JSON.parse(window.localStorage.getItem(key) ?? '{}')
    if (!stored || typeof stored !== 'object' || Array.isArray(stored)) return {}
    return Object.fromEntries(Object.entries(stored).filter(([, value]) => typeof value === 'boolean'))
  } catch {
    return {}
  }
}

export function AppShell({ navigation, applications }: { navigation: NavigationItem[]; applications: FrontendApplication[] }) {
  const { user, logout, platformCan } = useAuth()
  const { workspaces, activeWorkspace, activeWorkspaceId, selectWorkspace, can: workspaceCan } = useWorkspace()
  const { resolvedTheme, toggle } = useTheme()
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [quickFindOpen, setQuickFindOpen] = useState(false)
  const [closedSections, setClosedSections] = useState<Record<string, boolean>>(() => readSidebarPreferences(sectionsStorageKey))
  const [closedGroups, setClosedGroups] = useState<Record<string, boolean>>(() => readSidebarPreferences(groupsStorageKey))
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, retry: 1, refetchInterval: 30_000 })
  const discovered = useQuery({ queryKey: ['framework-applications', user?.id ?? '', activeWorkspaceId ?? ''], queryFn: api.framework.applications, enabled: Boolean(user && activeWorkspaceId), retry: false, refetchInterval: 30_000 })
  const dynamicNavigation = useMemo(() => {
    const existing = new Set(navigation.map(item => item.id))
    const external = (discovered.data ?? []).filter(item => item.deployment === 'external' && item.frontend_external && item.entry_authorized && item.frontend.entry_route && !existing.has(item.frontend.navigation_id))
    return [...navigation, ...external.map(item => ({ id: item.frontend.navigation_id, label: item.display_name, path: item.frontend.entry_route, order: 500, icon: Boxes, openInNewTab: true }))]
  }, [discovered.data, navigation])
  const healthy = health.data?.status === 'ok'
  const environment = health.data?.environment ? `${health.data.environment.charAt(0).toUpperCase()}${health.data.environment.slice(1)} environment` : 'Runtime environment'
  const visible = dynamicNavigation.filter(item => canAccessNavigation(item, platformCan, workspaceCan))
  const sections = groupNavigation(visible)

  useEffect(() => {
    try { window.localStorage.setItem(sectionsStorageKey, JSON.stringify(closedSections)) } catch { /* storage may be disabled */ }
  }, [closedSections])
  useEffect(() => {
    try { window.localStorage.setItem(groupsStorageKey, JSON.stringify(closedGroups)) } catch { /* storage may be disabled */ }
  }, [closedGroups])

  useEffect(() => {
    const media = window.matchMedia('(min-width: 1280px)')
    const expandOnWideScreen = () => { if (media.matches) setCollapsed(false) }
    expandOnWideScreen()
    media.addEventListener?.('change', expandOnWideScreen)
    return () => media.removeEventListener?.('change', expandOnWideScreen)
  }, [])

  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        setQuickFindOpen(open => !open)
      }
      if (event.key === 'Escape') {
        setQuickFindOpen(false)
        setMobileOpen(false)
      }
    }
    window.addEventListener('keydown', handleShortcut)
    return () => window.removeEventListener('keydown', handleShortcut)
  }, [])

  const closeMobile = () => setMobileOpen(false)
  const sidebarProps = {
    sections, activePath: location.pathname, activeWorkspace, workspaces, selectWorkspace, healthy, environment,
    closedSections, closedGroups,
    onToggleSection: (id: string) => setClosedSections(current => ({ ...current, [id]: !current[id] })),
    onToggleGroup: (id: string) => setClosedGroups(current => ({ ...current, [id]: !current[id] })),
    onQuickFind: () => setQuickFindOpen(true), onClose: closeMobile,
  }

  return <div className="flex h-dvh min-h-0 overflow-hidden bg-background text-foreground">
    <Sidebar {...sidebarProps} collapsed={collapsed} mobile={false} onCollapse={() => setCollapsed(true)} onExpand={() => setCollapsed(false)} />
    {mobileOpen && <><button type="button" aria-label="Close navigation" onClick={closeMobile} className="fixed inset-0 z-30 bg-slate-950/50 backdrop-blur-sm md:hidden" /><Sidebar {...sidebarProps} collapsed={false} mobile onCollapse={closeMobile} onExpand={closeMobile} /></>}

    <main className="flex h-dvh min-w-0 flex-1 flex-col overflow-hidden">
      <header className="flex min-h-16 shrink-0 items-center justify-between gap-3 border-b border-border bg-card/90 px-4 py-3 shadow-sm backdrop-blur-md sm:px-6 lg:px-10">
        <div className="flex min-w-0 items-center gap-3">
          <button type="button" onClick={() => setMobileOpen(true)} aria-label="Open navigation" className="rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground md:hidden"><Menu size={20} /></button>
          <div className="flex min-w-0 items-center gap-3 md:hidden"><Boxes size={20} className="text-primary" /><span className="font-semibold">ApexVoid</span></div>
          <div className="hidden min-w-0 md:block"><p className="truncate text-sm font-medium text-card-foreground">{activeWorkspace?.name ?? 'ApexVoid setup'}</p><p className="truncate text-xs text-muted-foreground">{activeWorkspace ? 'Current workspace' : 'Organization setup'}</p></div>
          {workspaces.length > 1 && <label className="relative hidden sm:block"><span className="sr-only">Switch workspace</span><select value={activeWorkspace?.id ?? ''} onChange={event => selectWorkspace(event.target.value)} className="appearance-none rounded-lg border border-border bg-background py-2 pl-3 pr-8 text-xs text-card-foreground outline-none focus:border-primary">{workspaces.map(workspace => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}</select><ChevronDown size={14} className="pointer-events-none absolute right-2 top-2.5 text-muted-foreground" /></label>}
        </div>
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground sm:gap-3"><button type="button" onClick={() => setQuickFindOpen(true)} aria-label="Open quick find" title="Quick find (⌘ K)" className="rounded-lg border border-border bg-background p-2 text-muted-foreground transition hover:bg-muted hover:text-foreground sm:hidden"><Search size={16} /></button><button type="button" onClick={toggle} aria-label={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`} title={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`} className="rounded-lg border border-border bg-background p-2 text-muted-foreground transition hover:bg-muted hover:text-foreground">{resolvedTheme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}</button><span className="hidden sm:inline">{user?.display_name}</span><button onClick={() => void logout()} aria-label="Sign out" title="Sign out" className="rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground"><LogOut size={16} /></button><span className={`h-2 w-2 rounded-full ${healthy ? 'bg-success' : 'bg-warning'}`} title={healthy ? 'API connected' : 'Connecting to API'} /></div>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden"><div className="w-full p-5 sm:p-6 lg:p-10"><Outlet context={{ applications }} /></div></div>
    </main>
    {quickFindOpen && <QuickFind navigation={visible} onClose={() => setQuickFindOpen(false)} onNavigate={closeMobile} />}
  </div>
}

type SidebarProps = {
  sections: ReturnType<typeof groupNavigation>
  activePath: string
  activeWorkspace: ReturnType<typeof useWorkspace>['activeWorkspace']
  workspaces: ReturnType<typeof useWorkspace>['workspaces']
  selectWorkspace: (id: string) => void
  healthy: boolean
  environment: string
  collapsed: boolean
  mobile: boolean
  closedSections: Record<string, boolean>
  closedGroups: Record<string, boolean>
  onToggleSection: (id: string) => void
  onToggleGroup: (id: string) => void
  onQuickFind: () => void
  onCollapse: () => void
  onExpand: () => void
  onClose: () => void
}

function Sidebar({ sections, activePath, activeWorkspace, workspaces, selectWorkspace, healthy, environment, collapsed, mobile, closedSections, closedGroups, onToggleSection, onToggleGroup, onQuickFind, onCollapse, onExpand, onClose }: SidebarProps) {
  const sidebarID = useId()
  return <aside onClick={event => { if (!mobile && collapsed && !(event.target as HTMLElement).closest('button,select')) onExpand() }} onKeyDown={event => { if (!mobile && collapsed && (event.key === 'Enter' || event.key === ' ')) { event.preventDefault(); onExpand() } }} tabIndex={!mobile && collapsed ? 0 : undefined} aria-label={!mobile && collapsed ? 'Click to expand sidebar' : undefined} className={`${mobile ? 'fixed inset-y-0 left-0 z-40 flex w-[min(86vw,288px)] shadow-2xl md:hidden' : 'relative hidden md:flex'} h-dvh max-h-dvh shrink-0 flex-col overflow-hidden border-r border-[hsl(var(--sidebar-border))] bg-[hsl(var(--sidebar))] text-[hsl(var(--sidebar-foreground))] transition-[width,padding,transform] duration-300 ${!mobile && (collapsed ? 'w-[84px] cursor-e-resize px-3' : 'w-72 px-4')}`}>
    <div className={`flex h-[76px] shrink-0 items-center ${collapsed ? 'justify-center' : 'justify-between px-2'}`}><div className="flex min-w-0 items-center gap-3"><div className="grid h-10 w-10 shrink-0 place-items-center rounded-2xl bg-gradient-to-br from-violet-400 via-primary to-indigo-600 text-white shadow-lg shadow-violet-950/20"><Boxes size={20} strokeWidth={2.4} /></div>{!collapsed && <div className="min-w-0"><p className="truncate text-[15px] font-semibold tracking-tight">ApexVoid</p><p className="truncate text-xs text-[hsl(var(--sidebar-muted))]">Business platform</p></div>}</div>{mobile ? <button type="button" onClick={onClose} aria-label="Close navigation" className="rounded-lg p-2 text-[hsl(var(--sidebar-muted))] hover:bg-[hsl(var(--sidebar-hover))] hover:text-white"><X size={18} /></button> : !collapsed && <button type="button" onClick={onCollapse} aria-label="Collapse sidebar" title="Collapse sidebar" className="rounded-lg p-2 text-[hsl(var(--sidebar-muted))] transition hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]"><PanelLeftClose size={16} /></button>}</div>
    {!collapsed && <div className="mb-4 rounded-2xl border border-[hsl(var(--sidebar-border))] bg-[hsl(var(--sidebar-card))] p-2 shadow-inner shadow-white/[0.03]"><label className="relative block"><span className="sr-only">Switch workspace</span><Building2 className="pointer-events-none absolute left-3 top-3 text-primary" size={16} /><select value={activeWorkspace?.id ?? ''} onChange={event => selectWorkspace(event.target.value)} className="w-full appearance-none rounded-xl bg-transparent py-2.5 pl-9 pr-9 text-sm font-medium text-[hsl(var(--sidebar-foreground))] outline-none"><option value="">{activeWorkspace?.name ?? 'Select workspace'}</option>{workspaces.filter(workspace => workspace.id !== activeWorkspace?.id).map(workspace => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}</select><ChevronDown className="pointer-events-none absolute right-3 top-3 text-[hsl(var(--sidebar-muted))]" size={16} /></label><div className="mt-1 flex items-center gap-2 px-3 text-[11px] text-[hsl(var(--sidebar-muted))]"><span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />Workspace active</div></div>}
    {collapsed && <div className="mb-4 flex justify-center"><div title={activeWorkspace?.name ?? 'No workspace selected'} className="grid h-10 w-10 place-items-center rounded-xl border border-[hsl(var(--sidebar-border))] bg-[hsl(var(--sidebar-card))] text-primary"><Building2 size={17} /></div></div>}
    {!collapsed && <button type="button" onClick={onQuickFind} className="mb-4 flex w-full items-center justify-between rounded-xl border border-[hsl(var(--sidebar-border))] px-3 py-2 text-left text-xs text-[hsl(var(--sidebar-muted))] transition hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]"><span className="flex items-center gap-2"><Command size={14} />Quick find</span><kbd className="rounded-md bg-[hsl(var(--sidebar-card))] px-1.5 py-0.5 font-mono text-[10px]">⌘ K</kbd></button>}
    <nav aria-label="Primary navigation" className="min-h-0 flex-1 overflow-y-auto overscroll-contain pr-1">
      {sections.map(section => {
        const sectionID = `${sidebarID}-section-${section.id}`
        const isClosed = Boolean(closedSections[section.id])
        return <div key={section.id} className="mb-4 last:mb-0">
          {collapsed ? <div aria-hidden="true" className="mb-2 flex items-center justify-center py-1"><span className="h-1 w-1 rounded-full bg-primary/70" /></div> :
            <button
              type="button"
              aria-label={`${section.label} section`}
              aria-expanded={!isClosed}
              aria-controls={sectionID}
              onClick={() => onToggleSection(section.id)}
              className="group mb-1 flex min-h-9 w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.18em] text-[hsl(var(--sidebar-muted))] transition-colors hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            >
              <span className="h-1 w-1 shrink-0 rounded-full bg-primary/70" />
              <span className="min-w-0 flex-1 truncate">{section.label}</span>
              <ChevronDown size={14} aria-hidden="true" className={`shrink-0 transition-transform duration-200 ${isClosed ? '-rotate-90' : ''}`} />
            </button>}
          <div id={sectionID} hidden={!collapsed && isClosed} className="space-y-1">
            {section.items.filter(item => !item.parentId).map(item =>
              <NavigationBranch
                key={item.id}
                item={item}
                childrenItems={section.items.filter(child => child.parentId === item.id)}
                collapsed={collapsed}
                activePath={activePath}
                groupClosed={Boolean(closedGroups[item.id])}
                onToggleGroup={() => onToggleGroup(item.id)}
                onExpand={() => {
                  onExpand()
                  if (closedSections[section.id]) onToggleSection(section.id)
                }}
                onNavigate={onClose}
              />,
            )}
          </div>
        </div>
      })}
    </nav>
    <div className={`mt-4 shrink-0 border-t border-[hsl(var(--sidebar-border))] pb-5 pt-4 ${collapsed ? 'flex justify-center' : 'px-2'}`}>{!collapsed ? <div className="flex items-center justify-between"><div className="flex min-w-0 items-center gap-2.5"><ShieldCheck size={15} className={healthy ? 'text-emerald-400' : 'text-amber-400'} /><div className="min-w-0"><p className="text-[11px] font-medium text-[hsl(var(--sidebar-foreground))]">{healthy ? 'Operational' : 'Checking status'}</p><p className="mt-0.5 text-[10px] text-[hsl(var(--sidebar-muted))]">{environment}</p></div></div><span title={healthy ? 'API healthy' : 'API unavailable'} className={`h-2 w-2 rounded-full ${healthy ? 'bg-emerald-400 shadow-[0_0_0_3px_rgba(52,211,153,0.12)]' : 'bg-amber-400 shadow-[0_0_0_3px_rgba(251,191,36,0.12)]'}`} /></div> : <div title={healthy ? `Operational · ${environment}` : 'Checking status'} className={`h-2.5 w-2.5 rounded-full ${healthy ? 'bg-emerald-400 shadow-[0_0_0_4px_rgba(52,211,153,0.12)]' : 'bg-amber-400 shadow-[0_0_0_4px_rgba(251,191,36,0.12)]'}`} />}</div>
  </aside>
}

function NavigationBranch({ item, childrenItems, collapsed, activePath, groupClosed, onToggleGroup, onExpand, onNavigate }: {
  item: NavigationItem
  childrenItems: NavigationItem[]
  collapsed: boolean
  activePath: string
  groupClosed: boolean
  onToggleGroup: () => void
  onExpand: () => void
  onNavigate: () => void
}) {
  const hasChildren = childrenItems.length > 0
  const isClosed = groupClosed && !collapsed
  const groupActive = childrenItems.some(child => activePath === child.path || activePath.startsWith(`${child.path}/`))
  const childrenID = useId()
  const Icon = item.icon
  return <div>
    {item.isGroup ? <button
      type="button"
      title={collapsed ? item.label : undefined}
      aria-label={`${item.label} menu`}
      aria-expanded={hasChildren ? !isClosed : undefined}
      aria-controls={hasChildren ? childrenID : undefined}
      onClick={() => {
        if (collapsed) {
          onExpand()
          if (groupClosed) onToggleGroup()
        } else if (hasChildren) onToggleGroup()
      }}
      className={`group flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left text-xs font-semibold transition-colors hover:bg-[hsl(var(--sidebar-hover))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ${groupActive ? 'text-[hsl(var(--sidebar-foreground))]' : 'text-[hsl(var(--sidebar-muted))]'} ${collapsed ? 'justify-center px-0' : ''}`}
    >
      <Icon size={16} aria-hidden="true" className={`shrink-0 ${groupActive ? 'text-primary' : 'text-[hsl(var(--sidebar-muted))]'}`} />
      {!collapsed && <><span className="min-w-0 flex-1 truncate">{item.label}</span>{hasChildren && <ChevronRight size={14} aria-hidden="true" className={`ml-auto shrink-0 transition-transform duration-200 ${!isClosed ? 'rotate-90 text-primary' : ''}`} />}</>}
    </button> :
      item.openInNewTab ? <a onClick={onNavigate} title={collapsed ? item.label : undefined} href={item.path} target="_blank" rel="noopener noreferrer" className={`group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition ${collapsed ? 'justify-center px-0' : ''} ${activePath === item.path ? 'bg-[hsl(var(--sidebar-active))] font-medium text-[hsl(var(--sidebar-active-foreground))] shadow-sm' : 'text-[hsl(var(--sidebar-muted))] hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]'}`}>
        <Icon size={17} className="shrink-0 transition group-hover:text-primary" />
        {!collapsed && <span>{item.label}</span>}
      </a> : <NavLink onClick={onNavigate} end={item.path === '/'} title={collapsed ? item.label : undefined} to={item.path} className={({ isActive }) => `group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition ${collapsed ? 'justify-center px-0' : ''} ${isActive ? 'bg-[hsl(var(--sidebar-active))] font-medium text-[hsl(var(--sidebar-active-foreground))] shadow-sm' : 'text-[hsl(var(--sidebar-muted))] hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]'}`}>
        <Icon size={17} className="shrink-0 transition group-hover:text-primary" />
        {!collapsed && <span>{item.label}</span>}
      </NavLink>}
    {hasChildren && <div id={childrenID} hidden={isClosed} className={`${collapsed ? 'mt-1 space-y-1' : 'ml-3 mt-1 space-y-1 border-l border-[hsl(var(--sidebar-border))] pl-3'}`}>
      {childrenItems.map(child => {
        const ChildIcon = child.icon
        return <NavLink onClick={onNavigate} key={child.id} end title={collapsed ? child.label : undefined} to={child.path} className={({ isActive }) => `group flex items-center gap-3 rounded-lg px-3 py-2 text-[13px] transition ${collapsed ? 'justify-center px-0' : ''} ${isActive ? 'bg-[hsl(var(--sidebar-active))] font-medium text-[hsl(var(--sidebar-active-foreground))]' : 'text-[hsl(var(--sidebar-muted))] hover:bg-[hsl(var(--sidebar-hover))] hover:text-[hsl(var(--sidebar-foreground))]'}`}>
          <ChildIcon size={15} className="shrink-0 group-hover:text-primary" />
          {!collapsed && <span>{child.label}</span>}
        </NavLink>
      })}
    </div>}
  </div>
}

function QuickFind({ navigation, onClose, onNavigate }: { navigation: NavigationItem[]; onClose: () => void; onNavigate: () => void }) {
  return <div className="fixed inset-0 z-50 grid place-items-start bg-slate-950/40 p-4 pt-[12vh] backdrop-blur-sm" role="presentation" onMouseDown={event => { if (event.target === event.currentTarget) onClose() }}><div role="dialog" aria-modal="true" aria-labelledby="quick-find-title" className="mx-auto w-full max-w-xl overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"><div className="flex items-center gap-3 border-b border-border px-4 py-3"><Search size={18} className="text-muted-foreground" /><input autoFocus aria-label="Search navigation" placeholder="Jump to a page…" className="min-w-0 flex-1 bg-transparent text-sm outline-none" onChange={event => { const value = event.target.value.toLowerCase(); for (const element of document.querySelectorAll<HTMLElement>('[data-quick-find-label]')) element.hidden = !element.dataset.quickFindLabel?.includes(value) }} /><button type="button" onClick={onClose} aria-label="Close quick find" className="rounded-lg p-1 text-muted-foreground hover:bg-muted"><X size={16} /></button></div><div id="quick-find-title" className="max-h-[60vh] overflow-y-auto p-2">{navigation.filter(item => !item.isGroup).map(item => { const Icon = item.icon; const props = { key: item.id, 'data-quick-find-label': item.label.toLowerCase(), onClick: () => { onClose(); onNavigate() }, className: 'flex items-center gap-3 rounded-xl px-3 py-3 text-sm text-muted-foreground hover:bg-muted hover:text-foreground' }; return item.openInNewTab ? <a {...props} href={item.path} target="_blank" rel="noopener noreferrer"><Icon size={17} /><span>{item.label}</span><span className="ml-auto text-xs text-muted-foreground">{item.path}</span></a> : <NavLink {...props} to={item.path}><Icon size={17} /><span>{item.label}</span><span className="ml-auto text-xs text-muted-foreground">{item.path}</span></NavLink> })}</div></div></div>
}
