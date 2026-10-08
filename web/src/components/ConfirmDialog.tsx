import { useEffect, useState, type ReactNode } from 'react'
import { Button } from './ui'

type ConfirmDialogProps = {
  open: boolean
  title: string
  description: ReactNode
  confirmLabel: string
  destructive?: boolean
  onCancel: () => void
  onConfirm: () => Promise<void> | void
}

export function ConfirmDialog({ open, title, description, confirmLabel, destructive = false, onCancel, onConfirm }: ConfirmDialogProps) {
  const [pending, setPending] = useState(false)
  useEffect(() => {
    if (!open) return
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !pending) onCancel()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onCancel, open, pending])
  if (!open) return null

  const confirm = async () => {
    setPending(true)
    try {
      await onConfirm()
    } finally {
      setPending(false)
    }
  }

  return <div className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4" role="presentation" onMouseDown={event => { if (event.target === event.currentTarget) onCancel() }}>
    <div role="dialog" aria-modal="true" aria-labelledby="confirm-dialog-title" aria-describedby="confirm-dialog-description" className="max-h-[calc(100dvh-2rem)] w-full max-w-md overflow-y-auto rounded-2xl border border-border bg-card p-6 shadow-xl">
      <h2 id="confirm-dialog-title" className="text-lg font-semibold text-foreground">{title}</h2>
      <div id="confirm-dialog-description" className="mt-2 text-sm text-muted-foreground">{description}</div>
      <div className="mt-6 flex justify-end gap-3">
        <Button type="button" variant="outline" onClick={onCancel} disabled={pending}>Cancel</Button>
        <Button type="button" variant={destructive ? 'destructive' : 'primary'} onClick={() => void confirm()} disabled={pending}>{pending ? 'Working…' : confirmLabel}</Button>
      </div>
    </div>
  </div>
}
