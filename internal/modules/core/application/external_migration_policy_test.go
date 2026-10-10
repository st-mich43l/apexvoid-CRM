package application

import "testing"

func TestMigrationPolicyPermitsOwnedTableDataBackfills(t *testing.T) {
	allowed := []string{
		"UPDATE photobooth.photobooth_bookings SET status='confirmed' WHERE status='reserved';",
		"UPDATE photobooth.photobooth_bookings\nSET booking_ref='PBT-123' WHERE booking_ref IS NULL;",
		"ALTER TABLE photobooth.photobooth_bookings ALTER COLUMN status SET DEFAULT 'confirmed';",
		"CREATE OR REPLACE FUNCTION photobooth.touch() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END; $$;",
		"INSERT INTO photobooth.photobooth_booking_events(workspace_id, booking_id) SELECT workspace_id, id FROM photobooth.photobooth_bookings;",
	}
	for _, statement := range allowed {
		if err := validateMigrationSQL([]byte(statement)); err != nil {
			t.Errorf("rejected ordinary migration SQL: %s (%v)", statement, err)
		}
	}
}

func TestMigrationPolicyStillRejectsPrivilegeOrSessionChanges(t *testing.T) {
	for _, statement := range []string{
		"SET ROLE postgres;",
		"SET search_path TO public;",
		"UPDATE photobooth.photobooth_bookings SET status='confirmed'; SET ROLE postgres;",
		"GRANT ALL ON photobooth.photobooth_bookings TO public;",
		"CREATE EXTENSION btree_gist;",
		"ALTER TABLE photobooth.photobooth_bookings OWNER TO postgres;",
		"SELECT pg_read_file('/etc/passwd');",
		`UPDATE "core_external_applications" SET description='cross application';`,
		"CREATE OR REPLACE FUNCTION photobooth.bad() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER AS $$ BEGIN RETURN NEW; END; $$;",
		"DO $$ BEGIN EXECUTE 'GRANT ALL ON photobooth.photobooth_bookings TO public'; END; $$;",
		"CREATE OR REPLACE FUNCTION photobooth.dynamic() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN EXECUTE 'UPDATE photobooth.photobooth_bookings SET status = ''confirmed'''; RETURN NEW; END; $$;",
		"CREATE OR REPLACE FUNCTION photobooth.file_read() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_read_file('/etc/passwd'); RETURN NEW; END; $$;",
		"DROP TABLE photobooth.photobooth_bookings;",
		"CREATE EXTENSION IF NOT EXISTS btree_gist;",
		"SELECT 1;",
	} {
		if err := validateMigrationSQL([]byte(statement)); err == nil {
			t.Errorf("allowed restricted migration SQL: %s", statement)
		}
	}
	for _, statement := range []string{
		"-- GRANT ALL ON photobooth.photobooth_bookings TO public\nUPDATE photobooth.photobooth_bookings SET notes='security definer and pg_read_file are ordinary text';",
		"INSERT INTO photobooth.photobooth_booking_events(reason) VALUES ('COPY photobooth.photobooth_bookings TO PROGRAM');",
		"/* outer comment /* nested GRANT ALL */ still comment */ UPDATE photobooth.photobooth_bookings SET notes=E'escaped \\' quote';",
	} {
		if err := validateMigrationSQL([]byte(statement)); err != nil {
			t.Errorf("rejected safe comment or string content: %s (%v)", statement, err)
		}
	}
}

func TestMigrationPolicyAcceptsPhotoboothPhaseTwoSafetyShape(t *testing.T) {
	const migration = `ALTER TABLE photobooth.photobooth_bookings ADD COLUMN IF NOT EXISTS notes VARCHAR(1000) NOT NULL DEFAULT '';
ALTER TABLE photobooth.photobooth_bookings ALTER COLUMN booking_ref SET DEFAULT ('PBT-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 18)));
UPDATE photobooth.photobooth_bookings SET status='confirmed' WHERE status='reserved';
CREATE UNIQUE INDEX IF NOT EXISTS photobooth_bookings_ref_unique ON photobooth.photobooth_bookings(workspace_id, booking_ref);
CREATE OR REPLACE FUNCTION photobooth.photobooth_booking_no_overlap() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END; $$;
DROP TRIGGER IF EXISTS photobooth_booking_no_overlap ON photobooth.photobooth_bookings;
CREATE TRIGGER photobooth_booking_no_overlap BEFORE INSERT ON photobooth.photobooth_bookings FOR EACH ROW EXECUTE FUNCTION photobooth.photobooth_booking_no_overlap();`
	if err := validateMigrationSQL([]byte(migration)); err != nil {
		t.Fatalf("rejected Photobooth migration 002 SQL shape: %v", err)
	}
}
