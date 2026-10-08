package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

func All() []module.Migration {
	return []module.Migration{{Module: "customization", Version: 1, Name: "create_workspace_configuration", UpSQL: `
CREATE TABLE IF NOT EXISTS customization_form_sections (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
 entity_name TEXT NOT NULL,
 name TEXT NOT NULL,
 description TEXT NOT NULL DEFAULT '',
 display_order INTEGER NOT NULL DEFAULT 0,
 created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL,
 UNIQUE(workspace_id, entity_name, name),
 UNIQUE(id, workspace_id)
);
CREATE TABLE IF NOT EXISTS customization_field_definitions (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
 entity_name TEXT NOT NULL,
 field_key TEXT NOT NULL,
 label TEXT NOT NULL,
 field_type TEXT NOT NULL CHECK(field_type IN ('string','text','decimal','integer','boolean','date','enum')),
 description TEXT NOT NULL DEFAULT '',
 required BOOLEAN NOT NULL DEFAULT FALSE,
 default_value JSONB NOT NULL DEFAULT 'null'::jsonb,
 options JSONB NOT NULL DEFAULT '[]'::jsonb,
 visible BOOLEAN NOT NULL DEFAULT TRUE,
 display_order INTEGER NOT NULL DEFAULT 0,
 section_id UUID,
 active BOOLEAN NOT NULL DEFAULT TRUE,
 created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL,
 UNIQUE(workspace_id, entity_name, field_key),
 UNIQUE(id, workspace_id),
 FOREIGN KEY(section_id, workspace_id) REFERENCES customization_form_sections(id, workspace_id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS customization_fields_workspace_entity_idx ON customization_field_definitions(workspace_id, entity_name, active, display_order);
CREATE TABLE IF NOT EXISTS customization_saved_views (
 id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
 entity_name TEXT NOT NULL,
 owner_user_id UUID NOT NULL REFERENCES users_users(id) ON DELETE CASCADE,
 name TEXT NOT NULL,
 shared BOOLEAN NOT NULL DEFAULT FALSE,
 filters JSONB NOT NULL DEFAULT '[]'::jsonb,
 columns JSONB NOT NULL DEFAULT '[]'::jsonb,
 sort_field TEXT NOT NULL DEFAULT '',
 sort_direction TEXT NOT NULL DEFAULT 'asc' CHECK(sort_direction IN ('asc','desc')),
 created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL,
 UNIQUE(workspace_id, entity_name, owner_user_id, name),
 UNIQUE(id, workspace_id)
);
CREATE INDEX IF NOT EXISTS customization_views_workspace_entity_idx ON customization_saved_views(workspace_id, entity_name, shared, owner_user_id);`, DownSQL: `
DROP TABLE IF EXISTS customization_saved_views;
DROP TABLE IF EXISTS customization_field_definitions;
DROP TABLE IF EXISTS customization_form_sections;`}}
}
