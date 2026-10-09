import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Navigate } from 'react-router-dom'
import { Boxes, Package, Plus, Search, ArchiveRestore, Archive, Pencil } from 'lucide-react'
import { api, type ERPProduct, type ERPProductInput } from '../../core/api/client'
import { useAuth } from '../../core/auth/context'
import { useWorkspace } from '../../core/workspace/context'
import { userWorkspaceQueryKey } from '../../core/workspace/query'
import type { AppModule } from '../../framework/module/types'
import { Button, Card, PageContainer, PageHeader } from '../../components/ui'

const empty: ERPProductInput = { sku: '', name: '', description: '', kind: 'good', unit: 'unit', unit_price: '0.0000', currency: 'USD' }
const inputClass = 'min-w-0 w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'

function ProductEditor({ initial, pending, error, onCancel, onSave }: {
  initial?: ERPProduct; pending: boolean; error: string
  onCancel: () => void; onSave: (input: ERPProductInput, version?: number) => void
}) {
  const [draft, setDraft] = useState<ERPProductInput>(initial ? {
    sku: initial.sku, name: initial.name, description: initial.description,
    kind: initial.kind, unit: initial.unit, unit_price: initial.unit_price, currency: initial.currency,
  } : { ...empty })
  const update = <K extends keyof ERPProductInput>(key: K, value: ERPProductInput[K]) =>
    setDraft(current => ({ ...current, [key]: value }))
  const submit = (event: FormEvent) => {
    event.preventDefault()
    onSave({
      ...draft,
      sku: draft.sku.trim().toUpperCase(),
      name: draft.name.trim(),
      currency: draft.currency.trim().toUpperCase(),
      unit_price: draft.unit_price.trim(),
    }, initial?.version)
  }
  return <Card className="mb-5 p-5 sm:p-6">
    <form onSubmit={submit} className="space-y-5">
      <div className="flex items-center justify-between gap-3">
        <div><h2 className="font-semibold">{initial ? 'Edit product' : 'New product or service'}</h2>
          <p className="mt-1 text-sm text-muted-foreground">Prices use exact decimal values. Inventory and tax rules will be added in later ERP modules.</p></div>
        <Button type="button" variant="ghost" onClick={onCancel}>Close</Button>
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="text-sm font-medium">SKU<input required maxLength={64} className={inputClass} value={draft.sku} onChange={event => update('sku', event.target.value)} placeholder="ITEM-1001" /></label>
        <label className="text-sm font-medium">Name<input required maxLength={200} className={inputClass} value={draft.name} onChange={event => update('name', event.target.value)} placeholder="Product name" /></label>
        <label className="text-sm font-medium">Type<select className={inputClass} value={draft.kind} onChange={event => update('kind', event.target.value as ERPProductInput['kind'])}><option value="good">Physical good</option><option value="service">Service</option></select></label>
        <label className="text-sm font-medium">Unit<select className={inputClass} value={draft.unit} onChange={event => update('unit', event.target.value as ERPProductInput['unit'])}><option value="unit">Unit</option><option value="hour">Hour</option><option value="kg">Kilogram</option></select></label>
        <label className="text-sm font-medium">Unit price<input required inputMode="decimal" pattern="(0|[1-9][0-9]{0,15})(\\.[0-9]{1,4})?" title="Positive decimal with up to four fractional digits" className={inputClass} value={draft.unit_price} onChange={event => update('unit_price', event.target.value)} /></label>
        <label className="text-sm font-medium">Currency<input required maxLength={3} minLength={3} pattern="[A-Za-z]{3}" className={inputClass} value={draft.currency} onChange={event => update('currency', event.target.value)} placeholder="USD" /></label>
      </div>
      <label className="block text-sm font-medium">Description<textarea maxLength={2000} rows={3} className={inputClass} value={draft.description} onChange={event => update('description', event.target.value)} placeholder="Optional product details" /></label>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      <div className="flex justify-end gap-2"><Button type="button" variant="outline" onClick={onCancel}>Cancel</Button><Button disabled={pending}>{pending ? 'Saving…' : initial ? 'Save changes' : 'Create product'}</Button></div>
    </form>
  </Card>
}

export function ERPProductsPage() {
  const { user } = useAuth()
  const { activeWorkspaceId, can } = useWorkspace()
  const client = useQueryClient()
  const canRead = can('erp.product.read')
  const [search, setSearch] = useState('')
  const [includeArchived, setIncludeArchived] = useState(false)
  const [page, setPage] = useState(1)
  const [editor, setEditor] = useState<ERPProduct | 'new' | null>(null)
  const [error, setError] = useState('')
  const key = userWorkspaceQueryKey(user?.id, activeWorkspaceId, 'erp', 'products')
  const products = useQuery({ queryKey: [...key, search, includeArchived, page], queryFn: () => api.erp.products({ search, page, limit: 25, include_archived: includeArchived }), enabled: Boolean(activeWorkspaceId && canRead), retry: false })
  const refresh = () => void client.invalidateQueries({ queryKey: key })
  const create = useMutation({ mutationFn: api.erp.createProduct, onSuccess: () => { setEditor(null); refresh() } })
  const update = useMutation({ mutationFn: ({ item, input }: { item: ERPProduct; input: ERPProductInput }) => api.erp.updateProduct(item.id, { ...input, version: item.version }), onSuccess: () => { setEditor(null); refresh() } })
  const archive = useMutation({ mutationFn: (item: ERPProduct) => api.erp.setArchived(item.id, item.version, item.status === 'active'), onSuccess: refresh })
  const saving = create.isPending || update.isPending
  const save = async (input: ERPProductInput) => {
    setError('')
    try {
      if (editor === 'new') await create.mutateAsync(input)
      else if (editor) await update.mutateAsync({ item: editor, input })
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to save this product')
    }
  }
  return <PageContainer variant="workspace">
    <PageHeader eyebrow="ERP / Catalog" title="Products & Services" description="Manage your workspace product catalog. This is the foundation for sales, procurement, inventory and invoicing." actions={can('erp.product.create') ? <Button onClick={() => { setError(''); setEditor('new') }}><Plus size={16} className="mr-2 inline" />Add product</Button> : undefined} />
    {!canRead ? <Card className="p-8 text-sm text-muted-foreground">You do not have permission to view the ERP catalog.</Card> : <>
      {editor && <ProductEditor key={editor === 'new' ? 'new' : editor.id} initial={editor === 'new' ? undefined : editor} pending={saving} error={error} onCancel={() => setEditor(null)} onSave={input => { void save(input) }} />}
      <Card className="mb-4 p-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative min-w-0 flex-1 sm:max-w-sm"><Search size={16} className="absolute left-3 top-3 text-muted-foreground" /><input aria-label="Search ERP products" className={inputClass + ' pl-9'} placeholder="Search name or SKU" value={search} onChange={event => { setSearch(event.target.value); setPage(1) }} /></div>
          <label className="flex items-center gap-2 text-sm text-muted-foreground"><input type="checkbox" checked={includeArchived} onChange={event => { setIncludeArchived(event.target.checked); setPage(1) }} />Include archived</label>
          <span className="ml-auto text-sm text-muted-foreground">{products.data?.total ?? 0} records</span>
        </div>
      </Card>
      <Card className="overflow-hidden">
        {products.isLoading && <div className="p-8 text-center text-sm text-muted-foreground">Loading catalog…</div>}
        {products.isError && <div role="alert" className="p-6 text-sm text-destructive">Unable to load the catalog. <button className="underline" onClick={() => void products.refetch()}>Retry</button></div>}
        {products.data?.items.length === 0 && <div className="p-10 text-center"><Package className="mx-auto mb-3 text-muted-foreground" size={24} /><p className="font-semibold">No products found</p><p className="mt-1 text-sm text-muted-foreground">Create your first product or adjust the search.</p></div>}
        {Boolean(products.data?.items.length) && <div className="overflow-x-auto"><table className="w-full min-w-[650px] text-left text-sm"><thead className="border-b border-border bg-muted/40 text-xs uppercase tracking-wide text-muted-foreground"><tr><th className="px-5 py-3">Product</th><th className="px-5 py-3">SKU</th><th className="px-5 py-3">Type</th><th className="px-5 py-3">Price</th><th className="px-5 py-3">Status</th><th className="px-5 py-3 text-right">Actions</th></tr></thead><tbody className="divide-y divide-border">{products.data?.items.map(item => <tr key={item.id} className="hover:bg-muted/30"><td className="px-5 py-4 font-medium">{item.name}</td><td className="px-5 py-4 font-mono text-xs">{item.sku}</td><td className="px-5 py-4 capitalize">{item.kind === 'good' ? 'Good' : 'Service'}</td><td className="px-5 py-4 tabular-nums">{item.unit_price} {item.currency}</td><td className="px-5 py-4 capitalize text-muted-foreground">{item.status}</td><td className="px-5 py-4"><div className="flex justify-end gap-2">{can('erp.product.update') && <Button variant="outline" onClick={() => { setError(''); setEditor(item) }}><Pencil size={14} className="mr-1 inline" />Edit</Button>}{can('erp.product.archive') && <Button variant="outline" disabled={archive.isPending} onClick={() => { setError(''); archive.mutate(item, { onError: cause => setError(cause instanceof Error ? cause.message : 'Unable to change status') }) }}>{item.status === 'active' ? <Archive size={14} className="mr-1 inline" /> : <ArchiveRestore size={14} className="mr-1 inline" />}{item.status === 'active' ? 'Archive' : 'Restore'}</Button>}</div></td></tr>)}</tbody></table></div>}
        {products.data && products.data.total > products.data.limit && <div className="flex justify-end gap-2 border-t border-border p-4"><Button variant="outline" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>Previous</Button><span className="px-2 py-2 text-xs text-muted-foreground">Page {page}</span><Button variant="outline" disabled={page * products.data.limit >= products.data.total} onClick={() => setPage(p => p + 1)}>Next</Button></div>}
      </Card>
      {error && !editor && <p role="alert" className="mt-3 text-sm text-destructive">{error}</p>}
    </>}
  </PageContainer>
}

export const erpModule: AppModule = {
  name: 'erp', version: '1.0.0', dependencies: ['core'],
  application: { id: 'erp', entryRoute: '/erp', navigationID: 'erp', apiContractVersion: 'v1' },
  routes: [
    { id: 'erp-entry', path: '/erp', element: <Navigate to="/erp/products" replace /> },
    { id: 'erp-products', path: '/erp/products', element: <ERPProductsPage /> },
  ],
  navigation: [
    { id: 'erp', label: 'ERP', path: '/erp', order: 400, icon: Boxes, isGroup: true, permission: { scope: 'workspace', name: 'erp.product.read' } },
    { id: 'erp-products', label: 'Products & Services', path: '/erp/products', order: 401, parentId: 'erp', icon: Package, permission: { scope: 'workspace', name: 'erp.product.read' } },
  ],
}
