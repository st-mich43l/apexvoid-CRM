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
	forbiddenStatement  = regexp.MustCompile(`(?i)^\s*(drop\s+(database|schema|role|owned)\b|truncate\b|alter\s+(system|database|role)\b|create\s+(database|role|schema|extension)\b|grant\b|revoke\b|set\s+(role|search_path|session_authorization|transaction|constraints)\b)`)
	forbiddenAnywhere   = regexp.MustCompile(`(?i)(copy\s+[^;]*\s+program\b|security\s+definer\b|pg_(read_file|write_file|execute_server_program)\b|alter\s+[^;]*\s+owner\s+to\b|\b(core_external_[a-z0-9_]+|workspace_(workspaces|memberships)|apexvoid_schema_migrations)\b|\b(public|pg_catalog|information_schema|core(?:_[a-z0-9_]*)?|workspace(?:_[a-z0-9_]*)?|apexvoid(?:_[a-z0-9_]*)?)\s*\.)`)
	dynamicSQLPattern   = regexp.MustCompile(`(?i)\b(do|execute)\b`)
	proceduralForbidden = regexp.MustCompile(`(?i)^(create|alter|drop|truncate|grant|revoke|copy|set|vacuum|call)\b`)
	anonymousBlock      = regexp.MustCompile(`(?i)^do\b`)
	allowedStatement    = regexp.MustCompile(`(?i)^(alter\s+table|create\s+(or\s+replace\s+)?(unique\s+)?(table|index|trigger|function)|drop\s+trigger|insert\s+into|update\b)`)
	functionDeclaration = regexp.MustCompile(`(?i)^create\s+(or\s+replace\s+)?function\b`)
	plpgsqlDeclaration  = regexp.MustCompile(`(?i)\blanguage\s+plpgsql\b`)
)

// validateMigrationSQL is a deliberately narrow lexical policy, not a SQL
// execution engine. It understands PostgreSQL comments, quoted strings,
// identifiers, statement boundaries, and dollar-quoted bodies. Dollar-quoted
// executable bodies are inspected separately; they are never discarded.
// PostgreSQL role privileges remain an independent security boundary.
func validateMigrationSQL(body []byte) error {
	raw := string(body)
	if strings.TrimSpace(raw) == "" {
		return errors.New("migration SQL is empty")
	}
	normalized, bodies, err := scanSQLForPolicy(raw)
	if err != nil {
		return err
	}
	if forbiddenAnywhere.MatchString(normalized) {
		return errors.New("privileged, server-side, or cross-application SQL is not permitted")
	}
	for _, statement := range splitSQLStatements(normalized) {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if forbiddenStatement.MatchString(statement) || anonymousBlock.MatchString(statement) {
			return errors.New("privileged or session-changing SQL is not permitted")
		}
		if !allowedStatement.MatchString(statement) {
			return errors.New("unsupported SQL statement is not permitted in application migrations")
		}
		if functionDeclaration.MatchString(statement) && !plpgsqlDeclaration.MatchString(statement) {
			return errors.New("only PL/pgSQL application trigger functions are permitted")
		}
	}
	for _, body := range bodies {
		if err := validateProcedureBody(body); err != nil {
			return err
		}
	}
	return nil
}

func validateProcedureBody(body string) error {
	normalized, _, err := scanSQLForPolicy(body)
	if err != nil {
		return errors.New("nested dollar-quoted procedural bodies are not supported")
	}
	if dynamicSQLPattern.MatchString(normalized) {
		return errors.New("anonymous blocks and dynamic SQL are not permitted in procedural bodies")
	}
	if forbiddenAnywhere.MatchString(normalized) {
		return errors.New("privileged, server-side, or cross-application SQL is not permitted in procedural bodies")
	}
	for _, statement := range splitSQLStatements(normalized) {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if forbiddenStatement.MatchString(statement) || proceduralForbidden.MatchString(statement) {
			return errors.New("unsupported or privileged procedural SQL is not permitted")
		}
	}
	return nil
}

// scanSQLForPolicy masks comments, literals, quoted identifiers, and dollar
// delimiters while preserving executable SQL keywords. It returns every
// dollar-quoted body for independent inspection.
func scanSQLForPolicy(raw string) (string, []string, error) {
	var out strings.Builder
	var bodies []string
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
				depth := 1
				for i < len(raw) && depth > 0 {
					if i+1 < len(raw) && raw[i] == '/' && raw[i+1] == '*' {
						depth++
						i += 2
						continue
					}
					if i+1 < len(raw) && raw[i] == '*' && raw[i+1] == '/' {
						depth--
						i += 2
						continue
					}
					i++
				}
				if depth != 0 {
					return "", nil, errors.New("unterminated nested SQL comment")
				}
				out.WriteByte(' ')
				continue
			}
		case '\'':
			end, err := quotedLiteralEnd(raw, i, '\'')
			if err != nil {
				return "", nil, err
			}
			i = end
			out.WriteByte(' ')
			continue
		case '"':
			identifier, end, err := quotedIdentifierEnd(raw, i)
			if err != nil {
				return "", nil, err
			}
			i = end
			out.WriteString(identifier)
			continue
		case '$':
			if body, end, ok, malformed := dollarQuoteBounds(raw, i); ok || malformed {
				if malformed {
					return "", nil, errors.New("unterminated dollar-quoted SQL body")
				}
				bodies = append(bodies, body)
				i = end
				out.WriteByte(' ')
				continue
			}
		}
		out.WriteByte(raw[i])
		i++
	}
	return strings.ToLower(out.String()), bodies, nil
}

func quotedLiteralEnd(raw string, start int, quote byte) (int, error) {
	for i := start + 1; i < len(raw); i++ {
		if quote == '\'' && raw[i] == '\\' {
			i++
			continue
		}
		if raw[i] != quote {
			continue
		}
		if i+1 < len(raw) && raw[i+1] == quote {
			i++
			continue
		}
		return i + 1, nil
	}
	return 0, errors.New("unterminated quoted SQL literal or identifier")
}

func quotedIdentifierEnd(raw string, start int) (string, int, error) {
	var identifier strings.Builder
	for i := start + 1; i < len(raw); i++ {
		if raw[i] != '"' {
			if raw[i] == ';' {
				identifier.WriteByte(' ')
			} else {
				identifier.WriteByte(raw[i])
			}
			continue
		}
		if i+1 < len(raw) && raw[i+1] == '"' {
			identifier.WriteByte('"')
			i++
			continue
		}
		return identifier.String(), i + 1, nil
	}
	return "", 0, errors.New("unterminated quoted SQL identifier")
}

func dollarQuoteBounds(raw string, start int) (string, int, bool, bool) {
	end := strings.IndexByte(raw[start+1:], '$')
	if end < 0 {
		return "", 0, false, false
	}
	end += start + 1
	tag := raw[start+1 : end]
	for index, ch := range tag {
		if !(ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || index > 0 && ch >= '0' && ch <= '9') {
			return "", 0, false, false
		}
	}
	delimiter := raw[start : end+1]
	close := strings.Index(raw[end+1:], delimiter)
	if close < 0 {
		return "", 0, false, true
	}
	close += end + 1
	return raw[end+1 : close], close + len(delimiter), true, false
}

func splitSQLStatements(raw string) []string {
	parts := strings.Split(raw, ";")
	return parts
}
