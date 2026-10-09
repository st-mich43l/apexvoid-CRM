import { useLocation } from 'react-router-dom'

const apiBase = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

export function ExternalApplicationPage() {
  const location = useLocation()
  const source = `${apiBase}${location.pathname}${location.search}`
  return <iframe title="External ApexVoid application" src={source} className="h-[calc(100dvh-8rem)] min-h-[640px] w-full rounded-xl border border-border bg-background" />
}
