package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

var (
	ErrUpgradeInvalid  = errors.New("external application update is invalid")
	ErrUpgradeNotFound = errors.New("external application update was not found")
	ErrUpgradeState    = errors.New("external application update is no longer available")
)

const upgradeManifestContext = "apexvoid-update-manifest-v1\n"

// ExternalUpgrade is an immutable, verified review snapshot. No SQL runs
// during preview. Only administrators can approve the exact pinned manifest.
type ExternalUpgrade struct {
	ID                uuid.UUID                   `json:"id"`
	ApplicationID     string                      `json:"application_id"`
	Status            string                      `json:"status"`
	InstalledVersion  string                      `json:"installed_version"`
	AvailableVersion  string                      `json:"available_version"`
	InstalledBundle   string                      `json:"installed_bundle_version"`
	AvailableBundle   string                      `json:"available_bundle_version"`
	AddedPermissions  []ExternalPermissionPreview `json:"added_permissions"`
	PendingMigrations []ExternalMigrationPreview  `json:"pending_migrations"`
	ManifestSHA256    string                      `json:"manifest_sha256"`
	ExpiresAt         time.Time                   `json:"expires_at"`
	ErrorCode         string                      `json:"error_code,omitempty"`
	Failure           *MigrationDiagnostic        `json:"failure,omitempty"`
	ErrorMessage      string                      `json:"error_message,omitempty"`
}
type ExternalMigrationPreview struct {
	Version int    `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Status  string `json:"status"`
}
type ExternalPermissionPreview struct {
	Name        string           `json:"name"`
	Scope       permission.Scope `json:"scope"`
	DisplayName string           `json:"display_name"`
}

func compareUpgradeVersion(left, right string) (int, error) {
	// Require stable numeric releases for automatic review comparisons; the
	// wire manifest accepts prereleases, but upgrade of prereleases requires
	// a separate explicit rollout policy.
	parse := func(value string) ([3]uint64, error) {
		var numbers [3]uint64
		parts := strings.Split(value, ".")
		if len(parts) != 3 {
			return numbers, ErrUpgradeInvalid
		}
		for i, part := range parts {
			if part == "" || (len(part) > 1 && part[0] == '0') {
				return numbers, ErrUpgradeInvalid
			}
			n, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				return numbers, ErrUpgradeInvalid
			}
			numbers[i] = n
		}
		return numbers, nil
	}
	a, err := parse(left)
	if err != nil {
		return 0, err
	}
	b, err := parse(right)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1, nil
		}
		if a[i] > b[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func validateUpgrade(installed, next ExternalManifest) ([]ExternalPermissionPreview, []ExternalMigration, error) {
	if err := validateManifest(next); err != nil {
		return nil, nil, err
	}
	if installed.Application.ID != next.Application.ID ||
		installed.Application.APIContractVersion != next.Application.APIContractVersion ||
		installed.Service.Identity != next.Service.Identity ||
		installed.Service.HealthPath != next.Service.HealthPath ||
		installed.Service.EnrollmentPath != next.Service.EnrollmentPath ||
		installed.Service.FrontendRoute != next.Service.FrontendRoute ||
		installed.Service.SettingsRoute != next.Service.SettingsRoute ||
		installed.Service.APIRoute != next.Service.APIRoute ||
		installed.Database.Name != next.Database.Name ||
		installed.Database.Schema != next.Database.Schema ||
		installed.Database.Role != next.Database.Role {
		return nil, nil, fmt.Errorf("%w: application identity, routes, API contract and database ownership must remain stable", ErrUpgradeInvalid)
	}
	comparison, err := compareUpgradeVersion(installed.Application.Version, next.Application.Version)
	if err != nil || comparison >= 0 {
		return nil, nil, fmt.Errorf("%w: new stable application version must be strictly greater", ErrUpgradeInvalid)
	}
	bundleComparison, err := compareUpgradeVersion(installed.Database.MigrationBundleVersion, next.Database.MigrationBundleVersion)
	if err != nil || bundleComparison > 0 {
		return nil, nil, fmt.Errorf("%w: migration bundle version cannot decrease", ErrUpgradeInvalid)
	}
	if len(next.Migrations) < len(installed.Migrations) {
		return nil, nil, fmt.Errorf("%w: previously applied migrations cannot be removed", ErrUpgradeInvalid)
	}
	for index, previous := range installed.Migrations {
		current := next.Migrations[index]
		if previous.Version != current.Version || !strings.EqualFold(previous.SHA256, current.SHA256) || previous.Path != current.Path {
			return nil, nil, fmt.Errorf("%w: an existing migration was modified", ErrUpgradeInvalid)
		}
	}
	addedMigrations := append([]ExternalMigration{}, next.Migrations[len(installed.Migrations):]...)
	if len(addedMigrations) > 0 && bundleComparison >= 0 {
		return nil, nil, fmt.Errorf("%w: new migrations require an increased bundle version", ErrUpgradeInvalid)
	}
	existing := make(map[string]ExternalPermission, len(installed.Permissions))
	for _, item := range installed.Permissions {
		existing[item.Name] = item
	}
	added := []ExternalPermissionPreview{}
	for _, item := range next.Permissions {
		if previous, ok := existing[item.Name]; ok {
			if item.Scope != previous.Scope {
				return nil, nil, fmt.Errorf("%w: permission scopes are immutable", ErrUpgradeInvalid)
			}
			delete(existing, item.Name)
		} else {
			added = append(added, ExternalPermissionPreview{Name: item.Name, Scope: item.Scope, DisplayName: item.DisplayName})
		}
	}
	if len(existing) > 0 {
		return nil, nil, fmt.Errorf("%w: removal of existing permissions requires a separate retirement workflow", ErrUpgradeInvalid)
	}
	return added, addedMigrations, nil
}

func (s *ExternalStore) signedUpdateManifest(ctx context.Context, app ExternalApplication) (ExternalManifest, []byte, error) {
	if app.CredentialRevoked || app.InstallationID == nil {
		return ExternalManifest{}, nil, fmt.Errorf("%w: only active manifest-enrolled applications can be upgraded", ErrUpgradeInvalid)
	}
	if err := validateServiceURL(app.ServiceEndpoint, s.allowedHosts); err != nil {
		return ExternalManifest{}, nil, fmt.Errorf("%w: untrusted application endpoint", ErrUpgradeInvalid)
	}
	var hash string
	if err := s.pool.QueryRow(ctx, `SELECT credential_hash FROM core_external_applications WHERE id=$1 AND status='active' AND credential_revoked_at IS NULL`, app.ID).Scan(&hash); err != nil {
		return ExternalManifest{}, nil, ErrUpgradeInvalid
	}
	key, err := hex.DecodeString(hash)
	if err != nil || len(key) != sha256.Size {
		return ExternalManifest{}, nil, ErrUpgradeInvalid
	}
	challenge, err := newCredential()
	if err != nil {
		return ExternalManifest{}, nil, err
	}
	endpoint, err := manifestEndpoint(app.ServiceEndpoint, ExternalManifestPath)
	if err != nil {
		return ExternalManifest{}, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ExternalManifest{}, nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-ApexVoid-Update-Challenge", challenge)
	response, err := s.client.Do(request)
	if err != nil {
		return ExternalManifest{}, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ExternalManifest{}, nil, fmt.Errorf("%w: signed manifest unavailable (HTTP %d)", ErrUpgradeInvalid, response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxManifestBytes+1))
	if err != nil || len(raw) > maxManifestBytes {
		return ExternalManifest{}, nil, ErrInvalidManifest
	}
	supplied, err := hex.DecodeString(strings.TrimPrefix(response.Header.Get("X-ApexVoid-Update-Signature"), "sha256="))
	if err != nil || len(supplied) != sha256.Size {
		return ExternalManifest{}, nil, fmt.Errorf("%w: update signature is missing", ErrUpgradeInvalid)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(upgradeManifestContext + challenge + "\n"))
	_, _ = mac.Write(raw)
	if !hmac.Equal(supplied, mac.Sum(nil)) {
		return ExternalManifest{}, nil, fmt.Errorf("%w: update manifest signature is invalid", ErrUpgradeInvalid)
	}
	manifest, err := decodeManifest(raw)
	if err != nil {
		return ExternalManifest{}, nil, err
	}
	return manifest, raw, nil
}

func manifestHash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

func upgradeSnapshot(installed, next ExternalManifest, raw []byte, id uuid.UUID, status string, expires time.Time) (ExternalUpgrade, error) {
	added, migrations, err := validateUpgrade(installed, next)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	return ExternalUpgrade{
		ID: id, ApplicationID: installed.Application.ID, Status: status,
		InstalledVersion: installed.Application.Version, AvailableVersion: next.Application.Version,
		InstalledBundle: installed.Database.MigrationBundleVersion, AvailableBundle: next.Database.MigrationBundleVersion,
		AddedPermissions: added, PendingMigrations: migrationPreviews(migrations, migrationPreviewStatus(status)), ManifestSHA256: manifestHash(raw), ExpiresAt: expires,
	}, nil
}

func migrationPreviewStatus(upgradeStatus string) string {
	switch upgradeStatus {
	case "applying":
		return "applying"
	case "applied":
		return "applied"
	case "failed":
		return "failed"
	default:
		return "pending_verification"
	}
}

func migrationPreviews(migrations []ExternalMigration, status string) []ExternalMigrationPreview {
	previews := make([]ExternalMigrationPreview, 0, len(migrations))
	for _, migration := range migrations {
		previews = append(previews, ExternalMigrationPreview{Version: migration.Version, Path: migration.Path, SHA256: migration.SHA256, Status: status})
	}
	return previews
}

func (s *ExternalStore) PreviewUpdate(ctx context.Context, applicationID string) (ExternalUpgrade, error) {
	app, err := s.Get(ctx, applicationID)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	var installed ExternalManifest
	if len(app.InstalledManifest) == 0 || json.Unmarshal(app.InstalledManifest, &installed) != nil {
		return ExternalUpgrade{}, fmt.Errorf("%w: verified installation manifest is required", ErrUpgradeInvalid)
	}
	next, raw, err := s.signedUpdateManifest(ctx, app)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	if next.Application.ID != applicationID {
		return ExternalUpgrade{}, ErrUpgradeInvalid
	}
	if string(mustJSON(installed)) == string(mustJSON(next)) {
		return ExternalUpgrade{ApplicationID: applicationID, Status: "up_to_date", InstalledVersion: installed.Application.Version, AvailableVersion: next.Application.Version, InstalledBundle: installed.Database.MigrationBundleVersion, AvailableBundle: next.Database.MigrationBundleVersion, AddedPermissions: []ExternalPermissionPreview{}, PendingMigrations: []ExternalMigrationPreview{}, ManifestSHA256: manifestHash(raw)}, nil
	}
	id := uuid.New()
	expires := time.Now().Add(30 * time.Minute)
	preview, err := upgradeSnapshot(installed, next, raw, id, "pending_approval", expires)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	// Capture the immutable manifest and the installed version it was reviewed
	// against. Approval compares both and re-verifies a fresh signed response.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('apexvoid-upgrade:' || $1))`, applicationID)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	var current []byte
	if err = tx.QueryRow(ctx, `SELECT installed_manifest FROM core_external_applications WHERE id=$1 AND status='active' FOR UPDATE`, applicationID).Scan(&current); err != nil {
		return ExternalUpgrade{}, err
	}
	if !hmac.Equal([]byte(manifestHash(current)), []byte(manifestHash(app.InstalledManifest))) {
		return ExternalUpgrade{}, fmt.Errorf("%w: installed application changed during preview", ErrUpgradeState)
	}
	_, err = tx.Exec(ctx, `UPDATE core_external_application_upgrades SET status='superseded',updated_at=NOW() WHERE application_id=$1 AND status='pending_approval'`, applicationID)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO core_external_application_upgrades(id,application_id,manifest,manifest_sha256,installed_manifest_sha256,status,expires_at) VALUES($1,$2,$3,$4,$5,'pending_approval',$6)`, id, applicationID, mustJSON(next), manifestHash(raw), manifestHash(current), expires)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	if err = s.recordAuditTx(ctx, tx, applicationID); err != nil {
		return ExternalUpgrade{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ExternalUpgrade{}, err
	}
	return preview, nil
}

func (s *ExternalStore) GetUpdate(ctx context.Context, applicationID string, id uuid.UUID) (ExternalUpgrade, error) {
	app, err := s.Get(ctx, applicationID)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	var nextRaw []byte
	var status, fingerprint, expected string
	var expires time.Time
	var failure, errorCode string
	var errorDetails []byte
	if err := s.pool.QueryRow(ctx, `SELECT manifest,manifest_sha256,installed_manifest_sha256,status,expires_at,error_code,error_message,error_details FROM core_external_application_upgrades WHERE id=$1 AND application_id=$2`, id, applicationID).Scan(&nextRaw, &fingerprint, &expected, &status, &expires, &errorCode, &failure, &errorDetails); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ExternalUpgrade{}, ErrUpgradeNotFound
		}
		return ExternalUpgrade{}, err
	}
	if manifestHash(app.InstalledManifest) != expected && status != "applied" {
		return ExternalUpgrade{}, fmt.Errorf("%w: installed manifest changed after review", ErrUpgradeManifest)
	}
	var next ExternalManifest
	if err := json.Unmarshal(nextRaw, &next); err != nil {
		return ExternalUpgrade{}, err
	}
	// The original installed manifest may have changed on an applied plan.
	var previous ExternalManifest
	if status == "applied" {
		previous = next
	} else if err := json.Unmarshal(app.InstalledManifest, &previous); err != nil {
		return ExternalUpgrade{}, err
	}
	result, err := upgradeSnapshot(previous, next, nextRaw, id, status, expires)
	if status == "applied" || err != nil {
		result = ExternalUpgrade{ID: id, ApplicationID: applicationID, Status: status, AvailableVersion: next.Application.Version, ManifestSHA256: fingerprint, ExpiresAt: expires}
	}
	result.ManifestSHA256 = fingerprint
	result.ErrorCode = errorCode
	if len(errorDetails) > 0 && string(errorDetails) != "{}" {
		var details MigrationDiagnostic
		if json.Unmarshal(errorDetails, &details) == nil && details.Code != "" {
			result.Failure = &details
		}
	}
	if err := s.hydrateMigrationStatuses(ctx, applicationID, &result); err != nil {
		return ExternalUpgrade{}, err
	}
	result.ErrorMessage = failure
	return result, nil
}

func (s *ExternalStore) hydrateMigrationStatuses(ctx context.Context, applicationID string, plan *ExternalUpgrade) error {
	if plan == nil || len(plan.PendingMigrations) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, `SELECT version FROM core_external_application_migrations WHERE application_id=$1`, applicationID)
	if err != nil {
		return err
	}
	defer rows.Close()
	applied := map[int]bool{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return err
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for index := range plan.PendingMigrations {
		migration := &plan.PendingMigrations[index]
		switch {
		case applied[migration.Version]:
			migration.Status = "applied"
		case plan.Status == "applying":
			migration.Status = "applying"
		case plan.Status == "failed" && plan.Failure != nil && plan.Failure.Version == migration.Version:
			migration.Status = "failed"
		case plan.Status == "pending_approval":
			migration.Status = "pending_verification"
		default:
			migration.Status = "pending_approval"
		}
	}
	return nil
}

// ApproveUpdate is intentionally append-only: no permission removal, no scope
// changes, no rewritten migrations, no silent database adoption, no auto-run.
func (s *ExternalStore) ApproveUpdate(ctx context.Context, applicationID string, id uuid.UUID, approvePermissions, approveMigrations bool, expectedHash string) (ExternalUpgrade, error) {
	if !approvePermissions || !approveMigrations || len(expectedHash) != 64 {
		return ExternalUpgrade{}, fmt.Errorf("%w: review both permission and migration changes and confirm the manifest fingerprint", ErrUpgradeInvalid)
	}
	connection, err := s.pool.Acquire(ctx)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	defer connection.Release()
	lock := "apexvoid-upgrade:" + applicationID
	if _, err := connection.Exec(ctx, `SELECT pg_advisory_lock(hashtext($1))`, lock); err != nil {
		return ExternalUpgrade{}, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = connection.Exec(cleanup, `SELECT pg_advisory_unlock(hashtext($1))`, lock)
	}()
	app, err := s.Get(ctx, applicationID)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	var storedRaw []byte
	var fingerprint, baseHash, status string
	var expires time.Time
	if err := s.pool.QueryRow(ctx, `SELECT manifest,manifest_sha256,installed_manifest_sha256,status,expires_at FROM core_external_application_upgrades WHERE id=$1 AND application_id=$2`, id, applicationID).Scan(&storedRaw, &fingerprint, &baseHash, &status, &expires); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ExternalUpgrade{}, ErrUpgradeNotFound
		}
		return ExternalUpgrade{}, err
	}
	if status != "pending_approval" && status != "failed" {
		return ExternalUpgrade{}, ErrUpgradeState
	}
	if time.Now().After(expires) {
		return ExternalUpgrade{}, ErrUpgradeExpired
	}
	if !constantEqual(expectedHash, fingerprint) || manifestHash(app.InstalledManifest) != baseHash {
		return ExternalUpgrade{}, ErrUpgradeManifest
	}
	var current, proposed ExternalManifest
	if json.Unmarshal(app.InstalledManifest, &current) != nil || json.Unmarshal(storedRaw, &proposed) != nil {
		return ExternalUpgrade{}, ErrUpgradeInvalid
	}
	if _, _, err := validateUpgrade(current, proposed); err != nil {
		return ExternalUpgrade{}, err
	}
	// Verify the service STILL serves the same authenticated manifest. This
	// rejects stale previews and prevents a publisher swapping SQL at approval.
	liveManifest, liveRaw, err := s.signedUpdateManifest(ctx, app)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	if liveManifest.Application.ID != applicationID || manifestHash(liveRaw) != fingerprint {
		return ExternalUpgrade{}, fmt.Errorf("%w: service manifest changed since review; preview again", ErrUpgradeManifest)
	}
	// The on-disk schema, role and password remain bound to the original app.
	var name, schema, role string
	var encrypted []byte
	if err := s.pool.QueryRow(ctx, `SELECT database_name,schema_name,role_name,encrypted_password FROM core_external_application_resources WHERE application_id=$1`, applicationID).Scan(&name, &schema, &role, &encrypted); err != nil {
		return ExternalUpgrade{}, err
	}
	if name != app.DatabaseName || schema != app.DatabaseSchema || role != app.DatabaseRole {
		return ExternalUpgrade{}, ErrUpgradeInvalid
	}
	password, err := recoverProvisioningPassword(s.provisioningKey, encrypted)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	if _, err = s.pool.Exec(ctx, `UPDATE core_external_application_upgrades SET status='applying',error_code='',error_message='',error_details='{}'::jsonb,updated_at=NOW() WHERE id=$1 AND status IN ('pending_approval','failed')`, id); err != nil {
		return ExternalUpgrade{}, err
	}
	fail := func(cause error) (ExternalUpgrade, error) {
		code := migrationFailureCode(cause)
		details := []byte(`{}`)
		if failure, ok := migrationFailure(cause); ok {
			details = mustJSON(failure.Diagnostic())
		}
		_, _ = s.pool.Exec(ctx, `UPDATE core_external_application_upgrades SET status='failed',error_code=$2,error_message=$3,error_details=$4,updated_at=NOW() WHERE id=$1 AND status='applying'`, id, code, safeInstallationError(cause), details)
		return ExternalUpgrade{}, cause
	}
	// Check the signed, pinned migration bodies before applying any change.
	for _, migration := range proposed.Migrations[len(current.Migrations):] {
		endpoint, endpointErr := manifestEndpoint(app.ServiceEndpoint, migration.Path)
		if endpointErr != nil {
			return fail(newMigrationFailure(MigrationFetchFailed, migration.Version, migration.Path, "The migration path is invalid and could not be retrieved.", endpointErr))
		}
		body, fetchErr := httpRequestLimit(ctx, s.client, http.MethodGet, endpoint, "", nil, maxMigrationBytes)
		if fetchErr != nil {
			return fail(newMigrationFailure(MigrationFetchFailed, migration.Version, migration.Path, "The application service did not return the migration SQL.", fmt.Errorf("%w: %v", ErrMigrationFetch, fetchErr)))
		}
		if migrationChecksum(body) != strings.ToLower(migration.SHA256) {
			return fail(newMigrationFailure(MigrationChecksumMismatch, migration.Version, migration.Path, "Downloaded bytes differ from the checksum pinned in the signed manifest.", ErrMigrationChecksum))
		}
		if policyErr := validateMigrationSQL(body); policyErr != nil {
			return fail(newMigrationFailure(MigrationPolicyRejected, migration.Version, migration.Path, "The SQL contains a privileged, session-changing, server-side, or cross-application operation.", fmt.Errorf("%w: %v", ErrMigrationPolicy, policyErr)))
		}
	}
	if err := s.applyMigrations(ctx, provisionedDatabase{Name: name, Schema: schema, Role: role, Password: password}, app.ServiceEndpoint, proposed); err != nil {
		return fail(err)
	}

	// All old permissions remain registered; only additions are allowed.
	newDefinitions := []permission.Definition{}
	old := make(map[string]bool, len(current.Permissions))
	for _, entry := range current.Permissions {
		old[entry.Name] = true
	}
	for _, entry := range proposed.Permissions {
		if !old[entry.Name] {
			newDefinitions = append(newDefinitions, permission.Definition{Name: entry.Name, Module: "external." + applicationID, Scope: entry.Scope, DisplayName: entry.DisplayName, Description: entry.Description})
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(newDefinitions) > 0 {
		if err := s.permissions.RegisterBatch(newDefinitions); err != nil {
			return fail(err)
		}
	}
	commitDone := false
	defer func() {
		if !commitDone && len(newDefinitions) > 0 {
			names := make([]string, 0, len(newDefinitions))
			for _, definition := range newDefinitions {
				names = append(names, definition.Name)
			}
			s.permissions.UnregisterBatch("external."+applicationID, names)
		}
	}()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fail(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('core_external_application_catalog'))`); err != nil {
		return fail(err)
	}
	for _, entry := range newDefinitions {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core_external_application_permissions WHERE name=$1 UNION SELECT 1 FROM core_external_permission_tombstones WHERE name=$1)`, entry.Name).Scan(&exists); err != nil {
			return fail(err)
		}
		if exists {
			return fail(ErrExternalDuplicate)
		}
	}
	for _, entry := range proposed.Permissions {
		_, err = tx.Exec(ctx, `INSERT INTO core_external_application_permissions(application_id,name,display_name,description,scope) VALUES($1,$2,$3,$4,$5) ON CONFLICT(application_id,name) DO UPDATE SET display_name=EXCLUDED.display_name,description=EXCLUDED.description`, applicationID, entry.Name, entry.DisplayName, entry.Description, entry.Scope)
		if err != nil {
			return fail(err)
		}
	}
	cmd, err := tx.Exec(ctx, `UPDATE core_external_applications SET display_name=$2,description=$3,version=$4,api_contract_version=$5,access_match=$6,access_permissions=$7,migration_bundle_version=$8,installed_manifest=$9,update_available=FALSE,available_version=$4,available_migration_bundle_version=$8,update_checked_at=NOW(),update_check_error='',updated_at=NOW() WHERE id=$1 AND status='active' AND installed_manifest=$10`, applicationID, proposed.Application.DisplayName, proposed.Application.Description, proposed.Application.Version, proposed.Application.APIContractVersion, proposed.Access.Match, mustJSON(proposed.Access.Permissions), proposed.Database.MigrationBundleVersion, mustJSON(proposed), app.InstalledManifest)
	if err != nil {
		return fail(err)
	}
	if cmd.RowsAffected() != 1 {
		return fail(ErrUpgradeState)
	}
	if _, err = tx.Exec(ctx, `UPDATE core_external_application_upgrades SET status='applied',applied_at=NOW(),updated_at=NOW() WHERE id=$1 AND status='applying'`, id); err != nil {
		return fail(err)
	}
	if err = s.recordAuditTx(ctx, tx, applicationID); err != nil {
		return fail(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fail(err)
	}
	commitDone = true
	for _, entry := range proposed.Permissions {
		_ = s.permissions.UpdatePresentation(entry.Name, entry.DisplayName, entry.Description)
	}
	plan, err := upgradeSnapshot(current, proposed, storedRaw, id, "applied", expires)
	if err != nil {
		return ExternalUpgrade{}, err
	}
	plan.ManifestSHA256 = fingerprint
	return plan, nil
}
