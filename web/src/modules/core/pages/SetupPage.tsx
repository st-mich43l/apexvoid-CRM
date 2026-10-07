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

  return <div className="mx-auto max-w-2xl py-8 md:py-16"><div className="mb-10"><p className="mb-3 text-sm font-medium uppercase tracking-[0.2em] text-primary">Welcome to ApexVoid</p><h1 className="text-3xl font-semibold tracking-tight md:text-4xl">Set up your company</h1><p className="mt-3 max-w-xl text-muted-foreground">Give your team a home base. We’ll create your organization and its first workspace together.</p></div><form onSubmit={submit} className="rounded-2xl border border-border bg-card p-6 shadow-2xl md:p-8"><label className="mb-5 block text-sm text-muted-foreground">Company name<input required value={organizationName} onChange={event => setOrganizationName(event.target.value)} placeholder="ApexVoid Technologies" className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary" /></label><label className="mb-5 block text-sm text-muted-foreground">First workspace <span className="text-muted-foreground/70">(optional)</span><input value={workspaceName} onChange={event => setWorkspaceName(event.target.value)} placeholder="Main Workspace" className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary" /></label><label className="mb-6 block text-sm text-muted-foreground">Timezone<select value={timezone} onChange={event => setTimezone(event.target.value)} className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-3 text-foreground outline-none focus:border-primary"><option value="UTC">UTC</option><option value="Asia/Ho_Chi_Minh">Asia/Ho_Chi_Minh</option><option value="Asia/Singapore">Asia/Singapore</option><option value="America/New_York">America/New_York</option><option value="Europe/London">Europe/London</option></select></label>{error && <p className="mb-5 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}<button disabled={saving} className="w-full rounded-lg bg-primary px-4 py-3 font-semibold text-primary-foreground disabled:opacity-50">{saving ? 'Preparing your workspace…' : 'Create company workspace'}</button></form></div>
}
