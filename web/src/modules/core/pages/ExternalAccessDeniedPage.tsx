import { Link } from 'react-router-dom'
import { ShieldAlert } from 'lucide-react'
import { Button, Card } from '../../../components/ui'

// This page is deliberately generic: authorization failures must not reveal
// whether an external app or workspace is registered to another organization.
export function ExternalAccessDeniedPage() {
  return <main className="grid min-h-dvh place-items-center bg-background px-4 py-8 text-foreground">
    <Card className="w-full max-w-md space-y-4 p-7 text-center shadow-xl">
      <span className="mx-auto grid h-12 w-12 place-items-center rounded-2xl bg-destructive/10 text-destructive"><ShieldAlert size={24} /></span>
      <h1 className="text-xl font-semibold">Application access unavailable</h1>
      <p className="text-sm leading-6 text-muted-foreground">This application is not enabled for your workspace, or your account does not have permission to open it. Contact your workspace administrator if you need access.</p>
      <Button asChild={false} type="button" variant="outline" className="min-h-10 w-full" onClick={() => { window.location.assign('/applications') }}>Return to applications</Button>
      <p className="text-xs text-muted-foreground">Your Enterprise session remains private. Access is checked on every request.</p>
    </Card>
  </main>
}
