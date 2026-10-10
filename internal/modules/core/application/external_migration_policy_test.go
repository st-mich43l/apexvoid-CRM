package application

import "testing"

func TestMigrationPolicyPermitsOwnedTableDataBackfills(t *testing.T) {
	allowed := []string{
		"UPDATE cafe.cafe_bookings SET status='confirmed' WHERE status='reserved';",
		"UPDATE cafe.cafe_bookings\nSET booking_ref='CAF-123' WHERE booking_ref IS NULL;",
		"ALTER TABLE cafe.cafe_bookings ALTER COLUMN status SET DEFAULT 'confirmed';",
		"CREATE OR REPLACE FUNCTION cafe.touch() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END; $$;",
		"INSERT INTO cafe.cafe_booking_events(workspace_id, booking_id) SELECT workspace_id, id FROM cafe.cafe_bookings;",
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
		"UPDATE cafe.cafe_bookings SET status='confirmed'; SET ROLE postgres;",
		"GRANT ALL ON cafe.cafe_bookings TO public;",
		"CREATE EXTENSION btree_gist;",
		"ALTER TABLE cafe.cafe_bookings OWNER TO postgres;",
		"SELECT pg_read_file('/etc/passwd');",
		"CREATE OR REPLACE FUNCTION cafe.bad() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER AS $$ BEGIN RETURN NEW; END; $$;",
	} {
		if err := validateMigrationSQL([]byte(statement)); err == nil {
			t.Errorf("allowed restricted migration SQL: %s", statement)
		}
	}
	for _, statement := range []string{
		"-- GRANT ALL ON cafe.cafe_bookings TO public\nUPDATE cafe.cafe_bookings SET notes='security definer and pg_read_file are ordinary text';",
		"INSERT INTO cafe.cafe_booking_events(reason) VALUES ('COPY cafe.cafe_bookings TO PROGRAM');",
	} {
		if err := validateMigrationSQL([]byte(statement)); err != nil {
			t.Errorf("rejected safe comment or string content: %s (%v)", statement, err)
		}
	}
}

func TestMigrationPolicyAcceptsCafesPhaseTwoSafetyShape(t *testing.T) {
	const migration = `ALTER TABLE cafe.cafe_bookings ADD COLUMN IF NOT EXISTS notes VARCHAR(1000) NOT NULL DEFAULT '';
ALTER TABLE cafe.cafe_bookings ALTER COLUMN booking_ref SET DEFAULT ('CAF-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 18)));
UPDATE cafe.cafe_bookings SET status='confirmed' WHERE status='reserved';
CREATE UNIQUE INDEX IF NOT EXISTS cafe_bookings_ref_unique ON cafe.cafe_bookings(workspace_id, booking_ref);
CREATE OR REPLACE FUNCTION cafe.cafe_booking_no_overlap() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END; $$;
DROP TRIGGER IF EXISTS cafe_booking_no_overlap ON cafe.cafe_bookings;
CREATE TRIGGER cafe_booking_no_overlap BEFORE INSERT ON cafe.cafe_bookings FOR EACH ROW EXECUTE FUNCTION cafe.cafe_booking_no_overlap();`
	if err := validateMigrationSQL([]byte(migration)); err != nil {
		t.Fatalf("rejected Café migration 002 SQL shape: %v", err)
	}
}
