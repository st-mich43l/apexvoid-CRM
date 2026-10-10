package application

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	MigrationFetchFailed      = "MIGRATION_FETCH_FAILED"
	MigrationChecksumMismatch = "MIGRATION_CHECKSUM_MISMATCH"
	MigrationPolicyRejected   = "MIGRATION_POLICY_REJECTED"
	MigrationExecutionFailed  = "MIGRATION_EXECUTION_FAILED"
	MigrationPermissionDenied = "MIGRATION_PERMISSION_DENIED"
	UpgradeManifestChanged    = "UPGRADE_MANIFEST_CHANGED"
	UpgradeReviewExpired      = "UPGRADE_REVIEW_EXPIRED"
)

var (
	ErrMigrationFetch      = errors.New("migration SQL could not be retrieved")
	ErrMigrationChecksum   = errors.New("migration SQL does not match its authenticated checksum")
	ErrMigrationExecution  = errors.New("migration SQL execution failed")
	ErrMigrationPermission = errors.New("application role lacks permission to execute migration")
	ErrUpgradeManifest     = errors.New("external application manifest changed after review")
	ErrUpgradeExpired      = errors.New("external application upgrade review has expired")
)

// MigrationFailure is safe to expose to an administrator. Cause is retained
// for errors.Is/errors.As but is never included in Error or the API payload.
type MigrationFailure struct {
	Code    string
	Version int
	Path    string
	Message string
	Cause   error
}

func (e *MigrationFailure) Error() string {
	return fmt.Sprintf("migration %d (%s): %s", e.Version, safeMigrationPath(e.Path), e.Message)
}

func (e *MigrationFailure) Unwrap() error { return e.Cause }

type MigrationDiagnostic struct {
	Code    string `json:"code"`
	Version int    `json:"version"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

func migrationFailureCode(err error) string {
	if failure, ok := migrationFailure(err); ok {
		return failure.Code
	}
	return ""
}

// MigrationFailureDetails exposes only the safe, structured diagnostic for
// transport handlers. The underlying database or HTTP error is never returned.
func MigrationFailureDetails(err error) (MigrationDiagnostic, bool) {
	failure, ok := migrationFailure(err)
	if !ok {
		return MigrationDiagnostic{}, false
	}
	return failure.Diagnostic(), true
}

func (e *MigrationFailure) Diagnostic() MigrationDiagnostic {
	return MigrationDiagnostic{Code: e.Code, Version: e.Version, Path: safeMigrationPath(e.Path), Message: e.Message}
}

func newMigrationFailure(code string, version int, path, message string, cause error) error {
	return &MigrationFailure{Code: code, Version: version, Path: path, Message: message, Cause: cause}
}

func migrationExecutionFailure(version int, path string, cause error) error {
	code, message, sentinel := MigrationExecutionFailed, "PostgreSQL rejected the migration; review the database error and retry after correction.", ErrMigrationExecution
	var pgErr *pgconn.PgError
	if errors.As(cause, &pgErr) && pgErr.Code == "42501" {
		code, message, sentinel = MigrationPermissionDenied, "The restricted application role lacks a required privilege; no migration changes were committed.", ErrMigrationPermission
	}
	return newMigrationFailure(code, version, path, message, fmt.Errorf("%w: %v", sentinel, cause))
}

func migrationFailure(err error) (*MigrationFailure, bool) {
	var failure *MigrationFailure
	return failure, errors.As(err, &failure)
}

func safeMigrationPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 160 || strings.Contains(value, "..") || strings.ContainsAny(value, "\\\r\n") {
		return "(invalid path)"
	}
	return value
}

// This compatibility pattern remains for older unit tests and callers. The
// actual validator below is statement-aware and is the only runtime policy.
var migrationForbiddenPattern = regexp.MustCompile(`(?i)(drop\s+(database|schema|role|owned)|truncate\s|alter\s+(system|database|role)|create\s+(database|role|schema)|grant\s|revoke\s|create\s+extension|copy\s+[^;]*\s+program|\bset\s+(role|search_path|session_authorization|transaction|constraints)\b|security\s+definer|pg_(read_file|write_file|execute_server_program)|\b(public|pg_catalog|core_|workspace_|apexvoid_)\w*\.)`)

var (
	forbiddenStatement = regexp.MustCompile(`(?i)^\s*(drop\s+(database|schema|role|owned)\b|truncate\b|alter\s+(system|database|role)\b|create\s+(database|role|schema|extension)\b|grant\b|revoke\b|set\s+(role|search_path|session_authorization|transaction|constraints)\b)`)
	forbiddenAnywhere  = regexp.MustCompile(`(?i)(copy\s+[^;]*\s+program\b|security\s+definer\b|pg_(read_file|write_file|execute_server_program)\b|alter\s+[^;]*\s+owner\s+to\b|\b(public|pg_catalog|core_|workspace_|apexvoid_)\w*\s*\.)`)
)

// validateMigrationSQL deliberately understands statement boundaries, quoted
// strings, comments, and PostgreSQL dollar-quoted function bodies. It permits
// normal application-owned DDL/DML such as UPDATE ... SET and ALTER TABLE ...
// SET DEFAULT, while rejecting privilege, session, server-file, and cross-
// schema operations. PostgreSQL role/database privileges remain authoritative.
func validateMigrationSQL(body []byte) error {
	raw := string(body)
	if strings.TrimSpace(raw) == "" {
		return errors.New("migration SQL is empty")
	}
	normalized := normalizeSQLForPolicy(raw)
	if forbiddenAnywhere.MatchString(normalized) {
		return errors.New("privileged, server-side, or cross-application SQL is not permitted")
	}
	for _, statement := range splitSQLStatements(normalized) {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if forbiddenStatement.MatchString(statement) {
			return errors.New("privileged or session-changing SQL is not permitted")
		}
	}
	return nil
}

func normalizeSQLForPolicy(raw string) string {
	var out strings.Builder
	out.Grow(len(raw))
	for i := 0; i < len(raw); {
		switch raw[i] {
		case '-':
			if i+1 < len(raw) && raw[i+1] == '-' {
				out.WriteByte(' ')
				i += 2
				for i < len(raw) && raw[i] != '\n' {
					i++
				}
				continue
			}
		case '/':
			if i+1 < len(raw) && raw[i+1] == '*' {
				i += 2
				for i+1 < len(raw) && !(raw[i] == '*' && raw[i+1] == '/') {
					i++
				}
				if i+1 < len(raw) {
					i += 2
				}
				out.WriteByte(' ')
				continue
			}
		case '\'':
			i++
			for i < len(raw) {
				if raw[i] == '\'' {
					if i+1 < len(raw) && raw[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			out.WriteByte(' ')
			continue
		case '$':
			if end, ok := dollarQuoteEnd(raw, i); ok {
				i = end
				out.WriteByte(' ')
				continue
			}
		}
		out.WriteByte(raw[i])
		i++
	}
	return strings.ToLower(out.String())
}

func dollarQuoteEnd(raw string, start int) (int, bool) {
	end := strings.IndexByte(raw[start+1:], '$')
	if end < 0 {
		return 0, false
	}
	end += start + 1
	for _, ch := range raw[start+1 : end] {
		if !(ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9') {
			return 0, false
		}
	}
	delimiter := raw[start : end+1]
	close := strings.Index(raw[end+1:], delimiter)
	if close < 0 {
		return len(raw), true
	}
	return end + 1 + close + len(delimiter), true
}

func splitSQLStatements(raw string) []string {
	parts := strings.Split(raw, ";")
	return parts
}
