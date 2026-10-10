import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { LoaderCircle, ShieldCheck } from 'lucide-react'
import { useAuth } from '../../../core/auth/context'
import { safeExternalReturnTo } from '../../../core/auth/externalReturn'
import { Card } from '../../../components/ui'

// This is a first-party page, not part of the external app bundle. A browser
// navigation to /apps/* cannot call the HttpOnly refresh-cookie endpoint, but
// this page can use the existing AuthProvider refresh and resume the launch.
export function ExternalAuthContinuePage() {
  const { loading, user } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const returnTo = safeExternalReturnTo(new URLSearchParams(location.search).get('return_to'))

  useEffect(() => {
    if (loading) return
    if (!returnTo) {
      navigate('/', { replace: true })
    } else if (!user) {
      navigate('/login?return_to=' + encodeURIComponent(returnTo), { replace: true })
    } else if (user.must_change_password) {
      navigate('/change-password?return_to=' + encodeURIComponent(returnTo), { replace: true })
    } else {
      // Full document navigation is required: /apps/* is gateway-proxied,
      // not a client-side React route. No credentials enter this URL.
      window.location.replace(returnTo)
    }
  }, [loading, user, returnTo, navigate])

  return <main className="grid min-h-dvh place-items-center bg-background p-5 text-foreground">
    <Card className="w-full max-w-sm space-y-4 p-6 text-center shadow-xl">
      <span className="mx-auto grid h-12 w-12 place-items-center rounded-2xl bg-primary/10 text-primary"><ShieldCheck size={25} /></span>
      <h1 className="text-lg font-semibold">Opening your application</h1>
      <p className="text-sm leading-6 text-muted-foreground">Checking your Enterprise session and workspace access…</p>
      <LoaderCircle className="mx-auto animate-spin text-primary" size={20} aria-label="Checking session" />
    </Card>
  </main>
}
