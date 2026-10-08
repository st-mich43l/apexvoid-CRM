package migrations

import "github.com/st-mich43l/apexvoid-CRM/internal/framework/module"

func All() []module.Migration {
	return []module.Migration{{Module: "contacts", Version: 1, Name: "create_contacts_business_foundation", UpSQL: `
CREATE TABLE IF NOT EXISTS contacts_contacts (
 id UUID PRIMARY KEY, workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE,
 kind TEXT NOT NULL CHECK (kind IN ('person','company')), display_name TEXT NOT NULL, email TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '', website TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT '', status TEXT NOT NULL CHECK (status IN ('active','archived')), custom_values JSONB NOT NULL DEFAULT '{}'::jsonb,
 created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, created_by UUID NOT NULL REFERENCES users_users(id), updated_by UUID NOT NULL REFERENCES users_users(id), UNIQUE(id, workspace_id)
);
CREATE INDEX IF NOT EXISTS contacts_contacts_workspace_idx ON contacts_contacts(workspace_id, status, kind, display_name);
CREATE INDEX IF NOT EXISTS contacts_contacts_search_idx ON contacts_contacts USING gin (to_tsvector('simple', display_name || ' ' || email || ' ' || phone));
CREATE TABLE IF NOT EXISTS contacts_relationships (
 id UUID PRIMARY KEY, workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE, person_id UUID NOT NULL, company_id UUID NOT NULL, relationship_type TEXT NOT NULL, job_title TEXT NOT NULL DEFAULT '', is_primary BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
 FOREIGN KEY (person_id, workspace_id) REFERENCES contacts_contacts(id, workspace_id) ON DELETE CASCADE, FOREIGN KEY (company_id, workspace_id) REFERENCES contacts_contacts(id, workspace_id) ON DELETE CASCADE, UNIQUE(workspace_id, person_id, company_id, relationship_type), CHECK(person_id <> company_id)
);
CREATE INDEX IF NOT EXISTS contacts_relationships_company_idx ON contacts_relationships(workspace_id, company_id);
CREATE TABLE IF NOT EXISTS contacts_tags (
 id UUID PRIMARY KEY, workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE, name TEXT NOT NULL, color TEXT NOT NULL, active BOOLEAN NOT NULL DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, UNIQUE(workspace_id, name), UNIQUE(id, workspace_id)
);
CREATE TABLE IF NOT EXISTS contacts_contact_tags (
 workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE, contact_id UUID NOT NULL, tag_id UUID NOT NULL, PRIMARY KEY(workspace_id, contact_id, tag_id), FOREIGN KEY(contact_id, workspace_id) REFERENCES contacts_contacts(id, workspace_id) ON DELETE CASCADE, FOREIGN KEY(tag_id, workspace_id) REFERENCES contacts_tags(id, workspace_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS contacts_notes (
 id UUID PRIMARY KEY, workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE, contact_id UUID NOT NULL, author_user_id UUID NOT NULL REFERENCES users_users(id), content TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, FOREIGN KEY(contact_id, workspace_id) REFERENCES contacts_contacts(id, workspace_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS contacts_activities (
 id UUID PRIMARY KEY, workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE, title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', activity_type TEXT NOT NULL CHECK(activity_type IN ('call','email','meeting','follow_up','task')), related_contact_id UUID, assigned_user_id UUID NOT NULL REFERENCES users_users(id), due_at TIMESTAMPTZ, status TEXT NOT NULL CHECK(status IN ('planned','completed','cancelled')), completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, FOREIGN KEY(related_contact_id, workspace_id) REFERENCES contacts_contacts(id, workspace_id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS contacts_activities_due_idx ON contacts_activities(workspace_id, status, due_at);
CREATE TABLE IF NOT EXISTS contacts_attachments (
 id UUID PRIMARY KEY, workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE, contact_id UUID NOT NULL, file_name TEXT NOT NULL, storage_key TEXT NOT NULL UNIQUE, size BIGINT NOT NULL CHECK(size >= 0), content_type TEXT NOT NULL, uploaded_by UUID NOT NULL REFERENCES users_users(id), created_at TIMESTAMPTZ NOT NULL, FOREIGN KEY(contact_id, workspace_id) REFERENCES contacts_contacts(id, workspace_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS contacts_custom_field_definitions (
 id UUID PRIMARY KEY, workspace_id UUID NOT NULL REFERENCES workspace_workspaces(id) ON DELETE CASCADE, entity TEXT NOT NULL CHECK(entity = 'contacts.contact'), field_key TEXT NOT NULL, label TEXT NOT NULL, field_type TEXT NOT NULL CHECK(field_type IN ('text','number','boolean','date','selection')), description TEXT NOT NULL DEFAULT '', required BOOLEAN NOT NULL DEFAULT FALSE, options JSONB NOT NULL DEFAULT '[]'::jsonb, display_order INTEGER NOT NULL DEFAULT 0, active BOOLEAN NOT NULL DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, UNIQUE(workspace_id, entity, field_key)
);
`, DownSQL: `DROP TABLE IF EXISTS contacts_custom_field_definitions; DROP TABLE IF EXISTS contacts_attachments; DROP TABLE IF EXISTS contacts_activities; DROP TABLE IF EXISTS contacts_notes; DROP TABLE IF EXISTS contacts_contact_tags; DROP TABLE IF EXISTS contacts_tags; DROP TABLE IF EXISTS contacts_relationships; DROP TABLE IF EXISTS contacts_contacts;`}}
}
