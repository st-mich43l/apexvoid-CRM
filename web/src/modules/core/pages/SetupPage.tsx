import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../../../core/api/client'
import { useWorkspace } from '../../../core/workspace/context'

export function SetupPage() {
  const navigate = useNavigate()
  const { reload } = useWorkspace()
  const [organizationName, setOrganizationName] = useState('')
  const [workspaceName, setWorkspaceName] = useState('')
  const [timezone, setTimezone] = useState(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.setup.createOrganization({ organization_name: organizationName, workspace_name: workspaceName, timezone })
      await reload()
      navigate('/', { replace: true })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to complete setup')
    } finally { setSaving(false) }
  }

  return <div className="mx-auto max-w-2xl py-8 md:py-16"><div className="mb-10"><p className="mb-3 text-sm font-medium uppercase tracking-[0.2em] text-accent">Welcome to ApexVoid</p><h1 className="text-3xl font-semibold tracking-tight md:text-4xl">Set up your company</h1><p className="mt-3 max-w-xl text-zinc-500">Give your team a home base. We’ll create your organization and its first workspace together.</p></div><form onSubmit={submit} className="rounded-2xl border border-line bg-panel p-6 shadow-2xl md:p-8"><label className="mb-5 block text-sm text-zinc-400">Company name<input required value={organizationName} onChange={event => setOrganizationName(event.target.value)} placeholder="ApexVoid Technologies" className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent" /></label><label className="mb-5 block text-sm text-zinc-400">First workspace <span className="text-zinc-600">(optional)</span><input value={workspaceName} onChange={event => setWorkspaceName(event.target.value)} placeholder="Main Workspace" className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent" /></label><label className="mb-6 block text-sm text-zinc-400">Timezone<select value={timezone} onChange={event => setTimezone(event.target.value)} className="mt-2 w-full rounded-lg border border-line bg-ink px-3 py-3 text-zinc-100 outline-none focus:border-accent"><option value="UTC">UTC</option><option value="Asia/Ho_Chi_Minh">Asia/Ho_Chi_Minh</option><option value="Asia/Singapore">Asia/Singapore</option><option value="America/New_York">America/New_York</option><option value="Europe/London">Europe/London</option></select></label>{error && <p className="mb-5 rounded-lg border border-red-400/30 bg-red-400/10 p-3 text-sm text-red-200">{error}</p>}<button disabled={saving} className="w-full rounded-lg bg-accent px-4 py-3 font-semibold text-ink disabled:opacity-50">{saving ? 'Preparing your workspace…' : 'Create company workspace'}</button></form></div>
}
