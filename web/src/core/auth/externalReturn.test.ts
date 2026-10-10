import { describe, expect, it } from 'vitest'
import { externalSessionContinuation, safeExternalReturnTo } from './externalReturn'

describe('external app SSO return targets', () => {
  it('keeps the original app and workspace but never the browser origin', () => {
    const returnTo = '/apps/photobooth?workspace_id=603bc0c1-386d-4740-9bf1-2998fabd727e'
    expect(safeExternalReturnTo(returnTo)).toBe(returnTo)
    expect(safeExternalReturnTo('/apps/cafe/history?page=2')).toBe('/apps/cafe/history?page=2')
    const continuation = externalSessionContinuation(returnTo)
    expect(new URL(continuation, 'https://enterprise.example').searchParams.get('return_to')).toBe(returnTo)
  })

  it.each([
    '', 'https://example.com', '//evil.example/apps/photobooth',
    '/login?return_to=//evil.example', '/api/apps/photobooth',
    '/apps/', '/apps/photobooth\\evil.example',
    '/apps/photobooth/%2f%2fevil.example',
    '/apps/photobooth/%5celsewhere',
    '/apps/photobooth/../admin',
    '/apps/photobooth/%2e%2e/admin',
    '/apps/photobooth/%252e%252e/admin',
    '/apps/photobooth#fragment',
  ])('rejects untrusted app return target %s', (value) => {
    expect(safeExternalReturnTo(value)).toBeNull()
  })
})
