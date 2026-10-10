export type DocBlock =
  | { type: 'paragraph'; text: string }
  | { type: 'list'; items: string[]; ordered?: boolean }
  | { type: 'code'; language: string; title: string; value: string }
  | { type: 'callout'; tone: 'info' | 'warning' | 'success'; title: string; text: string }
  | { type: 'table'; headings: [string, string]; rows: [string, string][] }

export type DocSection = { id: string; title: string; blocks: DocBlock[] }
export type DocCategory = 'Start here' | 'Build applications' | 'Platform architecture' | 'Run & maintain'
export type DocGuide = {
  slug: string
  category: DocCategory
  title: string
  description: string
  minutes: number
  keywords: string[]
  sections: DocSection[]
  related: string[]
}

export const categories: DocCategory[] = ['Start here', 'Build applications', 'Platform architecture', 'Run & maintain']

const p = (text: string): DocBlock => ({ type: 'paragraph', text })
const steps = (...items: string[]): DocBlock => ({ type: 'list', ordered: true, items })
const bullets = (...items: string[]): DocBlock => ({ type: 'list', items })
const snippet = (title: string, language: string, ...lines: string[]): DocBlock => ({ type: 'code', title, language, value: lines.join('\n') })
const note = (title: string, text: string, tone: 'info' | 'warning' | 'success' = 'info'): DocBlock => ({ type: 'callout', title, text, tone })
const table = (headings: [string, string], ...rows: [string, string][]): DocBlock => ({ type: 'table', headings, rows })

export const guides: DocGuide[] = [
  {
    slug: 'overview', category: 'Start here', title: 'Welcome to ApexVoid',
    description: 'Understand the Enterprise platform, application types, and where to begin.',
    minutes: 4, keywords: ['overview', 'architecture', 'introduction', 'crm', 'erp'],
    sections: [
      { id: 'what-is-apexvoid', title: 'What is ApexVoid Enterprise?', blocks: [
        p('ApexVoid Enterprise is the shared business platform for users, organizations, workspaces, access control, application registration, and a consistent application shell. It runs a Go API and React web client backed by PostgreSQL.'),
        p('ApexVoid supports built-in CRM/ERP modules and separately deployed external applications. An external app keeps its own service and business logic; Enterprise owns its identity, integration lifecycle and workspace access.'),
        table(['Component', 'Responsibility'], ['Enterprise API', 'Session authentication, workspace context, permission decisions, app registry and gateway'], ['Enterprise UI', 'Shared navigation, app discovery, workspace management and documentation'], ['Application service', 'Its own features, REST handlers, SQL migrations and domain logic'], ['PostgreSQL', 'Shared server infrastructure with independently provisioned databases and schemas for external apps']),
      ] },
      { id: 'choose-model', title: 'Choose your application model', blocks: [
        bullets('Built-in modules: compiled into the Enterprise Go runtime and React application; register explicitly at build time.', 'External applications: separate repositories and Docker deployments; discovered via signed manifests, enrolled and linked through the gateway.', 'Do not introduce business-specific code into Enterprise merely to register an external app.'),
        note('Recommended for new products', 'Build new independent business products (such as ApexVoid Café) as external applications. Use compiled-in modules for capabilities that intentionally ship as part of the Enterprise runtime.', 'success'),
      ] },
      { id: 'learning-path', title: 'Your learning path', blocks: [
        steps('Launch Enterprise locally and confirm both services are healthy.', 'Create a standalone Go application with an enrollment manifest, permission catalog and migration bundle.', 'Join the shared Docker networks and allowlist its service alias.', 'Register via Applications → Register application, review operations and enable workspaces.', 'Validate gateway access and per-operation RBAC; then plan production deployment.'),
        note('Scope', 'This documentation describes the capabilities implemented in the current repository. Proposed microfrontend composition and custom standalone domains are not automatically provided by the existing registration flow.'),
      ] },
    ], related: ['local-setup', 'first-application', 'compiled-modules'],
  },
  {
    slug: 'local-setup', category: 'Start here', title: 'Install and run Enterprise',
    description: 'Prerequisites, Docker Compose, first login, healthy startup and development URLs.',
    minutes: 7, keywords: ['install', 'docker', 'compose', 'getting started', 'localhost', 'environment'],
    sections: [
      { id: 'requirements', title: 'Requirements', blocks: [
        bullets('Docker Engine and Docker Compose v2.', 'Go 1.25 or newer, Node.js 22 or newer and npm for local development and test commands.', 'Permission to create the two Docker bridge networks used by Enterprise and external services.', 'A non-production environment for trying enrollment and database provisioning.'),
      ] },
      { id: 'start', title: 'Start the development stack', blocks: [
        snippet('In the apexvoid-enterprise repository', 'bash',
          'cp .env.example .env',
          '# Replace example integration secrets and provisioning credentials.',
          'make compose-up',
          'docker compose ps'),
        p('The Makefile creates the apexvoid-apps and apexvoid-data networks when missing, then builds and starts PostgreSQL, the Go backend and the Vite development frontend. Do not use compose-down to preserve local data: the Makefile target runs docker compose down -v.'),
        table(['Endpoint', 'Development address'], ['Enterprise web', 'http://localhost:8386'], ['Go backend', 'http://localhost:6868'], ['Liveness', 'GET http://localhost:6868/health'], ['Readiness', 'GET http://localhost:6868/ready']),
      ] },
      { id: 'first-login', title: 'First login and workspace', blocks: [
        steps('Open the Enterprise frontend and sign in.', 'On a fresh development database, use the documented bootstrap admin/admin credential only if the environment is development.', 'Complete the forced password-change flow and initial organization setup.', 'Select a workspace before registering or opening workspace-scoped apps.'),
        note('Production credentials', 'Production rejects the default bootstrap admin password. Set unique credentials, HTTPS origin and secure cookie settings before deploying.', 'warning'),
      ] },
      { id: 'environment', title: 'Integration configuration', blocks: [
        snippet('Required environment keys for external enrollment', 'dotenv',
          'DATABASE_PROVISIONER_PASSWORD=<unique-provisioner-password>',
          'DATABASE_PROVISIONING_URL=postgres://apexvoid_provisioner:<password>@postgres:5432/postgres?sslmode=disable',
          'DATABASE_PROVISIONING_KEY=<stable-random-secret-32-characters-or-more>',
          'INTEGRATIONS_ASSERTION_SECRET=<unique-random-secret-32-characters-or-more>',
          'INTEGRATIONS_ALLOWED_SERVICE_HOSTS=cafe',
          'APEXVOID_APPS_NETWORK=apexvoid-apps',
          'APEXVOID_DATA_NETWORK=apexvoid-data'),
        p('Use the postgres Docker service hostname inside the provisioning URL, never localhost. Values in .env are passed to containers only when Compose explicitly maps them; the development Compose file already maps the integration settings.'),
        note('Existing PostgreSQL volume', 'The restricted apexvoid_provisioner role is initialized only on first creation of a PostgreSQL volume. Existing volumes need a DBA-managed provisioner role and appropriate grants; do not delete production volumes to rerun initialization.', 'warning'),
      ] },
      { id: 'verify', title: 'Verify the running services', blocks: [
        snippet('Health checks', 'bash', 'curl --fail http://localhost:6868/health', 'curl --fail http://localhost:6868/ready', 'docker compose logs --tail=100 backend'),
        p('Healthy backend and readiness endpoints are the prerequisite for reliable application discovery. For service registration, also verify the external container can be resolved as a Docker DNS name from the backend network.'),
      ] },
    ], related: ['docker-and-database', 'first-application', 'troubleshooting'],
  },
  {
    slug: 'first-application', category: 'Build applications', title: 'Build your first external app',
    description: 'A complete developer-to-administrator walkthrough using ApexVoid Café.',
    minutes: 11, keywords: ['first application', 'cafe', 'register', 'tutorial', 'wizard', 'setup'],
    sections: [
      { id: 'contract', title: '1. Plan the application contract', blocks: [
        p('Start in a separate repository, for example apexvoid-cafe. Its Go backend and React frontend stay outside Enterprise. Define a stable lowercase application ID, service identity, version, API contract, workspace permissions and SQL migration bundle.'),
        table(['Field', 'Café example'], ['Application ID', 'cafe'], ['Service URL', 'http://cafe:8090 (Docker DNS address)'], ['Manifest path', '/.well-known/apexvoid/manifest.json'], ['Frontend entry', '/apps/cafe'], ['Application API gateway', '/api/apps/cafe'], ['Permission', 'cafe.order.read'], ['Database / schema', 'apexvoid_cafe / cafe']),
      ] },
      { id: 'implement', title: '2. Implement the bootstrap endpoints', blocks: [
        steps('Expose GET /health, returning a healthy response only when the application is ready.', 'Expose GET /.well-known/apexvoid/manifest.json, returning exact JSON bytes and an HMAC-SHA256 signature header derived from the one-time code.', 'Expose the versioned SQL migration files declared in that signed manifest.', 'Expose POST /.well-known/apexvoid/enroll, decrypt and persist the enrollment payload, and return the challenge acknowledgement.', 'Persist the one-time code consumption status across container restarts.'),
        note('Secret handling', 'Generate an unpredictable enrollment code of at least 32 characters. Log the initial service URL, manifest URL, and one-time code only for the authorized operator; never print database passwords or permanent service credentials.', 'warning'),
      ] },
      { id: 'docker', title: '3. Connect Docker networking', blocks: [
        snippet('Service Compose network wiring', 'yaml',
          'services:',
          '  cafe:',
          '    build: .',
          '    networks:',
          '      apexvoid-apps:',
          '        aliases: [cafe]',
          '      apexvoid-data:',
          '',
          'networks:',
          '  apexvoid-apps:',
          '    external: true',
          '    name: apexvoid-apps',
          '  apexvoid-data:',
          '    external: true',
          '    name: apexvoid-data'),
        p('Enterprise uses the apexvoid-apps bridge for service-to-service calls. External applications should not run a second PostgreSQL server for their provisioned production data; the enrollment procedure provisions an isolated database/schema on the existing server.'),
      ] },
      { id: 'register', title: '4. Register from Enterprise', blocks: [
        steps('Open Applications → Register application (requires core.application.manage).', 'Paste the Docker service URL and the one-time enrollment code.', 'Choose Discover application; review the signed manifest, version, API contract, permissions and migration checksums.', 'Choose target workspaces and explicitly approve application registration, permissions, database, schema and SQL migrations.', 'Select Approve & activate. Enterprise provisions the isolated database, sends encrypted credentials, verifies acknowledgement and activates the app after health checks.'),
        note('No manual SQL copy-paste', 'The administrator reviews the SQL migration list and approves it. Enterprise applies the authenticated, checksummed bundle using the restricted application login; it does not execute arbitrary scripts from an unauthenticated URL.'),
      ] },
      { id: 'verify', title: '5. Verify after activation', blocks: [
        steps('Confirm ApexVoid Café appears in Applications and in authorized workspace navigation.', 'Open its frontend entry route and check API responses through the Enterprise gateway.', 'Assign cafe.order.read to a workspace role and verify both allowed and denied operations.', 'Disable the app for another workspace and verify access is denied.', 'Restart containers and confirm enrollment state and app data survive.'),
        note('Gateway launch', 'External applications now open as top-level browser pages through the authenticated Enterprise gateway at /apps/{application-id}. They are not embedded in an iframe, and the chosen workspace is retained through authorization.', 'success'),
      ] },
    ], related: ['manifest-and-enrollment', 'security-and-rbac', 'frontend-and-routing'],
  },
  {
    slug: 'manifest-and-enrollment', category: 'Build applications', title: 'Manifest and secure enrollment',
    description: 'Versioned metadata, signed discovery, reviewed SQL migrations and encrypted credential handoff.',
    minutes: 13, keywords: ['manifest', 'hmac', 'enroll', 'migration', 'checksum', 'signature', 'secret'],
    sections: [
      { id: 'manifest-example', title: 'The v1 application manifest', blocks: [
        p('The manifest is the application-owned contract that Enterprise discovers. Identifiers, API version, frontend routes, permissions, database ownership, and SQL checksums are declared by the service—not manually invented by an administrator.'),
        snippet('Illustrative manifest.json — replace the checksum', 'json',
          '{',
          '  "manifest_version": "v1",',
          '  "application": {',
          '    "id": "cafe", "display_name": "ApexVoid Café",',
          '    "description": "Coffee and photo booth booking",',
          '    "version": "0.1.0", "api_contract_version": "v1"',
          '  },',
          '  "service": {',
          '    "identity": "cafe-service", "health_path": "/health",',
          '    "enrollment_path": "/.well-known/apexvoid/enroll",',
          '    "frontend_route": "/apps/cafe", "api_route": "/api"',
          '  },',
          '  "database": {',
          '    "name": "apexvoid_cafe", "schema": "cafe",',
          '    "role": "apexvoid_cafe", "migration_bundle_version": "0.1.0"',
          '  },',
          '  "permissions": [{',
          '    "name": "cafe.order.read", "display_name": "View orders",',
          '    "description": "View café orders", "scope": "workspace"',
          '  }],',
          '  "access": { "match": "all", "permissions": ["cafe.order.read"] },',
          '  "migrations": [{',
          '    "version": 1,',
          '    "path": "/.well-known/apexvoid/migrations/001-init.sql",',
          '    "sha256": "<SHA-256-OF-EXACT-SQL-BYTES>"',
          '  }]',
          '}'),
        note('Example, not a deployable manifest', 'The example checksum is a placeholder. Publish the actual 64-character SHA-256 hex of each exact SQL file before signing the real manifest.', 'warning'),
      ] },
      { id: 'signing', title: 'Signed discovery and one-time code', blocks: [
        p('Discovery GET does not send the enrollment code to the service. The service signs the exact manifest response bytes using the enrollment code as an HMAC key, with the domain prefix apexvoid-manifest-v1 followed by a newline. Enterprise validates the signature before trusting metadata.'),
        snippet('Use the public Go integration helper', 'go',
          'signature, err := integration.SignManifest(enrollmentCode, manifestJSON)',
          'if err != nil { /* reject invalid enrollment code */ }',
          '// Send unchanged manifestJSON response bytes.',
          'w.Header().Set("X-ApexVoid-Manifest-Signature", signature)'),
        p('The service should disable redirects and avoid intermediary transforms of signed JSON bytes. Enrollment codes must be randomly generated and persisted as single-use state.'),
      ] },
      { id: 'migrations', title: 'SQL migration ownership', blocks: [
        bullets('Each listed SQL path is fetched from the same approved service origin.', 'Each migration includes a version and SHA-256 checksum anchored in the authenticated manifest.', 'Enterprise creates an isolated database and restricted login, then executes approved migrations with that login under transaction locks and statement timeouts.', 'Do not rely on SQL text screening alone; the role privileges, manifest integrity, code review and explicit administrator approval are the security boundary.'),
      ] },
      { id: 'handoff', title: 'Encrypted enrollment handoff', blocks: [
        p('After migration approval, Enterprise encrypts application credentials into an AES-256-GCM envelope and POSTs them to the manifest enrollment path. The raw one-time code is never sent as an HTTP request credential. The service decrypts the payload with integration.OpenEnrollment, validates identity, persists secrets safely and marks the code consumed.'),
        snippet('Enrollment envelope shape', 'json',
          '{',
          '  "version": "v1",',
          '  "nonce": "<base64url-nonce>",',
          '  "ciphertext": "<base64url-ciphertext>"',
          '}'),
        p('Enterprise sends X-ApexVoid-Enrollment-Challenge. The service proves it saved the permanent credential by computing integration.SignEnrollmentAcknowledgement(serviceCredential, challenge) and returning X-ApexVoid-Enrollment-Ack. Acknowledgement must happen only after persistence succeeds.'),
        note('Transport security', 'The protocol protects secrets within the trusted administrator-controlled Docker network. For traffic crossing untrusted networks, configure TLS in addition to the enrollment protocol.', 'warning'),
      ] },
      { id: 'retry', title: 'Retries and updates', blocks: [
        p('An incomplete installation is persisted and may be retried without adopting unclaimed databases or rotating credentials unexpectedly. Registration, schema identity and migration checksums are immutable ownership boundaries. New versions require a challenge-signed upgrade preview and explicit administrator approval. New permissions and append-only SQL migrations are not installed automatically.'),
        steps('Deploy the independent application with a backward-compatible release and a challenge-signed manifest response.', 'Open Enterprise → Applications → Check upgrade to verify and preview its manifest, new permissions and migrations.', 'Review the exact manifest SHA-256 fingerprint and pinned SQL paths/checksums.', 'Approve both permission and migration changes to apply them under the existing restricted application database role.', 'Verify permissions and new database tables, then assign new permissions through workspace roles as needed.'),
        note('Upgradeable contract', 'The application ID, original database, gateway routes and existing permission scopes cannot change through this additive workflow. A changed or unsigned manifest must not be approved.', 'warning'),
      ] },
    ], related: ['first-application', 'docker-and-database', 'operations-and-release'],
  },
  {
    slug: 'docker-and-database', category: 'Platform architecture', title: 'Docker and database isolation',
    description: 'How independently deployed apps reach Enterprise and receive isolated PostgreSQL ownership.',
    minutes: 8, keywords: ['postgres', 'database', 'schema', 'docker networking', 'provisioner', 'compose', 'volumes'],
    sections: [
      { id: 'network-boundaries', title: 'Two shared Docker networks', blocks: [
        table(['Network', 'Purpose'], ['apexvoid-apps', 'Private Docker DNS connectivity between the Enterprise backend and registered app services'], ['apexvoid-data', 'PostgreSQL connectivity for approved services and the Enterprise backend']),
        p('The Enterprise backend joins both networks. An external app backend joins apexvoid-apps with the allowlisted alias and apexvoid-data if its database connection needs it. Docker Compose projects can use shared external networks without sharing deployment lifecycles.'),
        snippet('Inspect the shared networks', 'bash', 'docker network inspect apexvoid-apps', 'docker network inspect apexvoid-data', 'docker compose ps'),
      ] },
      { id: 'database-boundaries', title: 'Provisioned database, schema and role', blocks: [
        p('ApexVoid external enrollment uses one PostgreSQL server infrastructure but a dedicated logical database, schema and least-privilege login for each enrolled application. The canonical names derive from the application ID. This is stronger isolation than merely putting all applications into the Enterprise database schema.'),
        table(['Resource', 'Example for cafe'], ['PostgreSQL server', 'Enterprise postgres container'], ['Application database', 'apexvoid_cafe'], ['Application schema', 'cafe'], ['Application role', 'apexvoid_cafe'], ['Enterprise runtime database', 'apexvoid (separate runtime store)']),
        note('No additional Postgres container', 'Café does not need to bootstrap a separate PostgreSQL service for the database Enterprise provisions. Persistent application data remains its responsibility, and the app must connect using its provisioned restricted role.', 'success'),
      ] },
      { id: 'bootstrap-roles', title: 'Provisioning credentials and setup', blocks: [
        bullets('DATABASE_PROVISIONING_URL must point to a dedicated administrator/provisioner identity at postgres:5432; it must not reuse the application login.', 'DATABASE_PROVISIONING_KEY encrypts stored application credentials. Keep it stable across restarts and back it up securely.', 'The docker/postgres/init/10-provisioner.sh hook only executes on a fresh PostgreSQL volume.', 'Registration refuses to adopt a pre-existing unclaimed database or role; do not manually pre-create an app database to bypass enrollment.'),
      ] },
      { id: 'migration-safety', title: 'Migration and backup rules', blocks: [
        p('Each application owns its versioned SQL migrations, and Enterprise tracks applied versions and checksums. Revisions to already-applied migration bytes should be rejected; publish a new version instead. Backup and restore should account for every provisioned app database as well as Enterprise metadata and application-managed assets.'),
        note('Operations note', 'The Enterprise operations guide documents backing up its main database and attachments. That is not a complete backup of independently provisioned application databases; configure an explicit backup policy for each application.', 'warning'),
      ] },
    ], related: ['local-setup', 'manifest-and-enrollment', 'operations-and-release'],
  },
  {
    slug: 'security-and-rbac', category: 'Platform architecture', title: 'Authentication, workspace and RBAC',
    description: 'Session boundaries, service identity, gateway assertions and permission introspection.',
    minutes: 9, keywords: ['rbac', 'authentication', 'workspace', 'permission', 'identity assertion', 'authorization', 'api'],
    sections: [
      { id: 'ownership', title: 'What Enterprise owns', blocks: [
        bullets('User sessions and login; access and refresh credentials are held in HttpOnly cookies.', 'Organizations, workspaces and the current X-ApexVoid-Workspace context.', 'Permission definitions and platform/workspace role grants.', 'External application registration, allowed service identities and workspace enablement.'),
        p('An application declares its own namespaced permissions, such as cafe.order.read. Enterprise validates ownership of the prefix; those permissions can be granted by workspace administrators through role assignments.'),
      ] },
      { id: 'gateway', title: 'Trusted gateway request flow', blocks: [
        steps('The browser requests /apps/cafe or /api/apps/cafe on the Enterprise gateway.', 'Enterprise verifies the authenticated session, active workspace, application availability and entry policy.', 'The gateway removes cookies, Authorization and any client-supplied identity headers before proxying.', 'It attaches a short-lived encrypted X-ApexVoid-Identity-Assertion for the target service.', 'The service authenticates itself to Enterprise and introspects the assertion for each protected operation.'),
        note('No browser token forwarding', 'Never hand a user access token, refresh token or raw session cookie to an external application. The encrypted assertion is scoped to one application and expires after roughly a minute.', 'warning'),
      ] },
      { id: 'introspection', title: 'Authorize operations, not just menu entries', blocks: [
        snippet('Service-to-platform authorization request', 'http',
          'POST /api/v1/integrations/v1/session/introspect',
          'X-ApexVoid-Application-ID: cafe',
          'X-ApexVoid-Service-Credential: <service-secret>',
          'Content-Type: application/json',
          '',
          '{',
          '  "identity_assertion": "<gateway-assertion>",',
          '  "permission": "cafe.order.read"',
          '}'),
        p('The introspection response is limited to user ID, workspace ID, checked permission and an allowed decision. A service must deny when introspection fails or returns allowed=false. Application-entry visibility does not authorize individual read/write business operations.'),
      ] },
      { id: 'sdk', title: 'Go client integration', blocks: [
        snippet('Public Go integration client', 'go',
          'client, err := integration.NewClient(integration.Config{',
          '  PlatformURL: "http://backend:6868",',
          '  ApplicationID: "cafe",',
          '  ServiceCredential: os.Getenv("APEXVOID_SERVICE_CREDENTIAL"),',
          '})',
          'if err != nil { /* fail startup safely */ }',
          '',
          'assertion, err := integration.IdentityAssertionFromRequest(request)',
          'if err != nil { /* deny request */ }',
          'decision, err := client.Introspect(request.Context(), assertion, "cafe.order.read")',
          'if err != nil || !decision.Allowed { /* deny request */ }'),
        note('SDK setup', 'The integration helper lives in the apexvoid-enterprise Go module under integration/. Import the package into the external Go repository using the version of the Enterprise module you deploy; do not copy private platform internals.'),
      ] },
      { id: 'workspace', title: 'Workspace enablement is separate', blocks: [
        p('A successfully enrolled application is not automatically available to every workspace. Administrators select workspaces during installation and can manage availability afterwards. A disabled app is denied at the gateway and service-introspection boundaries.'),
      ] },
    ], related: ['first-application', 'frontend-and-routing', 'troubleshooting'],
  },
  {
    slug: 'frontend-and-routing', category: 'Platform architecture', title: 'Frontend delivery and routing',
    description: 'Navigate external apps through same-origin gateway URLs without an iframe.',
    minutes: 7, keywords: ['frontend', 'iframe', 'firefox', 'domain', 'routing', 'gateway', 'react', 'view'],
    sections: [
      { id: 'routes', title: 'Registered browser and API paths', blocks: [
        table(['Route', 'Purpose'], ['/apps/{application-id}', 'Enterprise-validated external application entry'], ['/api/apps/{application-id}', 'Authenticated proxy for external business HTTP APIs'], ['/applications', 'Enterprise application discovery, installation and management'], ['/docs', 'Built-in developer and operator documentation']),
        p('External applications declare their frontend_route and api_route in the signed manifest. Enterprise uses its gateway for authentication, RBAC and workspace validation before forwarding requests to the allowlisted service.'),
      ] },
      { id: 'current-view', title: 'Current application viewer behavior', blocks: [
        p('Enterprise now serves external applications as top-level, same-origin gateway documents at /apps/{application-id}. Both Vite and production Nginx forward /apps/ requests to the authenticated Go gateway. This is not yet a native microfrontend loaded within the shared React shell.'),
        note('No cross-origin iframe', 'The viewer no longer embeds localhost:6868 in localhost:8386. Use the Applications launcher or navigation link to open the same-origin gateway URL as a full page.', 'success'),
        p('Keep frame protections enabled. External frontend code is still trusted same-origin application code and must be reviewed. If app opening fails, inspect the gateway, workspace selection and app availability instead of weakening frame headers.'),
      ] },
      { id: 'future-composition', title: 'Standalone versus integrated UI', blocks: [
        bullets('Standalone deployment is already supported at the service level: each app can have its own Go/React deployment and Docker lifecycle.', 'The native Enterprise sidebar can discover authorized external applications without rebuilding the Enterprise image.', 'A shared-shell microfrontend contract and automatic external domain/SSO launch are architectural directions, not guarantees of the current frontend implementation.'),
        p('When implementing a future integrated frontend mode, define an explicit frontend registration contract, safe asset delivery, version compatibility and route ownership. Avoid treating a reverse proxy alone as a microfrontend runtime.'),
      ] },
    ], related: ['first-application', 'security-and-rbac', 'troubleshooting'],
  },
  {
    slug: 'compiled-modules', category: 'Build applications', title: 'Built-in module development',
    description: 'Use the existing Go/React module registries when a feature ships with Enterprise.',
    minutes: 8, keywords: ['module', 'compiled', 'go', 'react', 'frontend registry', 'backend registry'],
    sections: [
      { id: 'when', title: 'When should a module be compiled in?', blocks: [
        p('Built-in modules are suited to foundation capabilities and intentionally bundled Contacts, CRM and ERP features. They are compiled into the Enterprise runtime and React app. They are not dynamically loaded scripts or installable Docker applications.'),
      ] },
      { id: 'backend', title: 'Backend module checklist', blocks: [
        steps('Create internal/modules/<module>/ with domain, application, infrastructure/postgres, transport/http and optional api packages as needed.', 'Implement module.Module with a stable lowercase name, semantic version and explicit dependencies.', 'Register entities, permissions, capabilities, events and descriptor during Register; return validation errors.', 'Register HTTP handlers through module.RouteRegistry with precise authentication and workspace permissions.', 'Own SQL migrations and transaction boundaries; wire dependencies in internal/app/bootstrap.go.'),
        note('Boundary rule', 'Avoid global registries, module init side effects and direct imports of unrelated modules’ private infrastructure. Use narrow APIs and declared dependencies.'),
      ] },
      { id: 'frontend', title: 'Frontend module checklist', blocks: [
        steps('Create a React/TypeScript frontend module under web/src/modules/<module>/.', 'Export an AppModule with routes and permission-aware navigation.', 'If the module is also an application, declare application.id, entryRoute, navigationID and apiContractVersion.', 'Register it in web/src/app/bootstrap/modules.ts and match all frontend contract values to the backend descriptor.', 'Add tests for routing, permission visibility and empty/error states.'),
        p('ApexVoid never executes JavaScript delivered in framework metadata. Compiled module bundles are part of the reviewed Enterprise web build.'),
      ] },
    ], related: ['overview', 'first-application', 'security-and-rbac'],
  },
  {
    slug: 'operations-and-release', category: 'Run & maintain', title: 'Deployment and operations',
    description: 'Production Compose deployment, secrets, service health, upgrades and backups.',
    minutes: 9, keywords: ['production', 'deploy', 'upgrade', 'backup', 'release', 'logs', 'health'],
    sections: [
      { id: 'production', title: 'Production configuration', blocks: [
        p('Enterprise production serves built React assets through Nginx and runs its Go API separately. It does not use Vite. A trusted HTTPS ingress must terminate TLS in front of its loopback-bound frontend listener.'),
        snippet('Production Docker Compose', 'bash',
          'cp .env.production.example .env.production',
          '# Replace every placeholder with deployment-specific secrets.',
          'docker compose --env-file .env.production -f docker-compose.production.yml up -d --build'),
        bullets('Set the HTTPS APEXVOID_PUBLIC_ORIGIN and AUTH_COOKIE_SECURE=true.', 'Set non-default administrator bootstrap credentials.', 'Use a dedicated DATABASE_PROVISIONING_URL, stable DATABASE_PROVISIONING_KEY, unique INTEGRATIONS_ASSERTION_SECRET and exact allowlisted service hostnames.', 'Persist and protect PostgreSQL volumes, application-owned databases and attachment data.'),
      ] },
      { id: 'health', title: 'Health and diagnostics', blocks: [
        snippet('Container and readiness checks', 'bash',
          'docker compose --env-file .env.production -f docker-compose.production.yml ps',
          'docker compose --env-file .env.production -f docker-compose.production.yml logs --tail=100 backend',
          'curl --fail https://<your-enterprise-origin>/ready'),
        p('The health endpoint measures process liveness; readiness checks backend dependencies. Standard application API errors include a safe request ID, not underlying secrets or raw database traces.'),
      ] },
      { id: 'lifecycle', title: 'Deployment and update lifecycle', blocks: [
        bullets('Deployment of an independent external app is managed in that app repository; Enterprise does not restart its Docker container.', 'After a compatible app release is deployed, Enterprise verifies its challenge-signed manifest. New permissions and SQL migrations are applied only after explicit administrator review and approval.', 'Use Applications → Check upgrade to preview added permissions, new migration paths and pinned SHA-256 checksums; approve the exact manifest fingerprint before installation.', 'Revoked or retired services lose access. Permission tombstones prevent silent reuse of old role grants.'),
      ] },
      { id: 'backups', title: 'Backup and recovery', blocks: [
        p('For Enterprise, back up the main PostgreSQL database and attachment volume together after quiescing writes. Also back up every externally provisioned application database, its persistent secrets and any app-specific assets. A backup of apexvoid alone does not cover apexvoid_cafe.'),
        note('Production safety', 'Never run restore commands against a live production database without a validated recovery plan. Use an isolated restore drill, a distinct Compose project name and non-conflicting ports.', 'warning'),
      ] },
      { id: 'tests', title: 'Recommended verification commands', blocks: [
        snippet('Repository quality checks', 'bash', 'make test', 'make lint', 'make build', 'make test-integration'),
        p('The integration suite needs an accessible PostgreSQL test environment. The external service fixture can also be exercised with make test-external-docker when Docker is available.'),
      ] },
    ], related: ['docker-and-database', 'troubleshooting', 'local-setup'],
  },
  {
    slug: 'troubleshooting', category: 'Run & maintain', title: 'Troubleshooting',
    description: 'Diagnose manifest discovery, provisioning, frame errors, RBAC and failed installations.',
    minutes: 8, keywords: ['error', 'troubleshoot', 'fix', 'firefox', '403', 'health', 'dns', 'failed'],
    sections: [
      { id: 'discover-fails', title: 'Application discovery fails', blocks: [
        steps('Check that the application container and /health endpoint are available from the Enterprise backend container.', 'Confirm the service URL uses a Docker DNS name with an explicit port, not localhost or an IP literal.', 'Match the hostname against INTEGRATIONS_ALLOWED_SERVICE_HOSTS.', 'Verify the one-time code has at least 32 characters, is unused, and correctly signs the exact manifest response bytes.', 'Confirm that the signed manifest contract is v1 and that redirects are not involved.'),
      ] },
      { id: 'provisioning-fails', title: 'Database or migration approval fails', blocks: [
        bullets('DATABASE_PROVISIONING_URL must resolve postgres inside Docker and authenticate as a dedicated provisioner.', 'An existing Postgres volume does not rerun init scripts: a DBA must provision the restricted role.', 'DATABASE_PROVISIONING_KEY must remain stable across attempts; an incorrect key breaks encrypted enrollment state.', 'Migration paths must be same-origin and SHA-256 checksums must match the actual SQL response bytes.', 'A pre-existing unclaimed database or role cannot be silently adopted.'),
      ] },
      { id: 'frame-denied', title: 'Firefox cannot open the application view', blocks: [
        p('An earlier Enterprise viewer embedded the API origin (localhost:6868) into the web origin (localhost:8386), which Firefox correctly blocked. Phase 1 removes that iframe. External apps now open through the top-level /apps/{application-id} gateway URL.'),
        steps('Open the application with the Applications launcher or sidebar link; it should open /apps/{application-id} in a new tab.', 'Confirm the authenticated gateway returns the app document and its assets without cross-origin framing.', 'Verify the selected workspace and active membership; then check the app is enabled and the user has its entry permission.', 'Keep X-Frame-Options and frame-ancestors protection enabled, and check Nginx/Vite /apps/ proxy routing if the page fails to load.'),
        note('Separate concerns', 'A successful enrollment does not guarantee application health or correct frontend delivery. Diagnose the gateway, selected workspace, authorized service and browser asset responses separately.', 'info'),
      ] },
      { id: 'rbac-denied', title: 'Permission denied or missing navigation', blocks: [
        steps('Confirm the application is active and enabled in the selected workspace.', 'Check that the user role has the exact app-owned permission declared in the signed manifest.', 'Check entry policy separately from operation-level permissions.', 'Verify the service sends its own application ID and permanent credential to introspection.', 'Reject a missing, wrong-audience or expired identity assertion instead of bypassing RBAC.'),
      ] },
      { id: 'restart', title: 'Service loses enrollment after restart', blocks: [
        p('The external application must durably store both consumed enrollment state and the permanent service credential. Process-only memory loses those values when Docker restarts; use a secret store or protected persistent mount and recover consistently.'),
      ] },
      { id: 'logs', title: 'Where to investigate next', blocks: [
        snippet('Read container status and logs', 'bash',
          'docker compose ps',
          'docker compose logs --tail=150 backend',
          '# Run inside the external app repository:',
          'docker compose logs --tail=150 cafe'),
        p('Use request IDs from safe API error responses to correlate log entries. Do not paste live credentials, pairing codes, cookies, encrypted assertion headers or database connection passwords into tickets.'),
      ] },
    ], related: ['local-setup', 'frontend-and-routing', 'security-and-rbac'],
  },
]

export const guidesBySlug = new Map(guides.map(guide => [guide.slug, guide]))

export function searchGuides(query: string): DocGuide[] {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return guides
  return guides.filter(guide => [
    guide.title, guide.description, guide.category, ...guide.keywords,
    ...guide.sections.flatMap(section => [
      section.title,
      ...section.blocks.flatMap(block => {
        if (block.type === 'paragraph') return [block.text]
        if (block.type === 'list') return block.items
        if (block.type === 'code') return [block.title, block.value]
        if (block.type === 'callout') return [block.title, block.text]
        return [...block.headings, ...block.rows.flat()]
      }),
    ]),
  ].some(value => value.toLowerCase().includes(normalized)))
}
