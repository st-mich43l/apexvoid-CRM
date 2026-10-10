package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	coreapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/api"
)

var (
	ErrExternalNotFound      = errors.New("external application not found")
	ErrExternalDuplicate     = errors.New("external application is already registered")
	ErrInvalidExternalModule = errors.New("invalid external application contract")
	ErrUnsupportedContract   = errors.New("unsupported external application API contract")
	ErrInvalidCredential     = errors.New("external service authentication failed")
)

const SupportedExternalContractVersion = "v1"

type ExternalPermission struct {
	Name, DisplayName, Description string
	Scope                          permission.Scope
}

type ExternalApplication struct {
	ID, DisplayName, Description, Version, APIContractVersion string
	ServiceIdentity, ServiceEndpoint, HealthEndpoint          string
	FrontendRoute, SettingsRoute                              string
	Access                                                    frameworkapplication.PermissionPolicy
	Permissions                                               []ExternalPermission
	Enabled                                                   bool
	CredentialRevoked                                         bool
	Status                                                    string
	WorkspaceDefaultEnabled                                   bool
	InstallationID                                            *uuid.UUID
	DatabaseName, DatabaseSchema, DatabaseRole                string
	MigrationBundleVersion                                    string
	InstalledManifest                                         json.RawMessage
	UpdateAvailable                                           bool
	AvailableVersion, AvailableMigrationBundleVersion         string
	UpdateCheckedAt                                           *time.Time
	UpdateCheckError                                          string
}

type RegisterExternalInput struct {
	Application         ExternalApplication
	Credential          string
	WorkspaceDefaultSet bool
	DeferActivation     bool
}

// ExternalAuditEvent records an administrative state transition without any
// credential, token, assertion, or endpoint secret material.
type ExternalAuditEvent struct {
	ApplicationID string
	ActorUserID   uuid.UUID
	WorkspaceID   *uuid.UUID
	Action        string
	RequestID     string
}

type ExternalStore struct {
	pool            *pgxpool.Pool
	permissions     *permission.Registry
	metadata        coreapi.MetadataReader
	provisioningURL string
	provisioningKey string
	mu              sync.Mutex
	client          *http.Client
}

func NewExternalStore(pool *pgxpool.Pool, permissions *permission.Registry, metadata coreapi.MetadataReader, provisioningURL, provisioningKey string) *ExternalStore {
	return &ExternalStore{pool: pool, permissions: permissions, metadata: metadata, provisioningURL: provisioningURL, provisioningKey: provisioningKey, client: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (s *ExternalStore) Register(ctx context.Context, input RegisterExternalInput) (ExternalApplication, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app := input.Application
	if !input.WorkspaceDefaultSet {
		app.WorkspaceDefaultEnabled = true
	}
	if err := s.validateExternal(&app); err != nil {
		return ExternalApplication{}, "", err
	}
	credential := input.Credential
	if credential == "" {
		var err error
		credential, err = newCredential()
		if err != nil {
			return ExternalApplication{}, "", err
		}
	}
	if len(credential) < 32 {
		return ExternalApplication{}, "", fmt.Errorf("%w: service credential must be at least 32 characters", ErrInvalidExternalModule)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ExternalApplication{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('core_external_application_catalog'))`); err != nil {
		return ExternalApplication{}, "", err
	}
	if err = s.validateCatalog(ctx, tx, app); err != nil {
		return ExternalApplication{}, "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO core_external_applications (id,display_name,description,version,api_contract_version,service_identity,service_endpoint,health_endpoint,frontend_route,settings_route,access_match,access_permissions,enabled,credential_hash,status,workspace_default_enabled,installation_id,database_name,database_schema,database_role,migration_bundle_version,installed_manifest) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$21,$13,'registering',$14,$15,$16,$17,$18,$19,$20)`, app.ID, app.DisplayName, app.Description, app.Version, app.APIContractVersion, app.ServiceIdentity, app.ServiceEndpoint, app.HealthEndpoint, app.FrontendRoute, app.SettingsRoute, app.Access.Match, mustJSON(app.Access.Permissions), hashCredential(credential), app.WorkspaceDefaultEnabled, app.InstallationID, nullableText(app.DatabaseName), nullableText(app.DatabaseSchema), nullableText(app.DatabaseRole), nullableText(app.MigrationBundleVersion), nullableJSON(app.InstalledManifest), !input.DeferActivation)
	if err != nil {
		if isUnique(err) {
			return ExternalApplication{}, "", ErrExternalDuplicate
		}
		return ExternalApplication{}, "", err
	}
	if err := s.writePermissions(ctx, tx, app); err != nil {
		return ExternalApplication{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExternalApplication{}, "", err
	}
	if err := s.registerPermissions(app); err != nil {
		_, _ = s.pool.Exec(ctx, `DELETE FROM core_external_applications WHERE id=$1 AND status='registering'`, app.ID)
		return ExternalApplication{}, "", fmt.Errorf("register live permission catalog: %w", err)
	}
	// Activate and record the successful registration in one transaction. A
	// failed audit must never strand an active app with an undisclosed credential.
	activation, err := s.pool.Begin(ctx)
	if err == nil {
		var command pgconn.CommandTag
		command, err = activation.Exec(ctx, `UPDATE core_external_applications SET status='active',updated_at=NOW() WHERE id=$1 AND status='registering'`, app.ID)
		if err == nil && command.RowsAffected() != 1 {
			err = ErrExternalNotFound
		}
		if err == nil {
			err = s.recordAuditTx(ctx, activation, app.ID)
		}
		if err == nil {
			err = activation.Commit(ctx)
		}
		if err != nil {
			_ = activation.Rollback(ctx)
		}
	}
	if err != nil {
		s.unregisterPermissions(app)
		_, _ = s.pool.Exec(ctx, `DELETE FROM core_external_applications WHERE id=$1 AND status='registering'`, app.ID)
		return ExternalApplication{}, "", fmt.Errorf("activate external application: %w", err)
	}
	app.Status = "active"
	app.Enabled = !input.DeferActivation
	return app, credential, nil
}

func (s *ExternalStore) Update(ctx context.Context, app ExternalApplication) (ExternalApplication, error) {
	if err := s.validateExternal(&app); err != nil {
		return ExternalApplication{}, err
	}
	current, err := s.Get(ctx, app.ID)
	if err != nil {
		return ExternalApplication{}, err
	}
	if current.InstallationID != nil && (app.Version != current.Version || app.APIContractVersion != current.APIContractVersion) {
		return ExternalApplication{}, fmt.Errorf("%w: installed application version and API contract come from its verified manifest, not metadata edits", ErrInvalidExternalModule)
	}
	if !samePermissionCatalog(current.Permissions, app.Permissions) {
		return ExternalApplication{}, fmt.Errorf("%w: application-owned permissions are immutable after registration", ErrInvalidExternalModule)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ExternalApplication{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `UPDATE core_external_applications SET display_name=$2,description=$3,version=$4,api_contract_version=$5,service_identity=$6,service_endpoint=$7,health_endpoint=$8,frontend_route=$9,settings_route=$10,access_match=$11,access_permissions=$12,updated_at=NOW() WHERE id=$1`, app.ID, app.DisplayName, app.Description, app.Version, app.APIContractVersion, app.ServiceIdentity, app.ServiceEndpoint, app.HealthEndpoint, app.FrontendRoute, app.SettingsRoute, app.Access.Match, mustJSON(app.Access.Permissions))
	if err != nil {
		return ExternalApplication{}, err
	}
	if command.RowsAffected() == 0 {
		return ExternalApplication{}, ErrExternalNotFound
	}
	if _, err = tx.Exec(ctx, `DELETE FROM core_external_application_permissions WHERE application_id=$1`, app.ID); err != nil {
		return ExternalApplication{}, err
	}
	if err = s.writePermissions(ctx, tx, app); err != nil {
		return ExternalApplication{}, err
	}
	if err = s.recordAuditTx(ctx, tx, app.ID); err != nil {
		return ExternalApplication{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ExternalApplication{}, err
	}
	if err = s.registerPermissions(app); err != nil {
		return ExternalApplication{}, err
	}
	return app, nil
}

func (s *ExternalStore) writePermissions(ctx context.Context, tx pgx.Tx, app ExternalApplication) error {
	for _, item := range app.Permissions {
		_, err := tx.Exec(ctx, `INSERT INTO core_external_application_permissions (application_id,name,display_name,description,scope) VALUES ($1,$2,$3,$4,$5)`, app.ID, item.Name, item.DisplayName, item.Description, item.Scope)
		if err != nil {
			if isUnique(err) {
				return fmt.Errorf("%w: permission %s", ErrExternalDuplicate, item.Name)
			}
			return err
		}
	}
	return nil
}

func (s *ExternalStore) List(ctx context.Context) ([]ExternalApplication, error) {
	rows, err := s.pool.Query(ctx, externalApplicationSelect+` WHERE status='active' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ExternalApplication{}
	for rows.Next() {
		app, err := scanExternal(rows)
		if err != nil {
			return nil, err
		}
		permissions, err := s.permissionsFor(ctx, app.ID)
		if err != nil {
			return nil, err
		}
		app.Permissions = permissions
		items = append(items, app)
	}
	return items, rows.Err()
}

func (s *ExternalStore) Get(ctx context.Context, id string) (ExternalApplication, error) {
	row := s.pool.QueryRow(ctx, externalApplicationSelect+` WHERE id=$1 AND status='active'`, id)
	app, err := scanExternal(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalApplication{}, ErrExternalNotFound
	}
	if err != nil {
		return ExternalApplication{}, err
	}
	app.Permissions, err = s.permissionsFor(ctx, app.ID)
	if err != nil {
		return ExternalApplication{}, err
	}
	return app, nil
}

func (s *ExternalStore) SetWorkspaceEnabled(ctx context.Context, applicationID string, workspaceID uuid.UUID, enabled bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_workspaces WHERE id=$1)`, workspaceID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrExternalNotFound
	}
	command, err := tx.Exec(ctx, `INSERT INTO core_external_application_workspaces(application_id,workspace_id,enabled) VALUES($1,$2,$3) ON CONFLICT(application_id,workspace_id) DO UPDATE SET enabled=EXCLUDED.enabled,updated_at=NOW()`, applicationID, workspaceID, enabled)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrExternalNotFound
	}
	if err := s.recordAuditTx(ctx, tx, applicationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *ExternalStore) EnabledInWorkspace(ctx context.Context, applicationID string, workspaceID uuid.UUID) (bool, error) {
	var global bool
	var workspaceDefault bool
	err := s.pool.QueryRow(ctx, `SELECT enabled,workspace_default_enabled FROM core_external_applications WHERE id=$1`, applicationID).Scan(&global, &workspaceDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrExternalNotFound
	}
	if err != nil {
		return false, err
	}
	if !global {
		return false, nil
	}
	var enabled bool
	err = s.pool.QueryRow(ctx, `SELECT enabled FROM core_external_application_workspaces WHERE application_id=$1 AND workspace_id=$2`, applicationID, workspaceID).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return workspaceDefault, nil
	}
	return enabled, err
}
func (s *ExternalStore) RevokeCredential(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `UPDATE core_external_applications SET credential_revoked_at=NOW(),updated_at=NOW() WHERE id=$1 AND status='active'`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrExternalNotFound
	}
	if err := s.recordAuditTx(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *ExternalStore) RotateCredential(ctx context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential, err := newCredential()
	if err != nil {
		return "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `UPDATE core_external_applications SET credential_hash=$2,credential_rotated_at=NOW(),updated_at=NOW() WHERE id=$1 AND status='active' AND credential_revoked_at IS NULL`, id, hashCredential(credential))
	if err != nil {
		return "", err
	}
	if command.RowsAffected() == 0 {
		return "", ErrExternalNotFound
	}
	if err := s.recordAuditTx(ctx, tx, id); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return credential, nil
}

type externalAuditContextKey struct{}

// WithExternalAudit attaches validated administrator identity to one mutation.
// The mutation itself writes the audit record in its database transaction.
func WithExternalAudit(ctx context.Context, event ExternalAuditEvent) context.Context {
	return context.WithValue(ctx, externalAuditContextKey{}, event)
}

func (s *ExternalStore) recordAuditTx(ctx context.Context, tx pgx.Tx, applicationID string) error {
	event, ok := ctx.Value(externalAuditContextKey{}).(ExternalAuditEvent)
	if !ok {
		return nil // Non-HTTP callers may not have an administrator audit identity.
	}
	if event.ApplicationID != applicationID || event.ActorUserID == uuid.Nil || event.Action == "" {
		return errors.New("invalid external application audit context")
	}
	_, err := tx.Exec(ctx, `INSERT INTO core_external_application_audit(id,application_id,actor_user_id,workspace_id,action,request_id) VALUES($1,$2,$3,$4,$5,$6)`, uuid.New(), event.ApplicationID, event.ActorUserID, event.WorkspaceID, event.Action, event.RequestID)
	return err
}
func (s *ExternalStore) Unregister(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('core_external_application_catalog'))`); err != nil {
		return err
	}
	for _, item := range app.Permissions {
		if _, err = tx.Exec(ctx, `INSERT INTO core_external_permission_tombstones(name,application_id) VALUES($1,$2) ON CONFLICT(name) DO NOTHING`, item.Name, app.ID); err != nil {
			return err
		}
	}
	command, err := tx.Exec(ctx, `UPDATE core_external_applications SET status='retired',enabled=FALSE,credential_revoked_at=COALESCE(credential_revoked_at,NOW()),updated_at=NOW() WHERE id=$1 AND status='active'`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrExternalNotFound
	}
	if err = s.recordAuditTx(ctx, tx, id); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.unregisterPermissions(app)
	return nil
}

func (s *ExternalStore) Authenticate(ctx context.Context, applicationID, credential string) (ExternalApplication, error) {
	app, err := s.Get(ctx, applicationID)
	if err != nil {
		return ExternalApplication{}, ErrInvalidCredential
	}
	var hash string
	var revoked *time.Time
	err = s.pool.QueryRow(ctx, `SELECT credential_hash,credential_revoked_at FROM core_external_applications WHERE id=$1`, applicationID).Scan(&hash, &revoked)
	if err != nil || revoked != nil || !constantEqual(hash, hashCredential(credential)) {
		return ExternalApplication{}, ErrInvalidCredential
	}
	return app, nil
}
func (s *ExternalStore) Health(ctx context.Context, app ExternalApplication) string {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, app.HealthEndpoint, nil)
	if err != nil {
		return "invalid"
	}
	response, err := s.client.Do(request)
	if err != nil {
		return "unavailable"
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		return "healthy"
	}
	return "unhealthy"
}
func (s *ExternalStore) Hydrate(ctx context.Context) error {
	// A process crash between staging and activation never exposes the module;
	// discard that incomplete catalog before rebuilding the live registry.
	if _, err := s.pool.Exec(ctx, `DELETE FROM core_external_applications WHERE status='registering'`); err != nil {
		return fmt.Errorf("recover staged external applications: %w", err)
	}
	items, err := s.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := s.registerPermissions(item); err != nil {
			return err
		}
	}
	return nil
}
func (s *ExternalStore) permissionsFor(ctx context.Context, id string) ([]ExternalPermission, error) {
	rows, err := s.pool.Query(ctx, `SELECT name,display_name,description,scope FROM core_external_application_permissions WHERE application_id=$1 ORDER BY name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ExternalPermission{}
	for rows.Next() {
		var item ExternalPermission
		if err := rows.Scan(&item.Name, &item.DisplayName, &item.Description, &item.Scope); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *ExternalStore) registerPermissions(app ExternalApplication) error {
	definitions := make([]permission.Definition, 0, len(app.Permissions))
	existing := 0
	for _, item := range app.Permissions {
		if current, ok := s.permissions.Get(item.Name); ok {
			if current.Module != "external."+app.ID || current.Scope != item.Scope {
				return fmt.Errorf("%w: permission %s is already owned by %s", ErrExternalDuplicate, item.Name, current.Module)
			}
			existing++
			continue
		}
		definitions = append(definitions, permission.Definition{Name: item.Name, Module: "external." + app.ID, Scope: item.Scope, DisplayName: item.DisplayName, Description: item.Description})
	}
	if existing == len(app.Permissions) {
		for _, item := range app.Permissions {
			if err := s.permissions.UpdatePresentation(item.Name, item.DisplayName, item.Description); err != nil {
				return err
			}
		}
		return nil
	}
	if existing != 0 {
		return fmt.Errorf("%w: partial live permission registration", ErrExternalDuplicate)
	}
	return s.permissions.RegisterBatch(definitions)
}
func (s *ExternalStore) unregisterPermissions(app ExternalApplication) {
	names := make([]string, 0, len(app.Permissions))
	for _, item := range app.Permissions {
		names = append(names, item.Name)
	}
	s.permissions.UnregisterBatch("external."+app.ID, names)
}
func (s *ExternalStore) Descriptor(app ExternalApplication) frameworkapplication.Descriptor {
	access := frameworkapplication.Access{Entry: app.Access}
	if app.SettingsRoute != "" {
		policy := app.Access
		access.Settings = &policy
	}
	return frameworkapplication.Descriptor{ID: app.ID, DisplayName: app.DisplayName, Description: app.Description, Version: app.Version, APIContractVersion: app.APIContractVersion, Frontend: frameworkapplication.Frontend{EntryRoute: app.FrontendRoute, NavigationID: app.ID}, Settings: settings(app.SettingsRoute), Access: access, Deployment: frameworkapplication.DeploymentExternal, External: &frameworkapplication.ExternalService{ServiceIdentity: app.ServiceIdentity, Endpoint: app.ServiceEndpoint, HealthEndpoint: app.HealthEndpoint}}
}
func settings(route string) *frameworkapplication.Settings {
	if route == "" {
		return nil
	}
	return &frameworkapplication.Settings{Route: route}
}

var externalIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
var apiContractPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

func validateExternal(app *ExternalApplication) error {
	if app == nil || strings.TrimSpace(app.ID) == "" || strings.TrimSpace(app.DisplayName) == "" || strings.TrimSpace(app.Version) == "" || app.APIContractVersion == "" || app.ServiceIdentity == "" {
		return ErrInvalidExternalModule
	}
	if !externalIdentifierPattern.MatchString(app.ID) || !externalIdentifierPattern.MatchString(app.ServiceIdentity) || !apiContractPattern.MatchString(app.APIContractVersion) {
		return fmt.Errorf("%w: invalid application identity or API contract version", ErrInvalidExternalModule)
	}
	if app.APIContractVersion != SupportedExternalContractVersion {
		return fmt.Errorf("%w: %s is supported", ErrUnsupportedContract, SupportedExternalContractVersion)
	}
	for _, value := range []string{app.ServiceEndpoint, app.HealthEndpoint} {
		if err := validateServiceURL(value); err != nil {
			return fmt.Errorf("%w: invalid service URL", ErrInvalidExternalModule)
		}
	}
	if app.FrontendRoute != "" && !strings.HasPrefix(app.FrontendRoute, "/") {
		return fmt.Errorf("%w: frontend route must start with /", ErrInvalidExternalModule)
	}
	if app.FrontendRoute != "" && !strings.HasPrefix(app.FrontendRoute, "/apps/"+app.ID) {
		return fmt.Errorf("%w: external frontend route must use /apps/%s", ErrInvalidExternalModule, app.ID)
	}
	if app.SettingsRoute != "" && !strings.HasPrefix(app.SettingsRoute, "/") {
		return fmt.Errorf("%w: settings route must start with /", ErrInvalidExternalModule)
	}
	if app.Access.Match != frameworkapplication.PermissionMatchAll && app.Access.Match != frameworkapplication.PermissionMatchAny || len(app.Access.Permissions) == 0 {
		return fmt.Errorf("%w: access policy is required", ErrInvalidExternalModule)
	}
	seen := map[string]bool{}
	for _, p := range app.Permissions {
		if p.Name == "" || p.DisplayName == "" || !p.Scope.Valid() || seen[p.Name] || !strings.HasPrefix(p.Name, app.ID+".") {
			return fmt.Errorf("%w: invalid application-owned permission", ErrInvalidExternalModule)
		}
		if err := permission.ValidateDefinition(permission.Definition{Name: p.Name, Module: "external." + app.ID, Scope: p.Scope, DisplayName: p.DisplayName, Description: p.Description}); err != nil {
			return fmt.Errorf("%w: invalid application-owned permission", ErrInvalidExternalModule)
		}
		seen[p.Name] = true
	}
	for _, name := range app.Access.Permissions {
		if !seen[name] {
			return fmt.Errorf("%w: access policy must use a declared application permission", ErrInvalidExternalModule)
		}
	}
	sort.Strings(app.Access.Permissions)
	return nil
}
func (s *ExternalStore) validateExternal(app *ExternalApplication) error {
	return validateExternal(app)
}
func validateServiceURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme != "http" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return ErrInvalidExternalModule
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || host == "localhost" || net.ParseIP(host) != nil {
		return ErrInvalidExternalModule
	}
	port := u.Port()
	if port == "" {
		return ErrInvalidExternalModule
	}
	if _, err := net.LookupPort("tcp", port); err != nil {
		return ErrInvalidExternalModule
	}
	return nil
}
func (s *ExternalStore) validateCatalog(ctx context.Context, tx pgx.Tx, app ExternalApplication) error {
	if s.metadata != nil {
		for _, internal := range s.metadata.Snapshot().Applications {
			if internal.ID == app.ID {
				return fmt.Errorf("%w: application id collides with internal application", ErrExternalDuplicate)
			}
		}
	}
	for _, item := range app.Permissions {
		if existing, ok := s.permissions.Get(item.Name); ok {
			return fmt.Errorf("%w: permission %s collides with %s", ErrExternalDuplicate, item.Name, existing.Module)
		}
		var databaseOwner string
		if err := tx.QueryRow(ctx, `SELECT application_id FROM core_external_application_permissions WHERE name=$1`, item.Name).Scan(&databaseOwner); err == nil {
			return fmt.Errorf("%w: permission %s is already owned by %s", ErrExternalDuplicate, item.Name, databaseOwner)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var owner string
		err := tx.QueryRow(ctx, `SELECT application_id FROM core_external_permission_tombstones WHERE name=$1`, item.Name).Scan(&owner)
		if err == nil {
			return fmt.Errorf("%w: permission %s is retired and cannot be restored", ErrExternalDuplicate, item.Name)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	return nil
}
func samePermissionCatalog(left, right []ExternalPermission) bool {
	if len(left) != len(right) {
		return false
	}
	clone := append([]ExternalPermission{}, left...)
	sort.Slice(clone, func(i, j int) bool { return clone[i].Name < clone[j].Name })
	other := append([]ExternalPermission{}, right...)
	sort.Slice(other, func(i, j int) bool { return other[i].Name < other[j].Name })
	for i := range clone {
		if clone[i].Name != other[i].Name || clone[i].Scope != other[i].Scope {
			return false
		}
	}
	return true
}
func scanExternal(row pgx.Row) (ExternalApplication, error) {
	var app ExternalApplication
	var raw []byte
	err := row.Scan(&app.ID, &app.DisplayName, &app.Description, &app.Version, &app.APIContractVersion, &app.ServiceIdentity, &app.ServiceEndpoint, &app.HealthEndpoint, &app.FrontendRoute, &app.SettingsRoute, &app.Access.Match, &raw, &app.Enabled, &app.CredentialRevoked, &app.Status, &app.WorkspaceDefaultEnabled, &app.InstallationID, &app.DatabaseName, &app.DatabaseSchema, &app.DatabaseRole, &app.MigrationBundleVersion, &app.InstalledManifest, &app.UpdateAvailable, &app.AvailableVersion, &app.AvailableMigrationBundleVersion, &app.UpdateCheckedAt, &app.UpdateCheckError)
	if err != nil {
		return app, err
	}
	err = json.Unmarshal(raw, &app.Access.Permissions)
	return app, err
}

const externalApplicationSelect = `SELECT id,display_name,description,version,api_contract_version,service_identity,service_endpoint,health_endpoint,frontend_route,settings_route,access_match,access_permissions,enabled,credential_revoked_at IS NOT NULL,status,workspace_default_enabled,installation_id,COALESCE(database_name,''),COALESCE(database_schema,''),COALESCE(database_role,''),COALESCE(migration_bundle_version,''),COALESCE(installed_manifest,'null'::jsonb),update_available,COALESCE(available_version,''),COALESCE(available_migration_bundle_version,''),update_checked_at,COALESCE(update_check_error,'') FROM core_external_applications`

func mustJSON(value any) []byte { raw, _ := json.Marshal(value); return raw }
func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return value
}
func newCredential() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
func hashCredential(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func constantEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
func isUnique(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
