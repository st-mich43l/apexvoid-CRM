/* eslint-disable react-refresh/only-export-components */
import type { Dispatch, SetStateAction } from 'react'
import { inputClass } from './ui'

export type EffectiveField = {
  key: string
  label: string
  type: 'string' | 'text' | 'decimal' | 'integer' | 'boolean' | 'date' | 'enum'
  description?: string
  required: boolean
  read_only: boolean
  source: 'built_in' | 'custom'
  default_value?: unknown
  options: string[]
  visible: boolean
}

export function defaultCustomValues(fields: EffectiveField[]) {
  return Object.fromEntries(fields.filter(field => field.source === 'custom' && field.visible && field.default_value !== undefined).map(field => [field.key, field.default_value]))
}

export function CustomFieldControls({ fields, values, setValues, disabled = false }: { fields: EffectiveField[]; values: Record<string, unknown>; setValues: Dispatch<SetStateAction<Record<string, unknown>>>; disabled?: boolean }) {
  const setValue = (key: string, value: unknown) => setValues(current => ({ ...current, [key]: value }))
  return <>{fields.filter(field => field.source === 'custom' && field.visible).map(field => <label key={field.key} className="mb-4 block text-sm font-medium text-foreground">{field.label}{field.required && <span className="text-destructive"> *</span>}{field.description && <span className="mt-1 block text-xs font-normal leading-5 text-muted-foreground">{field.description}</span>}{field.type === 'text' ? <textarea required={field.required} disabled={disabled || field.read_only} value={String(values[field.key] ?? '')} onChange={event => setValue(field.key, event.target.value)} className={`${inputClass} min-h-24 py-2`} /> : field.type === 'enum' ? <select required={field.required} disabled={disabled || field.read_only} value={String(values[field.key] ?? '')} onChange={event => setValue(field.key, event.target.value)} className={inputClass}><option value="">Choose…</option>{field.options.map(option => <option key={option} value={option}>{option}</option>)}</select> : field.type === 'boolean' ? <span className="mt-2 flex items-center gap-2 text-sm font-normal"><input type="checkbox" checked={values[field.key] === true} disabled={disabled || field.read_only} onChange={event => setValue(field.key, event.target.checked)} /> Yes</span> : <input required={field.required} disabled={disabled || field.read_only} type={field.type === 'date' ? 'date' : field.type === 'decimal' || field.type === 'integer' ? 'number' : 'text'} step={field.type === 'decimal' ? '0.01' : undefined} value={String(values[field.key] ?? '')} onChange={event => setValue(field.key, field.type === 'decimal' || field.type === 'integer' ? (event.target.value === '' ? undefined : Number(event.target.value)) : event.target.value)} className={inputClass} />}</label>)}</>
}
