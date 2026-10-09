# ApexVoid Café & Photo Booth

First full business application inside **ApexVoid Enterprise**. The Go backend, React/Vite
frontend, PostgreSQL, authentication, workspace/RBAC authority, and Docker deployment
are all shared with the enterprise platform. No new service or repository is needed.

## Business model

The physical venue sells café items and separately books private self-photo booths.
The goal is to measure the two revenue streams and make visits more valuable with
coffee/photo bundles in a future release.

Customer journey:
1. Order a drink at the counter.
2. Choose an optional photo package and available booth.
3. Staff reserve the booth; guests check in and complete their session.
4. Staff mark fulfilled drink orders as completed. Completed order values are **not
   verified payment receipts**; payment integration is a future feature.
5. Print upsells, loyalty, packages, and event bookings can follow.

Reference examples:
- Photoby Studio & Café: https://www.elle.vn/trai-nghiem/dia-diem-chup-photobooth-noi-bat-nhat-sai-gon/
- Observed photo booth pricing in Vietnam: https://photobooth.joelin.cc/vi/vietnam/prices/
- Restaurant POS item customization/fulfillment: https://square.site/help/us/en/article/5696-customize-modifiers-for-order-entry

No prices are hardcoded; staff configure them through **ERP Products & Services**.
Create beverages as **Goods** and photo packages as **Services** using the same
currency (e.g., VND). Product classification/catalog filtering by venue/category
should be added before using a large shared ERP catalog: the MVP counter shows all
active goods and the booking screen shows all active services, with a 100-item
client-side cap.

## Workflows and API

Application ID `cafe`, frontend `/cafe`, backend routes under `/api/v1/cafe`.

- **Counter** (`/cafe/counter`): select ERP goods, create transactional
  multi-line orders, track open/completed/cancelled orders.
  Each line snapshots its ERP price and name; exact totals are computed using
  PostgreSQL `NUMERIC` rather than floating-point money.
- **Booths** (`/cafe/booths`): create named stations, reserve times using
  ERP service packages, check in, complete, or cancel.
- Cancellation releases a reserved time slot. The PostgreSQL partial GiST
  exclusion constraint makes overlapping active bookings impossible even
  under concurrent requests. Adjacent half-open ranges are allowed.
- REST APIs are workspace scoped, session authenticated, and require
  explicit `cafe.order.*`, `cafe.booth.*`, and `cafe.booking.*`
  workspace permissions. For menus, staff additionally require
  `erp.product.read`.

All routes use the current workspace from trusted middleware. The request does
not accept a workspace ID. ERP prices and products are accessed through a
narrow Go interface instead of direct cross-module application queries.

## Important MVP boundaries

This release is a **staff-operated prototype**, not a finished retail POS.
It does **not** charge customers, settle payments, issue tax-compliant receipts,
manage drink modifiers (size/ice/sugar), route tickets to baristas, manage
stock, use a cash drawer, allocate tables, or verify actual payment. Do not
use the completed order total as booked revenue until payment is integrated.

There is no public booking portal, calendar availability selector, buffer
times for cleaning, staffing shifts, late/no-show policies, loyalty, bundled
pricing, or device integration. Bookings are listed latest-100; long-term
analytics require pagination and reporting. Station disabling is not exposed
in MVP. Photos/biometric content are **not collected or stored**; the only
booking personal information is the guest name. Before public launch, design
guest-notice, deletion/retention and access policies under applicable law
(in Vietnam, Law No. 91/2025/QH15 takes effect January 1, 2026).

Deployment needs a PostgreSQL instance allowing the trusted `btree_gist`
extension (migration installs it if not already present). Verify migrations
and app discovery with `make test-integration`.

## Next implementation milestones

1. Café-specific catalog categories, item options/modifiers and menu availability.
2. Slot finder with opening hours, booth turnaround buffers, booking timezone.
3. Integrated payment, refunds and order-to-booking combo packages.
4. Inventory consumption and supplier procurement from ERP.
5. Sales analytics, staff dashboards, guest privacy and retention controls.
