import { useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../../../core/auth/context'
import { Button, Card, inputClass } from '../../../components/ui'

export function LoginPage() {
  const { user, login, loading } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  if (!loading && user) return <Navigate to={(location.state as { from?: string } | null)?.from ?? '/'} replace />
  const submit = async (event: React.FormEvent) => { event.preventDefault(); setError(''); try { await login(email, password); navigate('/', { replace: true }) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to sign in') } }
  return <main className="grid min-h-dvh place-items-center bg-background px-4 py-8 text-foreground sm:px-6"><Card className="w-full max-w-md p-6 shadow-xl sm:p-8"><form onSubmit={submit}><p className="mb-2 text-xs font-semibold uppercase tracking-[0.2em] text-primary">ApexVoid CRM</p><h1 className="mb-8 text-3xl font-semibold tracking-tight">Sign in</h1><label className="mb-4 block text-sm font-medium text-foreground">Email or username<input required type="text" value={email} onChange={event => setEmail(event.target.value)} className={inputClass} /></label><label className="mb-6 block text-sm font-medium text-foreground">Password<input required type="password" value={password} onChange={event => setPassword(event.target.value)} className={inputClass} /></label>{error && <p className="mb-4 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}<Button type="submit" disabled={loading} className="w-full min-h-11">{loading ? 'Loading…' : 'Sign in'}</Button></form></Card></main>
}
