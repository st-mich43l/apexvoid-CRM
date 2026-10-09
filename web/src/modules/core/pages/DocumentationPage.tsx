import { useEffect, useRef, useState, type RefObject } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  AlertTriangle, ArrowLeft, ArrowRight, BookOpen, Check, CheckCircle2, ChevronRight,
  CircleHelp, Clipboard, Clock3, Code2, Compass, FileText, Layers3, Lightbulb,
  Search, ShieldCheck, TerminalSquare, X,
} from 'lucide-react'
import type { DocBlock, DocCategory, DocGuide } from './documentation/content'
import { categories, guides, guidesBySlug, searchGuides } from './documentation/content'

const categoryDescriptions: Record<DocCategory, string> = {
  'Start here': 'Learn the foundations and bring Enterprise online.',
  'Build applications': 'Create, discover and connect independent apps.',
  'Platform architecture': 'Understand isolation, routing and permission boundaries.',
  'Run & maintain': 'Deploy, diagnose and operate with confidence.',
}

function categoryIcon(category: DocCategory) {
  switch (category) {
    case 'Start here': return Compass
    case 'Build applications': return Code2
    case 'Platform architecture': return Layers3
    case 'Run & maintain': return ShieldCheck
  }
}

function GuideLink({ guide, active, compact = false }: { guide: DocGuide; active: boolean; compact?: boolean }) {
  return <Link
    to={'/docs/' + guide.slug}
    aria-current={active ? 'page' : undefined}
    className={'group flex items-start gap-2.5 rounded-xl transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary ' +
      (compact ? 'px-3 py-2.5 ' : 'px-3 py-3 ') +
      (active ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-muted/70 hover:text-foreground')}
  >
    <FileText size={compact ? 14 : 16} className="mt-0.5 shrink-0" aria-hidden="true" />
    <span className="min-w-0 flex-1">
      <span className={'block leading-5 ' + (active ? 'font-semibold' : 'font-medium') + (compact ? ' text-xs' : ' text-sm')}>{guide.title}</span>
      {!compact && <span className="mt-1 block text-xs leading-5 text-muted-foreground">{guide.description}</span>}
    </span>
  </Link>
}

function SearchField({ query, onChange, inputRef }: { query: string; onChange: (value: string) => void; inputRef: RefObject<HTMLInputElement | null> }) {
  return <label className="relative block">
    <span className="sr-only">Search documentation</span>
    <Search size={17} aria-hidden="true" className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
    <input
      ref={inputRef}
      aria-label="Search documentation"
      autoComplete="off"
      type="search"
      value={query}
      onChange={event => onChange(event.target.value)}
      placeholder="Search articles, commands, permissions…"
      className="min-h-11 w-full rounded-xl border border-input bg-card pl-10 pr-10 text-sm text-foreground shadow-sm outline-none transition placeholder:text-muted-foreground focus:border-primary focus:ring-2 focus:ring-primary/15"
    />
    {query && <button type="button" aria-label="Clear documentation search" onClick={() => onChange('')} className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"><X size={14} /></button>}
  </label>
}

function TableOfContents({ guide }: { guide: DocGuide }) {
  return <aside aria-label="On this page" className="hidden 2xl:block">
    <div className="sticky top-6 space-y-3 border-l border-border pl-5">
      <p className="text-[11px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">On this page</p>
      <nav className="space-y-1">
        {guide.sections.map(section =>
          <a key={section.id} href={'#' + section.id} className="block rounded-md py-1.5 text-xs leading-5 text-muted-foreground transition hover:text-primary">{section.title}</a>,
        )}
      </nav>
      <div className="border-t border-border pt-4">
        <Link to="/docs/troubleshooting" className="inline-flex items-center gap-2 text-xs font-medium text-primary hover:underline"><CircleHelp size={14} />Troubleshooting</Link>
      </div>
    </div>
  </aside>
}

function CodeSample({ title, language, value }: { title: string; language: string; value: string }) {
  const [copyState, setCopyState] = useState<'ready' | 'copied' | 'failed'>('ready')
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopyState('copied')
    } catch {
      setCopyState('failed')
    }
  }
  return <div className="my-5 overflow-hidden rounded-xl border border-border bg-[hsl(var(--sidebar))] text-[hsl(var(--sidebar-foreground))]">
    <div className="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-2.5">
      <div className="flex min-w-0 items-center gap-2">
        <TerminalSquare size={14} className="shrink-0 text-violet-300" aria-hidden="true" />
        <span className="truncate text-xs font-medium">{title}</span>
        <span className="shrink-0 rounded border border-white/15 px-1.5 py-0.5 text-[10px] text-white/60">{language}</span>
      </div>
      <button type="button" onClick={() => void copy()} aria-label={'Copy ' + title} className="inline-flex shrink-0 items-center gap-1.5 rounded-lg px-2 py-1.5 text-xs text-white/70 transition hover:bg-white/10 hover:text-white">
        {copyState === 'copied' ? <Check size={13} /> : <Clipboard size={13} />}
        {copyState === 'copied' ? 'Copied' : copyState === 'failed' ? 'Unavailable' : 'Copy'}
      </button>
    </div>
    <pre className="overflow-x-auto p-4 text-[12px] leading-6 text-[#dfdbf9]"><code>{value}</code></pre>
  </div>
}

function DocBlockView({ block }: { block: DocBlock }) {
  if (block.type === 'paragraph') return <p className="my-3 text-sm leading-7 text-muted-foreground">{block.text}</p>
  if (block.type === 'list') {
    const ListTag = block.ordered ? 'ol' : 'ul'
    return <ListTag className={'my-4 space-y-2.5 pl-6 text-sm leading-7 text-muted-foreground ' + (block.ordered ? 'list-decimal' : 'list-disc')}>
      {block.items.map((item, index) => <li key={index} className="pl-1 marker:text-primary">{item}</li>)}
    </ListTag>
  }
  if (block.type === 'code') return <CodeSample title={block.title} language={block.language} value={block.value} />
  if (block.type === 'callout') {
    const Icon = block.tone === 'warning' ? AlertTriangle : block.tone === 'success' ? CheckCircle2 : Lightbulb
    const colors = block.tone === 'warning' ? 'border-warning/30 bg-warning/5' : block.tone === 'success' ? 'border-success/30 bg-success/5' : 'border-primary/25 bg-primary/5'
    const iconColor = block.tone === 'warning' ? 'text-warning' : block.tone === 'success' ? 'text-success' : 'text-primary'
    return <div role="note" className={'my-5 flex gap-3 rounded-xl border p-4 ' + colors}>
      <Icon size={18} aria-hidden="true" className={'mt-0.5 shrink-0 ' + iconColor} />
      <div><p className="text-sm font-semibold text-foreground">{block.title}</p><p className="mt-1 text-sm leading-6 text-muted-foreground">{block.text}</p></div>
    </div>
  }
  return <div className="my-5 overflow-x-auto rounded-xl border border-border">
    <table className="w-full min-w-[460px] border-collapse text-left text-xs">
      <thead className="bg-muted/60"><tr><th scope="col" className="w-[36%] px-4 py-3 font-semibold text-foreground">{block.headings[0]}</th><th scope="col" className="px-4 py-3 font-semibold text-foreground">{block.headings[1]}</th></tr></thead>
      <tbody>{block.rows.map(([key, value], index) => <tr key={index} className="border-t border-border/70"><th scope="row" className="px-4 py-3 align-top font-medium text-foreground">{key}</th><td className="break-words px-4 py-3 align-top leading-5 text-muted-foreground">{value}</td></tr>)}</tbody>
    </table>
  </div>
}

function GuideArticle({ guide }: { guide: DocGuide }) {
  return <article className="min-w-0">
    <div className="mb-5 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
      <Link to="/docs" className="hover:text-primary">Documentation</Link>
      <ChevronRight size={13} aria-hidden="true" />
      <span>{guide.category}</span>
      <ChevronRight size={13} aria-hidden="true" />
      <span className="font-medium text-foreground">{guide.title}</span>
    </div>
    <div className="rounded-2xl border border-border bg-card px-5 py-7 shadow-sm sm:px-8 sm:py-9">
      <div className="mb-7 border-b border-border pb-7">
        <span className="inline-flex items-center rounded-full bg-primary/10 px-2.5 py-1 text-[11px] font-medium text-primary">{guide.category}</span>
        <h1 className="mt-4 text-2xl font-bold tracking-tight text-foreground sm:text-[30px] sm:leading-10">{guide.title}</h1>
        <p className="mt-3 max-w-2xl text-sm leading-7 text-muted-foreground">{guide.description}</p>
        <div className="mt-5 flex flex-wrap items-center gap-4 text-xs text-muted-foreground"><span className="inline-flex items-center gap-1.5"><Clock3 size={14} />{guide.minutes} min read</span><span className="inline-flex items-center gap-1.5"><BookOpen size={14} />{guide.sections.length} sections</span></div>
      </div>
      <div className="mb-7 rounded-xl border border-border bg-muted/30 px-4 py-4 2xl:hidden">
        <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-foreground">In this guide</p>
        <nav aria-label="Article sections" className="flex flex-wrap gap-x-5 gap-y-2">{guide.sections.map(section => <a key={section.id} href={'#' + section.id} className="text-xs text-primary hover:underline">{section.title}</a>)}</nav>
      </div>
      {guide.sections.map((section, index) => <section key={section.id} id={section.id} className={'scroll-mt-6 ' + (index > 0 ? 'mt-9 border-t border-border/70 pt-8' : '')}>
        <h2 className="mb-4 text-lg font-semibold tracking-tight text-foreground">{section.title}</h2>
        {section.blocks.map((block, blockIndex) => <DocBlockView key={blockIndex} block={block} />)}
      </section>)}
      <div className="mt-10 border-t border-border pt-6">
        <p className="mb-3 text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">Continue learning</p>
        <div className="grid gap-2 sm:grid-cols-2">
          {guide.related.map(slug => {
            const next = guidesBySlug.get(slug)
            if (!next) return null
            return <Link key={slug} to={'/docs/' + slug} className="group flex items-center justify-between gap-3 rounded-xl border border-border px-4 py-3 text-sm font-medium transition hover:border-primary/40 hover:bg-muted/40">
              {next.title}<ArrowRight size={15} className="shrink-0 text-primary transition group-hover:translate-x-0.5" />
            </Link>
          })}
        </div>
      </div>
    </div>
    <div className="mt-5 flex items-center justify-between gap-4 text-xs text-muted-foreground"><span>Built into ApexVoid Enterprise · v1 guides</span><Link to="/docs" className="inline-flex items-center gap-1.5 text-primary hover:underline"><ArrowLeft size={14} />All documentation</Link></div>
  </article>
}

function DocumentationHome({ results, query }: { results: DocGuide[]; query: string }) {
  if (query.trim()) return <div className="space-y-5">
    <div><h1 className="text-2xl font-semibold tracking-tight text-foreground">Search results</h1><p className="mt-2 text-sm text-muted-foreground">{results.length} {results.length === 1 ? 'guide' : 'guides'} matching “{query.trim()}”</p></div>
    {results.length === 0 ? <div className="rounded-2xl border border-dashed border-border bg-card px-6 py-12 text-center"><Search size={26} className="mx-auto text-muted-foreground" /><p className="mt-4 font-semibold">No matching guides</p><p className="mt-2 text-sm text-muted-foreground">Try “Docker”, “enrollment”, “RBAC” or “Firefox”.</p></div> : <div className="grid gap-3">{results.map(guide =>
      <Link key={guide.slug} to={'/docs/' + guide.slug} className="group rounded-xl border border-border bg-card p-5 transition hover:border-primary/40 hover:shadow-sm"><p className="text-[11px] font-medium text-primary">{guide.category} · {guide.minutes} min read</p><h2 className="mt-2 font-semibold text-foreground group-hover:text-primary">{guide.title} <ArrowRight size={15} className="inline align-middle" /></h2><p className="mt-1 text-sm leading-6 text-muted-foreground">{guide.description}</p></Link>,
    )}</div>}
  </div>
  return <div className="min-w-0 space-y-9">
    <section className="relative overflow-hidden rounded-2xl border border-primary/20 bg-gradient-to-br from-primary/15 via-card to-card p-6 sm:p-9">
      <div aria-hidden="true" className="pointer-events-none absolute -right-10 -top-12 h-52 w-52 rounded-full bg-primary/10 blur-3xl" />
      <div className="relative max-w-2xl">
        <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-primary/20 bg-card/70 px-3 py-1.5 text-xs font-medium text-primary"><BookOpen size={14} />ApexVoid knowledge center</div>
        <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">Build confidently on ApexVoid.</h1>
        <p className="mt-4 text-sm leading-7 text-muted-foreground sm:text-base">Everything you need to install the Enterprise platform, develop independent business applications, register them securely, and run them in production.</p>
        <div className="mt-6 flex flex-wrap items-center gap-3">
          <Link to="/docs/local-setup" className="inline-flex min-h-10 items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground transition hover:opacity-90">Get started <ArrowRight size={16} /></Link>
          <Link to="/docs/first-application" className="inline-flex min-h-10 items-center gap-2 rounded-xl border border-border bg-card px-4 py-2.5 text-sm font-semibold text-foreground transition hover:bg-muted">Build an application</Link>
        </div>
      </div>
    </section>
    <div className="grid gap-3 sm:grid-cols-3">
      {[{ label: 'Quick-start', detail: 'From zero to running Docker', icon: TerminalSquare, slug: 'local-setup' }, { label: 'Application enrollment', detail: 'From signed manifest to activation', icon: Code2, slug: 'first-application' }, { label: 'Security model', detail: 'Sessions, workspaces and RBAC', icon: ShieldCheck, slug: 'security-and-rbac' }].map(card =>
        <Link key={card.slug} to={'/docs/' + card.slug} className="group rounded-2xl border border-border bg-card p-5 transition hover:border-primary/40 hover:shadow-sm">
          <span className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary"><card.icon size={19} /></span>
          <h2 className="mt-4 text-sm font-semibold text-foreground group-hover:text-primary">{card.label}</h2>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">{card.detail}</p>
          <span className="mt-4 inline-flex items-center gap-1 text-xs font-medium text-primary">Read guide <ArrowRight size={13} /></span>
        </Link>,
      )}
    </div>
    {categories.map(category => {
      const Icon = categoryIcon(category)
      return <section key={category} className="space-y-4">
        <div className="flex items-center gap-3"><div className="grid h-9 w-9 place-items-center rounded-lg bg-muted text-primary"><Icon size={18} /></div><div><h2 className="text-base font-semibold text-foreground">{category}</h2><p className="text-xs text-muted-foreground">{categoryDescriptions[category]}</p></div></div>
        <div className="grid gap-3 lg:grid-cols-2">{guides.filter(guide => guide.category === category).map(guide =>
          <Link key={guide.slug} to={'/docs/' + guide.slug} className="group rounded-xl border border-border bg-card p-4 transition hover:border-primary/40 hover:shadow-sm">
            <div className="flex items-center justify-between gap-3"><h3 className="text-sm font-semibold text-foreground group-hover:text-primary">{guide.title}</h3><ArrowRight size={15} className="shrink-0 text-primary" /></div>
            <p className="mt-2 text-xs leading-5 text-muted-foreground">{guide.description}</p><p className="mt-3 inline-flex items-center gap-1 text-[11px] text-muted-foreground"><Clock3 size={12} />{guide.minutes} min read</p>
          </Link>,
        )}</div>
      </section>
    })}
    <div className="flex flex-col gap-3 rounded-xl border border-border bg-muted/30 p-5 sm:flex-row sm:items-center sm:justify-between">
      <div><h2 className="text-sm font-semibold text-foreground">Having trouble connecting an app?</h2><p className="mt-1 text-xs leading-5 text-muted-foreground">Find practical fixes for Docker DNS, manifest signing, PostgreSQL provisioning, RBAC and iframe errors.</p></div>
      <Link to="/docs/troubleshooting" className="inline-flex shrink-0 items-center gap-2 text-sm font-semibold text-primary hover:underline">Troubleshooting <ArrowRight size={14} /></Link>
    </div>
  </div>
}

export function DocumentationPage() {
  const { slug } = useParams<{ slug?: string }>()
  const [query, setQuery] = useState('')
  const searchInput = useRef<HTMLInputElement>(null)
  const guide = slug ? guidesBySlug.get(slug) : undefined
  const results = searchGuides(query)

  useEffect(() => {
    const previous = document.title
    document.title = (guide ? guide.title + ' · ' : 'Documentation · ') + 'ApexVoid Enterprise'
    return () => { document.title = previous }
  }, [guide])

  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      const target = event.target
      if (event.key !== '/' || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return
      if (target instanceof HTMLElement && target.closest('input, textarea, select, [contenteditable="true"]')) return
      event.preventDefault()
      searchInput.current?.focus()
    }
    document.addEventListener('keydown', handleShortcut)
    return () => document.removeEventListener('keydown', handleShortcut)
  }, [])

  return <div className="mx-auto max-w-[1540px]">
    <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
      <div><p className="mb-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-primary">Platform resources</p><div className="flex items-center gap-2"><BookOpen size={20} className="text-primary" /><h2 className="text-lg font-semibold tracking-tight text-foreground">Documentation</h2></div></div>
      <span className="inline-flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1.5 text-xs text-muted-foreground"><CheckCircle2 size={13} className="text-success" />Included in Enterprise</span>
    </div>
    <div className="grid items-start gap-6 lg:grid-cols-[238px_minmax(0,1fr)] 2xl:grid-cols-[238px_minmax(0,1fr)_185px]">
      <aside aria-label="Documentation navigation" className="min-w-0 rounded-2xl border border-border bg-card p-3 lg:sticky lg:top-6">
        <div className="px-1 pb-3"><SearchField query={query} onChange={setQuery} inputRef={searchInput} /></div>
        <Link to="/docs" className={'mb-3 flex items-center gap-2 rounded-xl px-3 py-2.5 text-sm font-semibold transition ' + (!slug ? 'bg-primary/10 text-primary' : 'text-foreground hover:bg-muted')}><BookOpen size={16} />All guides</Link>
        {categories.map(category => {
          const entries = results.filter(result => result.category === category)
          if (entries.length === 0) return null
          const Icon = categoryIcon(category)
          return <div key={category} className="mb-4 last:mb-0">
            <p className="mb-1 flex items-center gap-2 px-3 py-2 text-[10px] font-bold uppercase tracking-[0.13em] text-muted-foreground"><Icon size={13} />{category}</p>
            <nav aria-label={category} className="space-y-0.5">{entries.map(item => <GuideLink key={item.slug} guide={item} active={guide?.slug === item.slug} compact />)}</nav>
          </div>
        })}
        {results.length === 0 && <p className="px-3 py-6 text-center text-xs leading-5 text-muted-foreground">No guides match your search.</p>}
        <div className="mt-5 border-t border-border px-2 pt-4"><p className="text-[11px] leading-5 text-muted-foreground">Tip: Press <kbd className="rounded border border-border px-1.5 py-0.5 font-mono">/</kbd> to search the documentation.</p></div>
      </aside>
      <section id="documentation-content" className="min-w-0">
        {slug && !guide ? <div className="rounded-2xl border border-border bg-card px-8 py-12 text-center"><CircleHelp size={30} className="mx-auto text-muted-foreground" /><h1 className="mt-4 text-xl font-semibold">Guide not found</h1><p className="mt-2 text-sm text-muted-foreground">The documentation page you requested does not exist.</p><Link to="/docs" className="mt-5 inline-flex items-center gap-2 text-sm font-medium text-primary hover:underline"><ArrowLeft size={14} />Back to documentation</Link></div> : guide && !query.trim() ? <GuideArticle guide={guide} /> : <DocumentationHome results={results} query={query} />}
      </section>
      {guide && !query.trim() ? <TableOfContents guide={guide} /> : <aside className="hidden 2xl:block"><div className="sticky top-6 rounded-xl border border-border bg-card p-4"><p className="text-xs font-semibold text-foreground">Start building</p><p className="mt-2 text-xs leading-5 text-muted-foreground">Follow the setup and first-app guides to connect a new Docker service.</p><Link to="/docs/first-application" className="mt-3 inline-flex items-center gap-1 text-xs font-medium text-primary">Developer guide <ArrowRight size={13} /></Link></div></aside>}
    </div>
  </div>
}
