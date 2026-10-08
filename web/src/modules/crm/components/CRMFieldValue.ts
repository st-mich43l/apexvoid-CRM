import type { EffectiveField } from '../../../core/api/client'

export function normalizeCRMFieldValue(type: EffectiveField['type'], raw: string | boolean): { value?: unknown; clear?: boolean; error?: string } {
  if (type === 'boolean') return { value: raw }
  if (raw === '') return { clear: true }
  if (type === 'integer' || type === 'decimal') {
    const number = Number(raw)
    const invalid = !Number.isFinite(number) || (type === 'integer' && (!Number.isSafeInteger(number) || !Number.isInteger(number)))
    if (invalid) return { error: type === 'integer' ? 'Enter a safe whole number.' : 'Enter a finite number.' }
    return { value: number }
  }
  return { value: raw }
}
