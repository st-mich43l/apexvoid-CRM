import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api, type EffectiveField } from '../../../core/api/client'
import { useAuth } from '../../../core/auth/context'
import { useWorkspace } from '../../../core/workspace/context'
import { userWorkspaceQueryKey } from '../../../core/workspace/query'
import { normalizeCRMFieldValue } from './CRMFieldValue'

const input = 'w-full rounded-lg border border-border bg-background px-3 py-2 text-sm'

function useCRMKey(part: string) {
  const { user } = useAuth()
  const { activeWorkspaceId } = useWorkspace()
  return userWorkspaceQueryKey(user?.id, activeWorkspaceId, 'crm', part)
}

export function CRMFieldRenderer({ entity, values, onChange }: { entity: string; values: Record<string, unknown>; onChange: (value: Record<string, unknown>) => void }) {
  const schema = useQuery({ queryKey: useCRMKey(`schema-${entity}`), queryFn: () => api.customization.schema(entity) })
  const fields = useMemo(() => schema.data?.fields.filter(field => field.source === 'custom' && field.visible) ?? [], [schema.data])
  const sections = schema.data?.sections ?? []
  const [validation, setValidation] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!fields.length) return
    const defaults = { ...values }
    let changed = false
    for (const field of fields) {
      if (defaults[field.key] === undefined && field.default_value !== undefined) {
        defaults[field.key] = field.default_value
        changed = true
      }
    }
    if (changed) onChange(defaults)
  }, [fields, values, onChange])

  const update = (field: EffectiveField, raw: string | boolean) => {
    const result = normalizeCRMFieldValue(field.type, raw)
    if (result.error) {
      setValidation(current => ({ ...current, [field.key]: result.error ?? '' }))
      return
    }
    if (result.clear) {
      const next = { ...values }
      delete next[field.key]
      setValidation(current => ({ ...current, [field.key]: '' }))
      onChange(next)
      return
    }
    setValidation(current => ({ ...current, [field.key]: '' }))
    onChange({ ...values, [field.key]: result.value })
  }

  const renderField = (field: EffectiveField) => <label key={field.key} className="grid gap-1 text-sm"><span>{field.label}{field.required && ' *'}</span>{field.type === 'enum' ? <select required={field.required} disabled={field.read_only} value={String(values[field.key] ?? '')} onChange={event => update(field, event.target.value)} className={input}><option value="">Choose…</option>{field.options.map(option => <option key={option}>{option}</option>)}</select> : field.type === 'boolean' ? <input disabled={field.read_only} type="checkbox" checked={values[field.key] === true} onChange={event => update(field, event.target.checked)} /> : <input required={field.required} disabled={field.read_only} type={field.type === 'date' ? 'date' : field.type === 'decimal' || field.type === 'integer' ? 'number' : 'text'} step={field.type === 'integer' ? '1' : field.type === 'decimal' ? 'any' : undefined} value={String(values[field.key] ?? '')} onChange={event => update(field, event.target.value)} className={input} />} {field.description && <span className="text-xs text-muted-foreground">{field.description}</span>}{validation[field.key] && <span className="text-xs text-destructive">{validation[field.key]}</span>}</label>

  return fields.length ? <fieldset className="grid gap-4 border-t border-border pt-4 md:col-span-2"><legend className="text-sm font-semibold">Custom fields</legend>{sections.map(section => { const sectionFields = fields.filter(field => field.section_id === section.id); return sectionFields.length ? <div key={section.id} className="grid gap-3"><h3 className="text-xs font-semibold uppercase tracking-[.16em] text-muted-foreground">{section.name}</h3><div className="grid gap-3 md:grid-cols-2">{sectionFields.map(renderField)}</div></div> : null })}<div className="grid gap-3 md:grid-cols-2">{fields.filter(field => !field.section_id || !sections.some(section => section.id === field.section_id)).map(renderField)}</div></fieldset> : null
}
