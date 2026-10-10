# External module integration

ApexVoid CRM is the system of record for users, opaque sessions, workspaces,
and RBAC. An external application is an administrator-approved Docker-network
service. It is never loaded as Go code into Enterprise and cannot add itself to
a platform or workspace role. A manifest-enrolled application receives a
restricted credential for **its own Enterprise-provisioned application database**,
not the Enterprise platform runtime database or unrestricted PostgreSQL access.

## Platform configuration

External registrations use signed enrollment and platform-administrator
approval. Use a unique, secret assertion key:

```yaml
integrations:
  assertion_secret: "at-least-32-secret-characters-kept-out-of-git"
```

`INTEGRATIONS_ASSERTION_SECRET` protects the short-lived gateway identity
assertions. Registration accepts HTTP Docker-network URLs only, with an
explicit port and no credentials/query/fragment. The signed manifest and
single-use enrollment code prove control of the service, while the platform
administrator approves the resulting application record. Loopback addresses,
IP literals, redirects, and malformed hosts are rejected; the approved endpoint
is persisted and reused for later gateway requests and upgrades.

## Registering and retiring a service

A platform administrator with `core.application.manage` registers a module at
`POST /api/v1/applications/external`. IDs and permission names are permanent:
an application's declared permissions must begin with `<application-id>.` and
may not collide with internal, active external, or retired permission names.
The route is exposed only after its database catalog and the live permission
registry have both been created. A failed staging attempt is removed during
startup recovery and is never discoverable.

The success response contains `service_credential` exactly once. Store it as a
container secret; ApexVoid stores only a hash and never returns it in read
responses. Rotate an active credential with:

```text
POST /api/v1/applications/external/{application}/credentials/rotate
```

The replacement secret is returned once and invalidates the old secret
immediately. Revoke is terminal:

```text
POST /api/v1/applications/external/{application}/credentials/revoke
```

Retire with `DELETE /api/v1/applications/external/{application}`. Retirement
disables the application, revokes its credential, removes its live permission
definitions, and keeps permission tombstones so historical role grants cannot
be silently rebound to a replacement service.

Workspace availability is separate from global registration:

```text
PUT /api/v1/applications/external/{application}/workspaces/{workspace-id}
{"enabled": true}
```

Disabled applications are omitted from discovery and denied by both the
gateway and service introspection.

## Service-to-service authorization

Each service call to ApexVoid authenticates the service itself:

```text
X-ApexVoid-Application-ID: reports
X-ApexVoid-Service-Credential: <container secret>
```

The browser never sends an access token to a service. The ApexVoid gateway
forwards a short-lived, encrypted `X-ApexVoid-Identity-Assertion`; a service
must return that assertion to Core to request exactly one permission decision:

```text
POST /api/v1/integrations/v1/session/introspect
{
  "identity_assertion": "<gateway header value>",
  "permission": "reports.report.read"
}
```

The requested permission must be owned by the authenticated application.
Core verifies the assertion audience, session, user, workspace membership,
module availability, and RBAC before returning only `user_id`, `workspace_id`,
`permission`, and `allowed`. Assertions expire after one minute and cannot be
used for another application. Never send access or refresh tokens to a module.

A service can check its own workspace availability at:

```text
GET /api/v1/integrations/v1/applications/{application}/availability?workspace_id=<UUID>
```

## Go client

Go services should import the public client from this repository rather than
copying the protocol. It has no dependency on ApexVoid internals or database
packages:

```go
client, err := integration.NewClient(integration.Config{
    PlatformURL: "http://backend:6868", ApplicationID: "reports",
    ServiceCredential: os.Getenv("APEXVOID_SERVICE_CREDENTIAL"),
})
if err != nil { /* fail service startup safely */ }

assertion, err := integration.IdentityAssertionFromRequest(request)
decision, err := client.Introspect(request.Context(), assertion, "reports.report.read")
if err != nil || !decision.Allowed { /* deny the operation */ }
```

The client uses bounded timeouts and at most two retries for transient gateway
failures. Its typed errors contain status, API error code, and request ID only;
they never retain credentials or assertions. A service must make a separate
introspection request for every protected operation and must not treat entry
authorization as operation authorization.

The only supported integration API contract is `v1`. `version` is the service
release version; `api_contract_version` is the compatible ApexVoid wire
contract. Registration and updates reject unsupported contract versions with
`UNSUPPORTED_CONTRACT`. All standard API failures use the existing safe shape:

```json
{"error":{"code":"FORBIDDEN","message":"…","request_id":"…"}}
```

## Gateway routes and trust boundary

An external frontend route is `/apps/{application-id}` (or a descendant), and
external business API calls use `/api/apps/{application-id}`. Both routes are
authenticated, workspace-scoped, enabled-state checked, and entry-RBAC checked
by Core before proxying. The proxy removes `Cookie`, `Authorization`, and any
client-supplied identity headers; it adds only the encrypted assertion and
does not forward upstream cookies back to the browser.

An external frontend is still trusted same-origin code after an administrator
registers it. Keep the service image and its dependencies under change control,
serve only application assets, and do not treat this gateway or its CSP as a
security sandbox or a separate-origin isolation boundary. The standard web
gateway sends a restrictive baseline CSP and no-store responses for proxied
external content, strips response cookies and redirects, bounds API request
bodies, and uses bounded upstream connection/header timeouts.

## Seamless browser sign-on and session recovery

Enterprise remains the sole login and authorization authority. An external
application must never prompt for the user's Enterprise password or maintain
an independent token store.

- A top-level navigation to `/apps/{application-id}?workspace_id=...` with
  a valid access session opens normally.
- If the access token has expired, **document navigations only** redirect to
  first-party `/auth/continue?return_to=<encoded-local-app-URL>`. That React
  route calls Enterprise's existing /auth/me and scoped /auth/refresh endpoints,
  then returns to the original app and workspace. If the refresh session has
  expired, it sends the user to /login and resumes after successful login.
  Required first-login password changes preserve the return destination too.
- API, JavaScript asset, and other non-document requests are **not** redirected
  to HTML. They still receive structured JSON `401 UNAUTHENTICATED`.

For a long-running app frontend, wrap the gateway fetch client:

1. Only when a gateway request returns `401 UNAUTHENTICATED`, send a same-origin
   `POST /api/v1/auth/refresh` with browser cookies included. Coordinate
   simultaneous refreshes with one shared promise.
2. Retry the original gateway request **at most once** after successful refresh.
   A first gateway 401 was blocked before proxying, so the operation was not
   executed. Preserve the original idempotency key on mutations.
3. If refresh fails, navigate to Enterprise login with a validated same-origin
   `/apps/{application-id}` return path. Never embed credentials or assertions
   in URLs and do not add refresh tokens to localStorage or broader-scope cookies.

The gateway still independently validates the live session, workspace
membership, app availability, and RBAC on every request. The existing gateway
strips browser cookies and Authorization headers before forwarding to the app;
only its short-lived identity assertion reaches the service.

**Workspace note:** the current browser workspace selector is not an
independent security credential; the external app should supply
`X-ApexVoid-Workspace` explicitly on API requests. A per-tab workspace routing
design remains separate follow-up work for simultaneously opening the same app
under different workspaces.

### Manual verification

With an existing Enterprise login and Photo Booth installed:

1. Remove only the access-token cookie, leaving the refresh cookie valid.
   Open `/apps/photobooth?workspace_id=<permitted-workspace-id>` in a new tab:
   it should recover silently and open the application.
2. In an already-open Photo Booth, expire the access token and use a protected
   operation. The API wrapper should refresh and retry once.
3. Remove/invalidate the refresh session too. A direct app launch should
   display Enterprise login and restore the app/workspace after login.
4. Verify that disabled apps, inaccessible workspaces and unauthorized users
   are still denied. JSON API and asset requests should **never** receive
   a login HTML document.

## Docker fixture and verification

`test/fixtures/external-module` is a non-product Docker service used by the
integration suite. It exposes a health endpoint and validates gateway
assertions by calling the real Core introspection endpoint with its own service
credential. The Docker integration command exercises registration, gateway
frontend/API access, RBAC denial, workspace disablement, credential rotation,
revocation, and retirement without any database access from the fixture.

## Reviewed external application upgrades

After an application is enrolled, the administrator can choose **Applications → Check upgrade**.
This does **not** install software or rebuild Docker images. The application
owner first deploys a backward-compatible release to its independently managed
container, then Enterprise previews and applies its application-owned catalog
and migration changes only after approval.

The update manifest must be authenticated with the **current** service
credential. Enterprise sends a fresh, random challenge on the normal manifest
GET using the `X-ApexVoid-Update-Challenge` header. The application returns
the *exact* manifest JSON bytes and:

```text
X-ApexVoid-Update-Signature: sha256=<64 lowercase hex characters>
```

The HMAC-SHA256 key is the **raw 32-byte SHA-256 digest** of the UTF-8 service
credential, not the credential itself and not its hexadecimal representation.
The authenticated message is these bytes concatenated, in order:

```text
apexvoid-update-manifest-v1\n<challenge>\n<exact manifest JSON response bytes>
```

Use `integration.SignUpdateManifest(serviceCredential, challenge, manifestBytes)`
from a compatible ApexVoid SDK release to produce the header. Never include the
service credential, access tokens, enrollment code or platform database
password in an update request or response. The server validates the response
with the currently active credential hash, which also supports credential
rotation. Each update preview is tied to that single signed response and a
short-lived immutable fingerprint.

### Administrator endpoints

```text
POST /api/v1/applications/external/{application}/updates/check
GET  /api/v1/applications/external/{application}/updates/{upgrade-id}
POST /api/v1/applications/external/{application}/updates/{upgrade-id}/approve
```

The preview includes the installed/new release and migration bundle versions,
new permission definitions, new migration versions/paths/checksums, the signed
manifest fingerprint and expiry. An unchanged, signed manifest yields an
`up_to_date` status. A changed manifest becomes a `pending_approval` plan.
For approval, the administrator sends:

```json
{
  "manifest_sha256": "<exact preview fingerprint>",
  "approve_permissions": true,
  "approve_migrations": true
}
```

Both approvals are mandatory even if one category has no additions.
Only platform administrators with `core.application.manage` can perform these
actions, and approval is auditable. The plan is not allowed to change or remove
existing permissions or their scopes, rewrite an earlier SQL migration, rename
the application or its database ownership, or change existing service gateway
routes. Stable application version must increase. Any new migration requires
an increased migration-bundle version.

Enterprise re-verifies a fresh signed response before applying a plan.
SQL is fetched from the original allowed Docker service URL, verified against
the pinned SHA-256 checksums and applied using the **existing restricted app
database role**. No new database is provisioned for an upgrade. Application
schema migrations should be *backward compatible*, since a previously deployed
service version may remain running during or after an unsuccessful review.
Failed reviews can be retried while valid, with existing applied migration
history checked for tampering. A completed review updates application metadata
and new permission definitions without assigning those permissions to roles.

Do **not** repurpose the legacy manual metadata editor as an upgrade API,
reuse the consumed registration code, or attempt unauthenticated update checks.
