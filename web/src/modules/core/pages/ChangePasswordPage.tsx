import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { api } from '../../../core/api/client'
import { useAuth } from '../../../core/auth/context'
import { externalSessionContinuation, safeExternalReturnTo } from '../../../core/auth/externalReturn'
import { Button, Card, inputClass } from '../../../components/ui'

export function ChangePasswordPage() {
  const { reload, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const returnTo = safeExternalReturnTo(new URLSearchParams(location.search).get('return_to'))
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setError('')
    if (next.length < 12) { setError('Password must be at least 12 characters.'); return }
    setSaving(true)
    try { await api.auth.changePassword(current, next); await reload(); navigate(returnTo ? externalSessionContinuation(returnTo) : '/', { replace: true }) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to change password') } finally { setSaving(false) }
  }
  return <main className="grid min-h-dvh place-items-center bg-background px-4 py-8 text-foreground sm:px-6"><Card className="w-full max-w-md p-6 sm:p-8"><form onSubmit={submit}><p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-primary">First sign-in</p><h1 className="mb-3 text-3xl font-semibold tracking-tight">Change your password</h1><p className="mb-8 text-sm leading-6 text-muted-foreground">For security, choose a new password before continuing.</p><label className="mb-4 block text-sm font-medium text-foreground">Current password<input required type="password" value={current} onChange={event => setCurrent(event.target.value)} className={inputClass} /></label><label className="mb-6 block text-sm font-medium text-foreground">New password<input required minLength={12} type="password" value={next} onChange={event => setNext(event.target.value)} className={inputClass} /></label>{error && <p className="mb-4 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}<Button type="submit" disabled={saving} className="w-full min-h-11">{saving ? 'Saving…' : 'Set new password'}</Button><Button type="button" variant="outline" onClick={() => void logout()} className="mt-3 w-full">Sign out</Button></form></Card></main>
}
