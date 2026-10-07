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
  return <main className="grid min-h-screen place-items-center bg-ink px-6 text-zinc-100"><form onSubmit={submit} className="w-full max-w-md rounded-2xl border border-line bg-panel p-8 shadow-2xl"><p className="mb-2 text-sm uppercase tracking-[0.25em] text-accent">ApexVoid CRM</p><h1 className="mb-8 text-3xl font-semibold">Sign in</h1><label className="mb-4 block text-sm text-zinc-400">Email<input required type="email" value={email} onChange={event => setEmail(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent" /></label><label className="mb-6 block text-sm text-zinc-400">Password<input required type="password" value={password} onChange={event => setPassword(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent" /></label>{error && <p className="mb-4 rounded-lg border border-red-400/30 bg-red-400/10 p-3 text-sm text-red-200">{error}</p>}<button disabled={loading} className="w-full rounded-lg bg-accent px-4 py-3 font-semibold text-ink disabled:opacity-50">{loading ? 'Loading…' : 'Sign in'}</button></form></main>
}
