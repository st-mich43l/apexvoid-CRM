import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../../../core/api/client'
import { useAuth } from '../../../core/auth/context'

export function ChangePasswordPage() {
  const { reload, logout } = useAuth()
  const navigate = useNavigate()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setError('')
    if (next.length < 12) { setError('Password must be at least 12 characters.'); return }
    setSaving(true)
    try { await api.auth.changePassword(current, next); await reload(); navigate('/', { replace: true }) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to change password') } finally { setSaving(false) }
  }
  return <main className="grid min-h-screen place-items-center bg-background px-6 text-foreground"><form onSubmit={submit} className="w-full max-w-md rounded-2xl border border-border bg-card p-8"><p className="mb-2 text-sm uppercase tracking-[0.25em] text-primary">First sign-in</p><h1 className="mb-3 text-3xl font-semibold">Change your password</h1><p className="mb-8 text-sm text-muted-foreground">For security, choose a new password before continuing.</p><label className="mb-4 block text-sm text-muted-foreground">Current password<input required type="password" value={current} onChange={event => setCurrent(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground" /></label><label className="mb-6 block text-sm text-muted-foreground">New password<input required minLength={12} type="password" value={next} onChange={event => setNext(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground" /></label>{error && <p className="mb-4 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}<button disabled={saving} className="w-full rounded-lg bg-primary px-4 py-3 font-semibold text-primary-foreground disabled:opacity-50">{saving ? 'Saving…' : 'Set new password'}</button><button type="button" onClick={() => void logout()} className="mt-4 w-full rounded-lg border border-border px-4 py-3 text-sm text-muted-foreground hover:text-foreground">Sign out</button></form></main>
}
