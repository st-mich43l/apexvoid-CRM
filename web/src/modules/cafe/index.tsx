import { useState, type FormEvent } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Camera, CheckCircle2, Coffee, Plus, ShoppingBasket, CalendarClock, XCircle } from 'lucide-react'
import { api, type ERPProduct, type CafeOrder, type CafeBooking } from '../../core/api/client'
import { useAuth } from '../../core/auth/context'
import { useWorkspace } from '../../core/workspace/context'
import { userWorkspaceQueryKey } from '../../core/workspace/query'
import type { AppModule } from '../../framework/module/types'
import { Badge, Button, Card, EmptyState, ErrorState, LoadingState, PageContainer, PageHeader, inputClass } from '../../components/ui'

const formatMoney = (raw: string, currency: string) => {
  const [whole, fraction] = raw.split('.')
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  const decimals = fraction?.replace(/0+$/, '')
  return `${grouped}${decimals ? '.' + decimals : ''} ${currency}`
}
const formatDate = (date: string) => new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(date))
function RequestError({ error }: { error: unknown }) {
  if (!error) return null
  return <p role="alert" className="rounded-lg border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">{error instanceof Error ? error.message : 'The operation failed.'}</p>
}
function Status({ value }: { value: string }) {
  return <Badge tone={value === 'completed' ? 'success' : value === 'cancelled' ? 'neutral' : value === 'checked_in' ? 'primary' : 'warning'}>{value.replaceAll('_', ' ')}</Badge>
}
function useCafeKey(area: string) {
  const { user } = useAuth()
  const { activeWorkspaceId } = useWorkspace()
  return userWorkspaceQueryKey(user?.id, activeWorkspaceId, 'cafe', area)
}
function AppNav({ current }: { current: 'counter' | 'booths' }) {
  const { can } = useWorkspace()
  return <nav aria-label="Café features" className="mb-6 flex flex-wrap gap-2">
    {can('cafe.order.read') && <Link to="/cafe/counter" aria-current={current === 'counter' ? 'page' : undefined} className={`rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${current === 'counter' ? 'border-primary bg-primary/10 text-primary' : 'border-border text-muted-foreground hover:bg-muted'}`}><Coffee size={15} className="mr-2 inline" />Café counter</Link>}
    {can('cafe.booking.read') && <Link to="/cafe/booths" aria-current={current === 'booths' ? 'page' : undefined} className={`rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${current === 'booths' ? 'border-primary bg-primary/10 text-primary' : 'border-border text-muted-foreground hover:bg-muted'}`}><Camera size={15} className="mr-2 inline" />Photo booth</Link>}
  </nav>
}

function CounterPage() {
  const { activeWorkspaceId, can } = useWorkspace()
  const queryClient = useQueryClient()
  const ordersKey = useCafeKey('orders')
  const catalogKey = useCafeKey('catalog')
  const orders = useQuery({ queryKey: ordersKey, queryFn: api.cafe.orders, enabled: Boolean(activeWorkspaceId && can('cafe.order.read')) })
  const catalog = useQuery({ queryKey: catalogKey, queryFn: () => api.erp.products({ limit: 100 }), enabled: Boolean(activeWorkspaceId && can('erp.product.read')) })
  const goods = (catalog.data?.items ?? []).filter(product => product.kind === 'good' && product.status === 'active')
  const [cart, setCart] = useState<Record<string, number>>({})
  const [note, setNote] = useState('')
  const [search, setSearch] = useState('')
  const [message, setMessage] = useState('')
  const selected = goods.filter(product => cart[product.id] > 0)
  const firstCurrency = selected[0]?.currency
  const checkout = useMutation({
    mutationFn: api.cafe.createOrder,
    onSuccess: () => {
      setCart({}); setNote(''); setMessage('Order created — confirm completion after serving.')
      void queryClient.invalidateQueries({ queryKey: ordersKey })
    },
  })
  const changeStatus = useMutation({
    mutationFn: ({ order, action }: { order: CafeOrder; action: 'complete' | 'cancel' }) => api.cafe.orderAction(order.id, action),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ordersKey }),
  })
  const add = (product: ERPProduct) => {
    setMessage('')
    setCart(current => ({ ...current, [product.id]: Math.min(99, (current[product.id] ?? 0) + 1) }))
  }
  const changeQty = (id: string, delta: number) => setCart(current => {
    const next = { ...current }
    const quantity = Math.max(0, Math.min(99, (next[id] ?? 0) + delta))
    if (!quantity) delete next[id]
    else next[id] = quantity
    return next
  })
  const submit = (event: FormEvent) => {
    event.preventDefault()
    if (!selected.length || selected.some(item => item.currency !== firstCurrency)) return
    checkout.mutate({ note, lines: selected.map(product => ({ product_id: product.id, quantity: cart[product.id] })) })
  }
  return <PageContainer variant="workspace">
    <PageHeader eyebrow="ApexVoid Café / Point of sale" title="Café Counter" description="Take orders using the shared ERP product catalog. This is an operational order register; payments and fiscal receipts are not yet integrated." />
    <AppNav current="counter" />
    {!can('cafe.order.read') ? <EmptyState title="Counter unavailable" description="You need café order permissions for this workspace." /> :
      <div className="grid gap-5 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.85fr)]">
        <div className="space-y-5">
          <Card className="p-5">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
              <div><h2 className="text-lg font-semibold">Menu</h2><p className="text-xs text-muted-foreground">Products managed in ERP → Products & Services</p></div>
              <input aria-label="Search menu" value={search} onChange={event => setSearch(event.target.value)} placeholder="Search drinks or SKU" className={`${inputClass} max-w-xs`} />
            </div>
            {!can('erp.product.read') && <p className="text-sm text-muted-foreground">ERP product read permission is required to display the menu.</p>}
            {catalog.isLoading && <LoadingState label="Loading menu…" />}
            {catalog.isError && <ErrorState message="Unable to load ERP products." onRetry={() => void catalog.refetch()} />}
            {catalog.isSuccess && !goods.length && <EmptyState title="No menu products" description="Create drinks as Goods in ERP Products & Services to populate the menu." />}
            <div className="grid gap-3 sm:grid-cols-2">
              {goods.filter(p => `${p.name} ${p.sku}`.toLowerCase().includes(search.toLowerCase())).map(product =>
                <button type="button" key={product.id} disabled={!can('cafe.order.create') || (Boolean(firstCurrency) && product.currency !== firstCurrency)} onClick={() => add(product)} className="group rounded-xl border border-border bg-muted/20 p-4 text-left transition-colors hover:border-primary/50 hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-50">
                  <div className="flex items-center justify-between gap-2"><Coffee size={19} className="text-primary" /><Plus size={16} className="text-muted-foreground group-hover:text-primary" /></div>
                  <p className="mt-3 truncate text-sm font-semibold">{product.name}</p>
                  <p className="mt-1 font-mono text-[11px] text-muted-foreground">{product.sku}</p>
                  <p className="mt-2 text-sm font-semibold tabular-nums">{formatMoney(product.unit_price, product.currency)}</p>
                </button>)}
            </div>
          </Card>
          <Card className="p-5">
            <h2 className="mb-4 text-lg font-semibold">Recent orders</h2>
            {orders.isLoading && <LoadingState label="Loading orders…" />}
            {orders.isError && <ErrorState message="Unable to load orders." onRetry={() => void orders.refetch()} />}
            {orders.isSuccess && orders.data.length === 0 && <EmptyState title="No orders yet" description="Create the first café sale from the menu." />}
            <div className="space-y-3">{orders.data?.map(order => <div key={order.id} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border p-3">
              <div><p className="text-sm font-medium">Order #{order.id.slice(0, 8)}</p><p className="text-xs text-muted-foreground">{formatDate(order.created_at)} · {order.line_count} items</p><div className="mt-1"><Status value={order.status} /></div></div>
              <div className="flex flex-wrap items-center gap-2"><span className="mr-2 text-sm font-semibold tabular-nums">{formatMoney(order.total, order.currency)}</span>{order.status === 'open' && can('cafe.order.manage') && <><Button type="button" variant="outline" disabled={changeStatus.isPending} onClick={() => changeStatus.mutate({ order, action: 'complete' })}>Served</Button><Button type="button" variant="ghost" disabled={changeStatus.isPending} onClick={() => changeStatus.mutate({ order, action: 'cancel' })}>Cancel</Button></>}</div>
            </div>)}</div>
            <RequestError error={changeStatus.error} />
          </Card>
        </div>
        <Card className="h-fit p-5 xl:sticky xl:top-4">
          <div className="mb-4 flex items-center gap-2"><ShoppingBasket size={19} className="text-primary" /><h2 className="text-lg font-semibold">Current order</h2></div>
          {!selected.length && <p className="rounded-lg bg-muted/40 p-6 text-center text-sm text-muted-foreground">Select menu items to build a new order.</p>}
          <div className="space-y-3">{selected.map(product => <div key={product.id} className="flex items-center justify-between gap-3 border-b border-border pb-3">
            <div className="min-w-0"><p className="truncate text-sm font-medium">{product.name}</p><p className="text-xs text-muted-foreground">{formatMoney(product.unit_price, product.currency)} each</p></div>
            <div className="flex shrink-0 items-center gap-2"><button type="button" aria-label={`Remove one ${product.name}`} className="rounded-md border border-border px-2 py-1" onClick={() => changeQty(product.id, -1)}>−</button><span className="w-4 text-center text-sm tabular-nums">{cart[product.id]}</span><button type="button" aria-label={`Add one ${product.name}`} className="rounded-md border border-border px-2 py-1" onClick={() => changeQty(product.id, 1)}>+</button></div>
          </div>)}</div>
          {selected.some(product => product.currency !== firstCurrency) && <p role="alert" className="mt-3 text-sm text-destructive">One order must use a single currency.</p>}
          <form className="mt-4 space-y-3" onSubmit={submit}>
            <label className="block text-xs font-semibold text-muted-foreground">Order note<textarea maxLength={300} rows={2} className={inputClass} value={note} onChange={event => setNote(event.target.value)} placeholder="Takeaway, table, customer instructions…" /></label>
            <Button type="submit" className="w-full" disabled={!selected.length || !can('cafe.order.create') || checkout.isPending || selected.some(p => p.currency !== firstCurrency)}>{checkout.isPending ? 'Creating…' : 'Create order'}</Button>
            {message && <p role="status" className="text-sm text-success">{message}</p>}
            <RequestError error={checkout.error} />
          </form>
        </Card>
      </div>}
  </PageContainer>
}

function BoothPage() {
  const { activeWorkspaceId, can } = useWorkspace()
  const client = useQueryClient()
  const boothsKey = useCafeKey('booths')
  const bookingsKey = useCafeKey('bookings')
  const catalogKey = useCafeKey('photo-packages')
  const booths = useQuery({ queryKey: boothsKey, queryFn: api.cafe.booths, enabled: Boolean(activeWorkspaceId && can('cafe.booth.read')) })
  const bookings = useQuery({ queryKey: bookingsKey, queryFn: api.cafe.bookings, enabled: Boolean(activeWorkspaceId && can('cafe.booking.read')) })
  const catalog = useQuery({ queryKey: catalogKey, queryFn: () => api.erp.products({ limit: 100 }), enabled: Boolean(activeWorkspaceId && can('erp.product.read')) })
  const packages = (catalog.data?.items ?? []).filter(product => product.kind === 'service' && product.status === 'active')
  const [boothName, setBoothName] = useState('')
  const [boothID, setBoothID] = useState('')
  const [productID, setProductID] = useState('')
  const [guestName, setGuestName] = useState('')
  const [localStart, setLocalStart] = useState('')
  const [duration, setDuration] = useState(20)
  const [message, setMessage] = useState('')
  const createBooth = useMutation({ mutationFn: api.cafe.addBooth, onSuccess: () => { setBoothName(''); void client.invalidateQueries({ queryKey: boothsKey }) } })
  const reserve = useMutation({
    mutationFn: api.cafe.reserve,
    onSuccess: () => { setGuestName(''); setMessage('Photo-booth slot reserved.'); void client.invalidateQueries({ queryKey: bookingsKey }) },
  })
  const transition = useMutation({ mutationFn: ({ booking, action }: { booking: CafeBooking; action: 'check-in' | 'complete' | 'cancel' }) => api.cafe.bookingAction(booking.id, action), onSuccess: () => void client.invalidateQueries({ queryKey: bookingsKey }) })
  const book = (event: FormEvent) => {
    event.preventDefault()
    setMessage('')
    const start = new Date(localStart)
    if (!Number.isFinite(start.getTime())) return
    reserve.mutate({ booth_id: boothID, package_product_id: productID, guest_name: guestName, starts_at: start.toISOString(), ends_at: new Date(start.getTime() + duration * 60_000).toISOString() })
  }
  return <PageContainer variant="workspace">
    <PageHeader eyebrow="ApexVoid Café / Experiences" title="Photo Booth" description="Manage shooting stations and scheduled guest sessions. A database constraint prevents overlapping reservations for the same booth." />
    <AppNav current="booths" />
    {!can('cafe.booking.read') ? <EmptyState title="Photo booth unavailable" description="You need photo booking permissions for this workspace." /> :
      <div className="grid gap-5 xl:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.75fr)]">
        <div className="space-y-5">
          <Card className="p-5"><div className="mb-4 flex items-center gap-2"><Camera size={19} className="text-primary" /><h2 className="text-lg font-semibold">Shooting stations</h2></div>
            {!can('cafe.booth.read') && <p className="text-sm text-muted-foreground">Booth read permission is needed to select a station.</p>}
            {booths.isError && <ErrorState message="Unable to load photo booths." onRetry={() => void booths.refetch()} />}
            <div className="grid gap-3 sm:grid-cols-2">{booths.data?.map(item => <div key={item.id} className="rounded-xl border border-border bg-muted/20 p-4"><Camera size={19} className="mb-2 text-primary" /><p className="text-sm font-semibold">{item.name}</p><p className="mt-1 text-xs text-muted-foreground">{item.active ? 'Available to book' : 'Inactive'}</p></div>)}</div>
            {booths.isSuccess && !booths.data.length && <EmptyState title="No photo booths configured" description="Add a shooting station to begin accepting reservations." />}
            {can('cafe.booth.manage') && <form className="mt-4 flex flex-wrap items-end gap-2" onSubmit={event => { event.preventDefault();createBooth.mutate(boothName) }}><label className="min-w-0 flex-1 text-xs font-semibold text-muted-foreground">New booth name<input className={inputClass} required maxLength={100} value={boothName} onChange={event => setBoothName(event.target.value)} placeholder="Booth A" /></label><Button type="submit" disabled={createBooth.isPending}><Plus size={15} className="mr-1 inline" />Add booth</Button></form>}
            <RequestError error={createBooth.error} />
          </Card>
          <Card className="p-5">
            <div className="mb-4 flex items-center gap-2"><CalendarClock size={19} className="text-primary" /><h2 className="text-lg font-semibold">Recent and upcoming bookings</h2></div>
            {bookings.isLoading && <LoadingState label="Loading bookings…" />}
            {bookings.isError && <ErrorState message="Unable to load bookings." onRetry={() => void bookings.refetch()} />}
            {bookings.isSuccess && !bookings.data.length && <EmptyState title="No sessions yet" description="Reservations will appear here after the first booking." />}
            <div className="space-y-3">{bookings.data?.map(booking => <div key={booking.id} className="flex flex-wrap justify-between gap-3 rounded-xl border border-border p-4">
              <div><p className="font-semibold">{booking.guest_name}</p><p className="mt-1 text-sm text-muted-foreground">{booking.booth_name} · {formatDate(booking.starts_at)}</p><p className="mt-1 text-xs text-muted-foreground">{formatMoney(booking.price, booking.currency)}</p><div className="mt-2"><Status value={booking.status} /></div></div>
              <div className="flex items-center gap-2">{can('cafe.booking.manage') && booking.status === 'reserved' && <><Button type="button" variant="outline" disabled={transition.isPending} onClick={() => transition.mutate({ booking, action: 'check-in' })}>Check in</Button><Button type="button" variant="ghost" disabled={transition.isPending} onClick={() => transition.mutate({ booking, action: 'cancel' })}><XCircle size={15} className="mr-1 inline" />Cancel</Button></>}{can('cafe.booking.manage') && booking.status === 'checked_in' && <Button type="button" variant="outline" disabled={transition.isPending} onClick={() => transition.mutate({ booking, action: 'complete' })}><CheckCircle2 size={15} className="mr-1 inline" />Complete</Button>}</div>
            </div>)}</div><RequestError error={transition.error} />
          </Card>
        </div>
        {can('cafe.booking.create') && <Card className="h-fit p-5 xl:sticky xl:top-4">
          <h2 className="mb-1 text-lg font-semibold">Reserve photo session</h2><p className="mb-5 text-xs text-muted-foreground">Select a physical booth, an ERP service package, and the start time.</p>
          <form className="space-y-4" onSubmit={book}>
            <label className="block text-sm font-medium">Booth<select className={inputClass} required value={boothID} onChange={event => setBoothID(event.target.value)}><option value="">Choose a booth…</option>{booths.data?.filter(item => item.active).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
            <label className="block text-sm font-medium">Photo package<select className={inputClass} required value={productID} onChange={event => setProductID(event.target.value)}><option value="">Choose a service…</option>{packages.map(product => <option key={product.id} value={product.id}>{product.name} — {formatMoney(product.unit_price, product.currency)}</option>)}</select></label>
            <label className="block text-sm font-medium">Guest name<input className={inputClass} required maxLength={120} value={guestName} onChange={event => setGuestName(event.target.value)} placeholder="Guest or group name" /></label>
            <label className="block text-sm font-medium">Start time<input className={inputClass} required type="datetime-local" value={localStart} onChange={event => setLocalStart(event.target.value)} /></label>
            <label className="block text-sm font-medium">Session length<select className={inputClass} value={duration} onChange={event => setDuration(Number(event.target.value))}><option value={15}>15 minutes</option><option value={20}>20 minutes</option><option value={30}>30 minutes</option><option value={45}>45 minutes</option><option value={60}>60 minutes</option></select></label>
            {!can('erp.product.read') && <p className="text-sm text-destructive">ERP product read permission is required to select packages.</p>}
            <Button type="submit" className="w-full" disabled={reserve.isPending || !boothID || !productID}>{reserve.isPending ? 'Booking…' : 'Reserve booth'}</Button>
            <RequestError error={reserve.error} />{message && <p role="status" className="text-sm text-success">{message}</p>}
          </form>
        </Card>}
      </div>}
  </PageContainer>
}

function CafeEntry() {
  const { can } = useWorkspace()
  if (can('cafe.order.read')) return <Navigate to="/cafe/counter" replace />
  if (can('cafe.booking.read')) return <Navigate to="/cafe/booths" replace />
  return <EmptyState title="Café access unavailable" description="Ask an administrator to grant the café role permissions." />
}

// eslint-disable-next-line react-refresh/only-export-components
export const cafeModule: AppModule = {
  name: 'cafe', version: '1.0.0', dependencies: ['core', 'erp'],
  application: { id: 'cafe', entryRoute: '/cafe', navigationID: 'cafe', apiContractVersion: 'v1' },
  routes: [
    { id: 'cafe-entry', path: '/cafe', element: <CafeEntry /> },
    { id: 'cafe-counter', path: '/cafe/counter', element: <CounterPage /> },
    { id: 'cafe-booths', path: '/cafe/booths', element: <BoothPage /> },
  ],
  navigation: [
    { id: 'cafe', label: 'Café & Photo Booth', path: '/cafe', order: 450, icon: Coffee, isGroup: true, permissionsAny: [{ scope: 'workspace', name: 'cafe.order.read' }, { scope: 'workspace', name: 'cafe.booking.read' }] },
    { id: 'cafe-counter', label: 'Café Counter', path: '/cafe/counter', order: 451, icon: ShoppingBasket, parentId: 'cafe', permission: { scope: 'workspace', name: 'cafe.order.read' } },
    { id: 'cafe-booths', label: 'Photo Booth', path: '/cafe/booths', order: 452, icon: Camera, parentId: 'cafe', permission: { scope: 'workspace', name: 'cafe.booking.read' } },
  ],
}
