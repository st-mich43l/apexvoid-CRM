// A return target is accepted only when it names a same-origin Enterprise
// external-application document. Never use an arbitrary URL as a login return.
export function safeExternalReturnTo(raw: string | null): string | null {
  if (!raw || raw.length > 2048 || !raw.startsWith('/apps/') || raw.startsWith('//') ||
      raw.includes('\\') || raw.includes('#') ||
      [...raw].some(char => char.charCodeAt(0) < 32 || char.charCodeAt(0) === 127)) return null
  const rawPath = raw.split(/[?#]/, 1)[0]
  if (/%(?:2e|2f|5c|00|25)/i.test(rawPath)) return null
  if (rawPath.split('/').some(segment => segment === '.' || segment === '..')) return null
  try {
    const parsed = new URL(raw, 'https://apexvoid.invalid')
    if (parsed.origin !== 'https://apexvoid.invalid' ||
        !/^\/apps\/[a-z0-9][a-z0-9_.-]*(?:\/|$)/i.test(parsed.pathname)) return null
    return parsed.pathname + parsed.search
  } catch {
    return null
  }
}

export function externalSessionContinuation(returnTo: string): string {
  return '/auth/continue?return_to=' + encodeURIComponent(returnTo)
}
