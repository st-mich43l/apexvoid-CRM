package application

import "testing"

func TestMigrationPolicyPermitsOwnedTableDataBackfills(t *testing.T) {
	allowed := []string{
		"UPDATE cafe.cafe_bookings SET status='confirmed' WHERE status='reserved';",
		"UPDATE cafe.cafe_bookings\nSET booking_ref='CAF-123' WHERE booking_ref IS NULL;",
	}
	for _, statement := range allowed {
		if migrationForbiddenPattern.MatchString(statement) {
			t.Errorf("rejected ordinary migration DML: %s", statement)
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
	} {
		if !migrationForbiddenPattern.MatchString(statement) {
			t.Errorf("allowed restricted migration SQL: %s", statement)
		}
	}
}
