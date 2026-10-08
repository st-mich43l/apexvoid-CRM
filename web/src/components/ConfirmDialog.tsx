import { useState, type ReactNode } from 'react'

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
    <div role="dialog" aria-modal="true" aria-labelledby="confirm-dialog-title" className="w-full max-w-md rounded-2xl border border-border bg-card p-6 shadow-xl">
      <h2 id="confirm-dialog-title" className="text-lg font-semibold text-foreground">{title}</h2>
      <div className="mt-2 text-sm text-muted-foreground">{description}</div>
      <div className="mt-6 flex justify-end gap-3">
        <button type="button" onClick={onCancel} disabled={pending} className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted disabled:opacity-50">Cancel</button>
        <button type="button" onClick={() => void confirm()} disabled={pending} className={`rounded-lg px-4 py-2 text-sm font-semibold disabled:opacity-50 ${destructive ? 'bg-destructive text-destructive-foreground' : 'bg-primary text-primary-foreground'}`}>{pending ? 'Working…' : confirmLabel}</button>
      </div>
    </div>
  </div>
}
