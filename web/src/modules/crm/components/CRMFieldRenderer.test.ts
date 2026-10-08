import { describe, expect, it } from 'vitest'
import { normalizeCRMFieldValue } from './CRMFieldValue'

describe('normalizeCRMFieldValue', () => {
  it('keeps numeric custom values numeric and safe', () => {
    expect(normalizeCRMFieldValue('decimal', '12.5')).toEqual({ value: 12.5 })
    expect(normalizeCRMFieldValue('integer', '42')).toEqual({ value: 42 })
    expect(normalizeCRMFieldValue('integer', '1.5').error).toContain('whole')
    expect(normalizeCRMFieldValue('integer', '9007199254740992').error).toContain('safe')
  })

  it('distinguishes an explicit clear from a boolean false', () => {
    expect(normalizeCRMFieldValue('decimal', '')).toEqual({ clear: true })
    expect(normalizeCRMFieldValue('boolean', false)).toEqual({ value: false })
  })
})
