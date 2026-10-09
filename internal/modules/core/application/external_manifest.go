package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

const (
	ExternalManifestPath = "/.well-known/apexvoid/manifest.json"
	maxManifestBytes     = 256 << 10
	maxMigrationBytes    = 4 << 20
	installationTTL      = 15 * time.Minute
)

var (
	ErrInvalidManifest        = errors.New("invalid external application manifest")
	ErrInstallationNotFound   = errors.New("external application installation not found")
	ErrInstallationExpired    = errors.New("external application enrollment has expired")
	ErrInstallationState      = errors.New("external application installation is not in an actionable state")
	ErrMigrationPolicy        = errors.New("external migration violates the SQL safety policy")
	manifestVersionPattern    = regexp.MustCompile(`^v[1-9][0-9]*$`)
	semanticVersionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
	databaseIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	sha256Pattern             = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

type ExternalManifest struct {
	ManifestVersion string                                `json:"manifest_version"`
	Application     ManifestApplication                   `json:"application"`
	Service         ManifestService                       `json:"service"`
	Database        ManifestDatabase                      `json:"database"`
	Permissions     []ExternalPermission                  `json:"permissions"`
	Access          frameworkapplication.PermissionPolicy `json:"access"`
	Migrations      []ExternalMigration                   `json:"migrations"`
}

type ManifestApplication struct {
	ID                 string `json:"id"`
	DisplayName        string `json:"display_name"`
	Description        string `json:"description"`
	Version            string `json:"version"`
	APIContractVersion string `json:"api_contract_version"`
}

type ManifestService struct {
	Identity       string `json:"identity"`
	HealthPath     string `json:"health_path"`
	EnrollmentPath string `json:"enrollment_path"`
	FrontendRoute  string `json:"frontend_route"`
	SettingsRoute  string `json:"settings_route"`
	APIRoute       string `json:"api_route"`
}

type ManifestDatabase struct {
	Name                   string `json:"name"`
	Schema                 string `json:"schema"`
	Role                   string `json:"role"`
	MigrationBundleVersion string `json:"migration_bundle_version"`
}

type ExternalMigration struct {
	Version int    `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

type ExternalInstallation struct {
	ID                   uuid.UUID        `json:"id"`
	ApplicationID        string           `json:"application_id"`
	ServiceURL           string           `json:"service_url"`
	Manifest             ExternalManifest `json:"manifest"`
	Status               string           `json:"status"`
	ExpiresAt            time.Time        `json:"expires_at"`
	LastStep             string           `json:"last_step"`
	ErrorMessage         string           `json:"error_message,omitempty"`
	SelectedWorkspaceIDs []uuid.UUID      `json:"selected_workspace_ids"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

type DiscoverExternalInput struct {
	ServiceURL     string
	EnrollmentCode string
	CreatedBy      uuid.UUID
}

type ApproveExternalInput struct {
	EnrollmentCode      string
	ApproveRegistration bool
	ApprovePermissions  bool
	ApproveDatabase     bool
	ApproveSchema       bool
	ApproveMigrations   bool
	WorkspaceIDs        []uuid.UUID
}

type ExternalUpdateReport struct {
	ApplicationID     string   `json:"application_id"`
	InstalledVersion  string   `json:"installed_version"`
	AvailableVersion  string   `json:"available_version"`
	InstalledBundle   string   `json:"installed_bundle_version"`
	AvailableBundle   string   `json:"available_bundle_version"`
	UpdateAvailable   bool     `json:"update_available"`
	PermissionChanges []string `json:"permission_changes"`
	MigrationChanges  []int    `json:"migration_changes"`
}

func decodeManifest(raw []byte) (ExternalManifest, error) {
	if len(raw) == 0 || len(raw) > maxManifestBytes {
		return ExternalManifest{}, fmt.Errorf("%w: manifest size is invalid", ErrInvalidManifest)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest ExternalManifest
	if err := decoder.Decode(&manifest); err != nil {
		return ExternalManifest{}, fmt.Errorf("%w: invalid JSON", ErrInvalidManifest)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ExternalManifest{}, fmt.Errorf("%w: manifest must contain one JSON object", ErrInvalidManifest)
	}
	if err := validateManifest(manifest); err != nil {
		return ExternalManifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest ExternalManifest) error {
	if manifest.ManifestVersion != SupportedExternalContractVersion || !manifestVersionPattern.MatchString(manifest.ManifestVersion) {
		return fmt.Errorf("%w: unsupported manifest version", ErrInvalidManifest)
	}
	if !externalIdentifierPattern.MatchString(manifest.Application.ID) || strings.TrimSpace(manifest.Application.DisplayName) == "" || !semanticVersionPattern.MatchString(manifest.Application.Version) || manifest.Application.APIContractVersion != SupportedExternalContractVersion {
		return fmt.Errorf("%w: invalid application identity or version", ErrInvalidManifest)
	}
	if !externalIdentifierPattern.MatchString(manifest.Service.Identity) {
		return fmt.Errorf("%w: invalid service identity", ErrInvalidManifest)
	}
	for _, route := range []string{manifest.Service.HealthPath, manifest.Service.EnrollmentPath, manifest.Service.APIRoute} {
		if err := validateManifestPath(route, false); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
	}
	if err := validateManifestRoute(manifest.Service.FrontendRoute, manifest.Application.ID, true); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	if manifest.Service.SettingsRoute != "" {
		if err := validateManifestRoute(manifest.Service.SettingsRoute, manifest.Application.ID, false); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
	}
	for _, item := range []string{manifest.Database.Name, manifest.Database.Schema, manifest.Database.Role} {
		if !databaseIdentifierPattern.MatchString(item) || item == "postgres" || item == "public" || strings.HasPrefix(item, "pg_") {
			return fmt.Errorf("%w: invalid database identifier", ErrInvalidManifest)
		}
	}
	canonicalDB, canonicalSchema, canonicalRole := canonicalApplicationDatabase(manifest.Application.ID)
	if len(canonicalDB) > 63 || manifest.Database.Name != canonicalDB || manifest.Database.Schema != canonicalSchema || manifest.Database.Role != canonicalRole {
		return fmt.Errorf("%w: application database, schema and role must use canonical ownership names", ErrInvalidManifest)
	}
	if !semanticVersionPattern.MatchString(manifest.Database.MigrationBundleVersion) {
		return fmt.Errorf("%w: invalid migration bundle version", ErrInvalidManifest)
	}
	if manifest.Access.Match != frameworkapplication.PermissionMatchAll && manifest.Access.Match != frameworkapplication.PermissionMatchAny || len(manifest.Access.Permissions) == 0 {
		return fmt.Errorf("%w: access policy is required", ErrInvalidManifest)
	}
	seen := map[string]bool{}
	for _, item := range manifest.Permissions {
		if len(manifest.Permissions) > 200 || item.Name == "" || !strings.HasPrefix(item.Name, manifest.Application.ID+".") || seen[item.Name] || !item.Scope.Valid() {
			return fmt.Errorf("%w: invalid permission catalog", ErrInvalidManifest)
		}
		if err := permission.ValidateDefinition(permission.Definition{Name: item.Name, Module: "external." + manifest.Application.ID, Scope: item.Scope, DisplayName: item.DisplayName, Description: item.Description}); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		seen[item.Name] = true
	}
	for _, name := range manifest.Access.Permissions {
		if !seen[name] {
			return fmt.Errorf("%w: access policy references an undeclared permission", ErrInvalidManifest)
		}
	}
	if len(manifest.Migrations) > 500 {
		return fmt.Errorf("%w: too many migrations", ErrInvalidManifest)
	}
	for index, migration := range manifest.Migrations {
		if migration.Version != index+1 || !sha256Pattern.MatchString(migration.SHA256) || !strings.HasPrefix(migration.Path, "/.well-known/apexvoid/migrations/") {
			return fmt.Errorf("%w: migrations must be contiguous, pinned, and served from the well-known path", ErrInvalidManifest)
		}
		if err := validateManifestPath(migration.Path, true); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
	}
	return nil
}

func validateManifestPath(value string, allowWellKnown bool) error {
	if value == "" || !strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || strings.Contains(value, "?") || strings.Contains(value, "#") || path.Clean(value) != value || strings.Contains(value, "..") {
		return errors.New("path must be absolute, clean, and contain no query or traversal")
	}
	if allowWellKnown && !strings.HasPrefix(value, "/.well-known/apexvoid/") {
		return errors.New("path must be under /.well-known/apexvoid")
	}
	return nil
}

func validateManifestRoute(value, applicationID string, required bool) error {
	if value == "" && !required {
		return nil
	}
	prefix := "/apps/" + applicationID
	if !(value == prefix || strings.HasPrefix(value, prefix+"/")) || strings.Contains(value, "?") || strings.Contains(value, "#") || path.Clean(value) != value {
		return errors.New("frontend route must be under /apps/{application_id}")
	}
	return nil
}

func manifestEndpoint(base, route string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	joined, err := url.Parse(route)
	if err != nil || joined.IsAbs() || joined.Host != "" {
		return "", ErrInvalidManifest
	}
	baseURL.Path = route
	baseURL.RawQuery = ""
	baseURL.Fragment = ""
	return baseURL.String(), nil
}

func migrationChecksum(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *ExternalStore) Discover(ctx context.Context, input DiscoverExternalInput) (ExternalInstallation, error) {
	if err := validateServiceURL(input.ServiceURL, s.allowedHosts); err != nil || len(input.EnrollmentCode) < minEnrollmentCodeLength || len(input.EnrollmentCode) > 256 {
		return ExternalInstallation{}, fmt.Errorf("%w: invalid service URL or enrollment code", ErrInvalidManifest)
	}
	endpoint, err := manifestEndpoint(strings.TrimRight(input.ServiceURL, "/"), ExternalManifestPath)
	if err != nil {
		return ExternalInstallation{}, err
	}
	request, err := authenticatedManifest(ctx, s.client, endpoint, input.EnrollmentCode)
	if err != nil {
		return ExternalInstallation{}, err
	}
	manifest, err := decodeManifest(request)
	if err != nil {
		return ExternalInstallation{}, err
	}
	if _, err = manifestEndpoint(input.ServiceURL, manifest.Service.HealthPath); err != nil {
		return ExternalInstallation{}, fmt.Errorf("%w: invalid health endpoint", ErrInvalidManifest)
	}
	if s.metadata != nil {
		for _, internal := range s.metadata.Snapshot().Applications {
			if internal.ID == manifest.Application.ID {
				return ExternalInstallation{}, ErrExternalDuplicate
			}
		}
	}
	if s.permissions != nil {
		for _, item := range manifest.Permissions {
			if _, exists := s.permissions.Get(item.Name); exists {
				return ExternalInstallation{}, ErrExternalDuplicate
			}
		}
	}
	var alreadyRegistered bool
	if err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core_external_applications WHERE id=$1 AND status NOT IN ('retired','failed'))`, manifest.Application.ID).Scan(&alreadyRegistered); err != nil {
		return ExternalInstallation{}, err
	}
	if alreadyRegistered {
		return ExternalInstallation{}, ErrExternalDuplicate
	}
	plan := ExternalInstallation{ID: uuid.New(), ApplicationID: manifest.Application.ID, ServiceURL: strings.TrimRight(input.ServiceURL, "/"), Manifest: manifest, Status: "pending_approval", LastStep: "discovered", ExpiresAt: time.Now().Add(installationTTL), SelectedWorkspaceIDs: []uuid.UUID{}}
	raw, _ := json.Marshal(manifest)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ExternalInstallation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('core_external_application_catalog'))`); err != nil {
		return ExternalInstallation{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO core_external_application_installations(id,application_id,service_url,manifest,enrollment_code_hash,status,expires_at,last_step,created_by) VALUES($1,$2,$3,$4,$5,'pending_approval',$6,$7,$8)`, plan.ID, plan.ApplicationID, plan.ServiceURL, raw, hashCredential(input.EnrollmentCode), plan.ExpiresAt, plan.LastStep, nullableUUID(input.CreatedBy))
	if err != nil {
		if isUnique(err) {
			return ExternalInstallation{}, ErrExternalDuplicate
		}
		return ExternalInstallation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ExternalInstallation{}, err
	}
	return plan, nil
}

func (s *ExternalStore) GetInstallation(ctx context.Context, id uuid.UUID) (ExternalInstallation, error) {
	row := s.pool.QueryRow(ctx, `SELECT id,application_id,service_url,manifest,status,expires_at,last_step,error_message,selected_workspace_ids,created_at,updated_at FROM core_external_application_installations WHERE id=$1`, id)
	return scanInstallation(row)
}

func (s *ExternalStore) ListInstallations(ctx context.Context) ([]ExternalInstallation, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,application_id,service_url,manifest,status,expires_at,last_step,error_message,selected_workspace_ids,created_at,updated_at FROM core_external_application_installations ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ExternalInstallation{}
	for rows.Next() {
		item, scanErr := scanInstallation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *ExternalStore) CheckForUpdate(ctx context.Context, id string) (ExternalUpdateReport, error) {
	app, err := s.Get(ctx, id)
	if err != nil {
		return ExternalUpdateReport{}, err
	}
	var installed ExternalManifest
	if len(app.InstalledManifest) == 0 || json.Unmarshal(app.InstalledManifest, &installed) != nil {
		return ExternalUpdateReport{}, fmt.Errorf("%w: installed manifest is unavailable", ErrInvalidManifest)
	}
	endpoint, err := manifestEndpoint(app.ServiceEndpoint, ExternalManifestPath)
	if err != nil {
		return ExternalUpdateReport{}, err
	}
	raw, err := httpRequest(ctx, s.client, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return ExternalUpdateReport{}, err
	}
	available, err := decodeManifest(raw)
	if err != nil {
		return ExternalUpdateReport{}, err
	}
	report := ExternalUpdateReport{ApplicationID: id, InstalledVersion: installed.Application.Version, AvailableVersion: available.Application.Version, InstalledBundle: installed.Database.MigrationBundleVersion, AvailableBundle: available.Database.MigrationBundleVersion, PermissionChanges: []string{}, MigrationChanges: []int{}}
	if report.InstalledVersion != report.AvailableVersion || report.InstalledBundle != report.AvailableBundle {
		report.UpdateAvailable = true
	}
	installedPermissions := map[string]bool{}
	for _, item := range installed.Permissions {
		installedPermissions[item.Name] = true
	}
	availablePermissions := map[string]bool{}
	for _, item := range available.Permissions {
		availablePermissions[item.Name] = true
		if !installedPermissions[item.Name] {
			report.PermissionChanges = append(report.PermissionChanges, "+"+item.Name)
		}
	}
	for name := range installedPermissions {
		if !availablePermissions[name] {
			report.PermissionChanges = append(report.PermissionChanges, "-"+name)
		}
	}
	if len(report.PermissionChanges) > 0 {
		report.UpdateAvailable = true
	}
	installedMigrations := map[int]string{}
	for _, migration := range installed.Migrations {
		installedMigrations[migration.Version] = strings.ToLower(migration.SHA256)
	}
	for _, migration := range available.Migrations {
		if checksum, exists := installedMigrations[migration.Version]; !exists || checksum != strings.ToLower(migration.SHA256) {
			report.MigrationChanges = append(report.MigrationChanges, migration.Version)
			report.UpdateAvailable = true
		}
	}
	sort.Strings(report.PermissionChanges)
	return report, nil
}

func (s *ExternalStore) ApproveAndInstall(ctx context.Context, id uuid.UUID, input ApproveExternalInput) (ExternalInstallation, ExternalApplication, string, error) {
	// Serialize installation attempts across all Enterprise backend instances.
	// This lock is bound to an acquired connection rather than a transaction.
	connection, err := s.pool.Acquire(ctx)
	if err != nil {
		return ExternalInstallation{}, ExternalApplication{}, "", err
	}
	defer connection.Release()
	lockKey := "apexvoid-external-install:" + id.String()
	if _, err = connection.Exec(ctx, "SELECT pg_advisory_lock(hashtext($1))", lockKey); err != nil {
		return ExternalInstallation{}, ExternalApplication{}, "", err
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = connection.Exec(releaseCtx, "SELECT pg_advisory_unlock(hashtext($1))", lockKey)
	}()
	plan, err := s.GetInstallation(ctx, id)
	if err != nil {
		return ExternalInstallation{}, ExternalApplication{}, "", err
	}
	if time.Now().After(plan.ExpiresAt) {
		return s.failInstallation(ctx, plan, ErrInstallationExpired)
	}
	if plan.Status != "pending_approval" && plan.Status != "failed" {
		return ExternalInstallation{}, ExternalApplication{}, "", ErrInstallationState
	}
	if !input.ApproveRegistration || !input.ApprovePermissions || !input.ApproveDatabase || !input.ApproveSchema || !input.ApproveMigrations || len(input.WorkspaceIDs) == 0 {
		return ExternalInstallation{}, ExternalApplication{}, "", fmt.Errorf("%w: every approval and at least one workspace are required", ErrInvalidManifest)
	}
	seenWorkspaces := map[uuid.UUID]bool{}
	for _, workspaceID := range input.WorkspaceIDs {
		if workspaceID == uuid.Nil || seenWorkspaces[workspaceID] {
			return ExternalInstallation{}, ExternalApplication{}, "", fmt.Errorf("%w: workspace selection is invalid", ErrInvalidManifest)
		}
		var exists bool
		if err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_workspaces WHERE id=$1)`, workspaceID).Scan(&exists); err != nil {
			return ExternalInstallation{}, ExternalApplication{}, "", err
		}
		if !exists {
			return ExternalInstallation{}, ExternalApplication{}, "", fmt.Errorf("%w: selected workspace does not exist", ErrInvalidManifest)
		}
		seenWorkspaces[workspaceID] = true
	}
	var enrollmentHash string
	if err = s.pool.QueryRow(ctx, `SELECT enrollment_code_hash FROM core_external_application_installations WHERE id=$1`, id).Scan(&enrollmentHash); err != nil {
		return ExternalInstallation{}, ExternalApplication{}, "", err
	}
	if !constantEqual(enrollmentHash, hashCredential(input.EnrollmentCode)) {
		return ExternalInstallation{}, ExternalApplication{}, "", ErrInvalidCredential
	}
	command, err := s.pool.Exec(ctx, `UPDATE core_external_application_installations SET status='provisioning',last_step='provisioning_database',error_message='',selected_workspace_ids=$2,updated_at=NOW() WHERE id=$1 AND status IN ('pending_approval','failed')`, id, mustJSON(input.WorkspaceIDs))
	if err != nil {
		return ExternalInstallation{}, ExternalApplication{}, "", err
	}
	if command.RowsAffected() != 1 {
		return ExternalInstallation{}, ExternalApplication{}, "", ErrInstallationState
	}
	fail := func(step string, cause error) (ExternalInstallation, ExternalApplication, string, error) {
		_, _ = s.pool.Exec(ctx, `UPDATE core_external_application_installations SET status='failed',last_step=$2,error_message=$3,updated_at=NOW() WHERE id=$1`, id, step, safeInstallationError(cause))
		return ExternalInstallation{}, ExternalApplication{}, "", cause
	}
	database, err := s.provisionDatabase(ctx, id, plan.Manifest)
	if err != nil {
		return fail("provisioning_database", err)
	}
	if _, err = s.pool.Exec(ctx, `UPDATE core_external_application_installations SET status='migrating',last_step='migrating',updated_at=NOW() WHERE id=$1`, id); err != nil {
		return fail("migrating", err)
	}
	if err = s.applyMigrations(ctx, database, plan.ServiceURL, plan.Manifest); err != nil {
		return fail("migrating", err)
	}
	if _, err = s.pool.Exec(ctx, `UPDATE core_external_application_installations SET status='verifying',last_step='enrolling_service',updated_at=NOW() WHERE id=$1`, id); err != nil {
		return fail("enrolling_service", err)
	}
	// Keep the service credential encrypted until activation, allowing retries
	// to re-use the same credential instead of orphaning a provisioned service.
	var protectedCredential []byte
	if err = s.pool.QueryRow(ctx, `SELECT service_credential_encrypted FROM core_external_application_installations WHERE id=$1`, id).Scan(&protectedCredential); err != nil {
		return fail("enrolling_service", err)
	}
	var credential string
	if len(protectedCredential) != 0 {
		credential, err = recoverProvisioningPassword(s.provisioningKey, protectedCredential)
	} else {
		credential, err = newCredential()
		if err == nil {
			protectedCredential, err = protectProvisioningPassword(s.provisioningKey, credential)
		}
		if err == nil {
			_, err = s.pool.Exec(ctx, `UPDATE core_external_application_installations SET service_credential_encrypted=$2 WHERE id=$1`, id, protectedCredential)
		}
	}
	if err != nil {
		return fail("enrolling_service", err)
	}
	enrollmentEndpoint, err := manifestEndpoint(plan.ServiceURL, plan.Manifest.Service.EnrollmentPath)
	if err != nil {
		return fail("enrolling_service", err)
	}
	permissions := append([]ExternalPermission(nil), plan.Manifest.Permissions...)
	app := ExternalApplication{
		ID: plan.Manifest.Application.ID, DisplayName: plan.Manifest.Application.DisplayName,
		Description: plan.Manifest.Application.Description, Version: plan.Manifest.Application.Version,
		APIContractVersion: plan.Manifest.Application.APIContractVersion,
		ServiceIdentity:    plan.Manifest.Service.Identity, ServiceEndpoint: plan.ServiceURL,
		HealthEndpoint: mustManifestEndpoint(plan.ServiceURL, plan.Manifest.Service.HealthPath),
		FrontendRoute:  plan.Manifest.Service.FrontendRoute, SettingsRoute: plan.Manifest.Service.SettingsRoute,
		Access: plan.Manifest.Access, Permissions: permissions, WorkspaceDefaultEnabled: false,
		InstallationID: &plan.ID, DatabaseName: database.Name, DatabaseSchema: database.Schema,
		DatabaseRole: database.Role, MigrationBundleVersion: plan.Manifest.Database.MigrationBundleVersion,
		InstalledManifest: mustJSON(plan.Manifest),
	}
	registered, registeredErr := s.Get(ctx, app.ID)
	if registeredErr != nil && !errors.Is(registeredErr, ErrExternalNotFound) {
		return fail("registering_catalog", registeredErr)
	}
	if registeredErr == nil {
		if registered.InstallationID == nil || *registered.InstallationID != id || registered.Enabled {
			return fail("registering_catalog", errors.New("application already active or owned by a different installation"))
		}
	} else {
		enrollmentPayload, _ := json.Marshal(map[string]any{
			"application_id": plan.ApplicationID, "service_credential": credential,
			"database": map[string]string{
				"name": database.Name, "schema": database.Schema, "role": database.Role,
				"password":                 database.Password,
				"migration_bundle_version": plan.Manifest.Database.MigrationBundleVersion,
			},
			"api_contract_version": plan.Manifest.Application.APIContractVersion,
		})
		if err = postEnrollment(ctx, s.client, enrollmentEndpoint, input.EnrollmentCode, enrollmentPayload); err != nil {
			return fail("enrolling_service", err)
		}
	}
	if _, err = s.pool.Exec(ctx, `UPDATE core_external_application_installations SET status='verifying',last_step='verifying_service',updated_at=NOW() WHERE id=$1`, id); err != nil {
		return fail("verifying_service", err)
	}
	if health := s.Health(ctx, app); health != "healthy" {
		return fail("verifying_service", fmt.Errorf("application health verification failed (%s)", health))
	}
	if registeredErr != nil {
		registered, _, err = s.Register(ctx, RegisterExternalInput{
			Application: app, Credential: credential, WorkspaceDefaultSet: true, DeferActivation: true,
		})
		if err != nil {
			return fail("registering_catalog", err)
		}
	}
	// The service remains gateway-disabled until ALL workspace grants and the
	// installation lifecycle are committed in one Enterprise DB transaction.
	activation, err := s.pool.Begin(ctx)
	if err != nil {
		return fail("assigning_workspaces", err)
	}
	for _, workspaceID := range input.WorkspaceIDs {
		_, err = activation.Exec(ctx, `INSERT INTO core_external_application_workspaces(application_id,workspace_id,enabled) VALUES($1,$2,TRUE) ON CONFLICT(application_id,workspace_id) DO UPDATE SET enabled=TRUE,updated_at=NOW()`, registered.ID, workspaceID)
		if err != nil {
			break
		}
	}
	if err == nil {
		_, err = activation.Exec(ctx, `UPDATE core_external_applications SET enabled=TRUE,updated_at=NOW() WHERE id=$1 AND status='active' AND enabled=FALSE AND installation_id=$2`, registered.ID, id)
	}
	if err == nil {
		_, err = activation.Exec(ctx, `UPDATE core_external_application_installations SET status='active',last_step='active',service_credential_encrypted=NULL,updated_at=NOW() WHERE id=$1 AND status='verifying'`, id)
	}
	if err != nil {
		_ = activation.Rollback(ctx)
		return fail("activating", err)
	}
	if err = activation.Commit(ctx); err != nil {
		return fail("activating", err)
	}
	registered.Enabled = true
	plan, err = s.GetInstallation(ctx, id)
	if err != nil {
		return ExternalInstallation{}, ExternalApplication{}, "", err
	}
	return plan, registered, credential, nil
}

func (s *ExternalStore) failInstallation(ctx context.Context, plan ExternalInstallation, cause error) (ExternalInstallation, ExternalApplication, string, error) {
	_, _ = s.pool.Exec(ctx, `UPDATE core_external_application_installations SET status='failed',last_step='expired',error_message=$2,updated_at=NOW() WHERE id=$1`, plan.ID, safeInstallationError(cause))
	return ExternalInstallation{}, ExternalApplication{}, "", cause
}

func postEnrollment(ctx context.Context, client *http.Client, endpoint, code string, payload []byte) error {
	encrypted, err := sealEnrollment(code, payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encrypted))
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("external service enrollment returned status %d", response.StatusCode)
	}
	return nil
}

func safeInstallationError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 500 {
		message = message[:500]
	}
	for _, secret := range []string{"enrollment", "credential", "password", "token"} {
		if strings.Contains(strings.ToLower(message), secret) {
			return "installation failed; inspect service and platform logs"
		}
	}
	return message
}

func mustManifestEndpoint(base, route string) string {
	value, _ := manifestEndpoint(base, route)
	return value
}

func scanInstallation(row pgxRow) (ExternalInstallation, error) {
	var item ExternalInstallation
	var rawManifest, rawWorkspaces []byte
	if err := row.Scan(&item.ID, &item.ApplicationID, &item.ServiceURL, &rawManifest, &item.Status, &item.ExpiresAt, &item.LastStep, &item.ErrorMessage, &rawWorkspaces, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ExternalInstallation{}, ErrInstallationNotFound
		}
		return ExternalInstallation{}, err
	}
	if err := json.Unmarshal(rawManifest, &item.Manifest); err != nil {
		return ExternalInstallation{}, err
	}
	if err := json.Unmarshal(rawWorkspaces, &item.SelectedWorkspaceIDs); err != nil {
		return ExternalInstallation{}, err
	}
	return item, nil
}

type pgxRow interface{ Scan(dest ...any) error }

func httpRequest(ctx context.Context, client *http.Client, method, endpoint, enrollmentCode string, body io.Reader) ([]byte, error) {
	return httpRequestLimit(ctx, client, method, endpoint, enrollmentCode, body, maxManifestBytes)
}

func httpRequestLimit(ctx context.Context, client *http.Client, method, endpoint, enrollmentCode string, body io.Reader, limit int) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("external service returned status %d", response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
}

func nullableUUID(value uuid.UUID) any {
	if value == uuid.Nil {
		return nil
	}
	return value
}

// Keep imports and manifest output deterministic for callers that compare plans.
func normalizeManifest(manifest *ExternalManifest) {
	sort.Slice(manifest.Permissions, func(i, j int) bool { return manifest.Permissions[i].Name < manifest.Permissions[j].Name })
}
