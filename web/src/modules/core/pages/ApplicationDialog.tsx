import { useEffect, useId, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X, Blocks } from 'lucide-react'

type Props = {
  title: string
  description: string
  children: ReactNode
  onClose: () => void
  busy?: boolean
}

export function ApplicationDialog({ title, description, children, onClose, busy = false }: Props) {
  const titleId = useId()
  const descriptionId = useId()
  const panelRef = useRef<HTMLDivElement>(null)
  const openerRef = useRef<HTMLElement | null>(document.activeElement instanceof HTMLElement ? document.activeElement : null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const closeCallback = useRef(onClose)
  const isBusy = useRef(busy)
  useEffect(() => { closeCallback.current = onClose; isBusy.current = busy }, [onClose, busy])

  useEffect(() => {
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    closeRef.current?.focus()

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (!isBusy.current) { event.preventDefault(); closeCallback.current() }
        return
      }
      if (event.key !== 'Tab' || !panelRef.current) return
      const focusable = Array.from(panelRef.current.querySelectorAll<HTMLElement>(
        'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])',
      )).filter(element => !element.closest('[hidden]') && window.getComputedStyle(element).visibility !== 'hidden')
      if (!focusable.length) { event.preventDefault(); return }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && (document.activeElement === first || !panelRef.current.contains(document.activeElement))) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && (document.activeElement === last || !panelRef.current.contains(document.activeElement))) {
        event.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = previousOverflow
      openerRef.current?.focus()
    }
  }, [])

  return createPortal(
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-slate-950/65 px-3 py-5 backdrop-blur-[6px] sm:px-6" role="presentation" onMouseDown={event => { if (event.target === event.currentTarget && !busy) onClose() }}>
      <div ref={panelRef} role="dialog" aria-modal="true" aria-labelledby={titleId} aria-describedby={descriptionId} className="flex max-h-[min(94dvh,960px)] w-full max-w-3xl flex-col overflow-hidden rounded-[28px] border border-border/80 bg-card shadow-[0_32px_120px_rgba(0,0,0,0.38)]">
        <div className="relative shrink-0 overflow-hidden border-b border-border bg-gradient-to-r from-primary/[0.09] via-background to-background px-5 py-5 sm:px-7 sm:py-6">
          <div className="pointer-events-none absolute -right-12 -top-20 h-48 w-48 rounded-full bg-primary/10 blur-3xl" aria-hidden="true" />
          <div className="relative flex items-start gap-4">
            <div className="grid h-12 w-12 shrink-0 place-items-center rounded-2xl border border-primary/20 bg-primary/10 text-primary shadow-sm"><Blocks size={23} /></div>
            <div className="min-w-0 flex-1"><h2 id={titleId} className="text-xl font-semibold tracking-tight text-foreground">{title}</h2><p id={descriptionId} className="mt-1 max-w-xl text-sm leading-6 text-muted-foreground">{description}</p></div>
            <button ref={closeRef} type="button" aria-label="Close application dialog" onClick={onClose} disabled={busy} className="grid h-9 w-9 shrink-0 place-items-center rounded-xl border border-border bg-background/75 text-muted-foreground transition hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary disabled:opacity-50"><X size={18} /></button>
          </div>
        </div>
        <div className="min-h-0 overflow-y-auto overscroll-contain px-5 py-5 sm:px-7 sm:py-6">{children}</div>
      </div>
    </div>,
    document.body,
  )
}
