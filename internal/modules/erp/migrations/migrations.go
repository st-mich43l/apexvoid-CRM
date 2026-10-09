package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

func All() []module.Migration {
	return []module.Migration{{
		Module: "erp", Version: 1, Name: "create_erp_product_catalog",
		UpSQL: `CREATE TABLE erp_products (
			id UUID PRIMARY KEY,
			workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
			sku TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL CHECK(kind IN ('good','service')),
			unit TEXT NOT NULL CHECK(unit IN ('unit','hour','kg')),
			unit_price NUMERIC(20,4) NOT NULL CHECK(unit_price >= 0),
			currency CHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
			status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
			version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
			created_by UUID NOT NULL REFERENCES users_users(id),
			updated_by UUID NOT NULL REFERENCES users_users(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(id,workspace_id),
			CHECK(length(name) BETWEEN 1 AND 200),
			CHECK(length(description) <= 2000),
			CHECK(sku ~ '^[A-Z0-9][A-Z0-9._-]{0,63}$')
		);
		CREATE UNIQUE INDEX erp_products_sku_workspace_uidx ON erp_products(workspace_id,lower(sku));
		CREATE INDEX erp_products_search_idx ON erp_products(workspace_id,status,name);`,
		DownSQL: "DROP TABLE IF EXISTS erp_products;",
	}}
}
