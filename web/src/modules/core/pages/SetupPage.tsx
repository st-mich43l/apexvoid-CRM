import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../../../core/api/client'
import { useWorkspace } from '../../../core/workspace/context'
import { Button, Card, PageContainer, PageHeader, inputClass } from '../../../components/ui'

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

  return <PageContainer variant="detail" className="py-8 md:py-14"><PageHeader eyebrow="Welcome to ApexVoid" title="Set up your company" description="Give your team a home base. We’ll create your organization and its first workspace together." /><Card className="max-w-2xl p-6 md:p-8"><form onSubmit={submit}><label className="mb-5 block text-sm font-medium text-foreground">Company name<input required value={organizationName} onChange={event => setOrganizationName(event.target.value)} placeholder="ApexVoid Technologies" className={inputClass} /></label><label className="mb-5 block text-sm font-medium text-foreground">First workspace <span className="font-normal text-muted-foreground">(optional)</span><input value={workspaceName} onChange={event => setWorkspaceName(event.target.value)} placeholder="Main Workspace" className={inputClass} /></label><label className="mb-6 block text-sm font-medium text-foreground">Timezone<select value={timezone} onChange={event => setTimezone(event.target.value)} className={inputClass}><option value="UTC">UTC</option><option value="Asia/Ho_Chi_Minh">Asia/Ho_Chi_Minh</option><option value="Asia/Singapore">Asia/Singapore</option><option value="America/New_York">America/New_York</option><option value="Europe/London">Europe/London</option></select></label>{error && <p className="mb-5 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</p>}<Button type="submit" disabled={saving} className="w-full min-h-11">{saving ? 'Preparing your workspace…' : 'Create company workspace'}</Button></form></Card></PageContainer>
}
