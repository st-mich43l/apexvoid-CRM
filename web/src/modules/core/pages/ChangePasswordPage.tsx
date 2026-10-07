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
  return <main className="grid min-h-screen place-items-center bg-ink px-6 text-zinc-100"><form onSubmit={submit} className="w-full max-w-md rounded-2xl border border-line bg-panel p-8"><p className="mb-2 text-sm uppercase tracking-[0.25em] text-accent">First sign-in</p><h1 className="mb-3 text-3xl font-semibold">Change your password</h1><p className="mb-8 text-sm text-zinc-500">For security, choose a new password before continuing.</p><label className="mb-4 block text-sm text-zinc-400">Current password<input required type="password" value={current} onChange={event => setCurrent(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100" /></label><label className="mb-6 block text-sm text-zinc-400">New password<input required minLength={12} type="password" value={next} onChange={event => setNext(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100" /></label>{error && <p className="mb-4 rounded-lg border border-red-400/30 bg-red-400/10 p-3 text-sm text-red-200">{error}</p>}<button disabled={saving} className="w-full rounded-lg bg-accent px-4 py-3 font-semibold text-ink disabled:opacity-50">{saving ? 'Saving…' : 'Set new password'}</button><button type="button" onClick={() => void logout()} className="mt-4 w-full rounded-lg border border-line px-4 py-3 text-sm text-zinc-400 hover:text-zinc-100">Sign out</button></form></main>
}
