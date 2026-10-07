import { useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../../../core/auth/context'

export function LoginPage() {
  const { user, login, loading } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  if (!loading && user) return <Navigate to={(location.state as { from?: string } | null)?.from ?? '/'} replace />
  const submit = async (event: React.FormEvent) => { event.preventDefault(); setError(''); try { await login(email, password); navigate('/', { replace: true }) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Unable to sign in') } }
  return <main className="grid min-h-screen place-items-center bg-background px-6 text-foreground"><form onSubmit={submit} className="w-full max-w-md rounded-2xl border border-border bg-card p-8 shadow-2xl"><p className="mb-2 text-sm uppercase tracking-[0.25em] text-primary">ApexVoid CRM</p><h1 className="mb-8 text-3xl font-semibold">Sign in</h1><label className="mb-4 block text-sm text-muted-foreground">Email or username<input required type="text" value={email} onChange={event => setEmail(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary" /></label><label className="mb-6 block text-sm text-muted-foreground">Password<input required type="password" value={password} onChange={event => setPassword(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary" /></label>{error && <p className="mb-4 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}<button disabled={loading} className="w-full rounded-lg bg-primary px-4 py-3 font-semibold text-primary-foreground disabled:opacity-50">{loading ? 'Loading…' : 'Sign in'}</button></form></main>
}
