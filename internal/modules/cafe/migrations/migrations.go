package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

func All() []module.Migration {
	return []module.Migration{{
		Module: "cafe", Version: 1, Name: "cafe_orders_and_photo_booth_bookings",
		UpSQL: `
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE cafe_booths (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
 name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 100),
 active BOOLEAN NOT NULL DEFAULT TRUE,
 created_by UUID NOT NULL REFERENCES users_users(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(id, workspace_id)
);
CREATE UNIQUE INDEX cafe_booths_workspace_name_uidx ON cafe_booths(workspace_id,lower(name));

CREATE TABLE cafe_bookings (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
 booth_id UUID NOT NULL,
 package_product_id UUID NOT NULL,
 guest_name TEXT NOT NULL CHECK(length(trim(guest_name)) BETWEEN 1 AND 120),
 starts_at TIMESTAMPTZ NOT NULL,
 ends_at TIMESTAMPTZ NOT NULL,
 status TEXT NOT NULL DEFAULT 'reserved' CHECK(status IN ('reserved','checked_in','completed','cancelled')),
 price NUMERIC(20,4) NOT NULL CHECK(price >= 0),
 currency CHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 created_by UUID NOT NULL REFERENCES users_users(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 FOREIGN KEY(booth_id, workspace_id) REFERENCES cafe_booths(id,workspace_id) ON DELETE RESTRICT,
 FOREIGN KEY(package_product_id, workspace_id) REFERENCES erp_products(id,workspace_id) ON DELETE RESTRICT,
 CHECK(ends_at > starts_at),
 CHECK(ends_at <= starts_at + INTERVAL '2 hours'),
 CONSTRAINT cafe_bookings_no_overlap EXCLUDE USING gist (
   booth_id WITH =, tstzrange(starts_at,ends_at,'[)') WITH &&
 ) WHERE (status IN ('reserved','checked_in'))
);
CREATE INDEX cafe_bookings_workspace_schedule_idx ON cafe_bookings(workspace_id,starts_at DESC);

CREATE TABLE cafe_orders (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
 status TEXT NOT NULL DEFAULT 'open' CHECK(status IN ('open','completed','cancelled')),
 total NUMERIC(20,4) NOT NULL DEFAULT 0 CHECK(total >= 0),
 currency CHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 note TEXT NOT NULL DEFAULT '' CHECK(length(note) <= 300),
 created_by UUID NOT NULL REFERENCES users_users(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(id,workspace_id)
);
CREATE INDEX cafe_orders_workspace_created_idx ON cafe_orders(workspace_id,created_at DESC);

CREATE TABLE cafe_order_lines (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL,
 order_id UUID NOT NULL,
 product_id UUID NOT NULL,
 product_name TEXT NOT NULL CHECK(length(product_name) BETWEEN 1 AND 200),
 quantity INTEGER NOT NULL CHECK(quantity BETWEEN 1 AND 99),
 unit_price NUMERIC(20,4) NOT NULL CHECK(unit_price >= 0),
 line_total NUMERIC(20,4) GENERATED ALWAYS AS (quantity * unit_price) STORED,
 FOREIGN KEY(order_id, workspace_id) REFERENCES cafe_orders(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(product_id, workspace_id) REFERENCES erp_products(id,workspace_id) ON DELETE RESTRICT
);
CREATE INDEX cafe_order_lines_order_idx ON cafe_order_lines(workspace_id,order_id);
`,
		DownSQL: `DROP TABLE IF EXISTS cafe_order_lines;
DROP TABLE IF EXISTS cafe_orders;
DROP TABLE IF EXISTS cafe_bookings;
DROP TABLE IF EXISTS cafe_booths;`,
	}}
}
