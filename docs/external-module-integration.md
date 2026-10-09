# External module integration

ApexVoid CRM is the system of record for users, opaque sessions, workspaces,
and RBAC. An external module is an administrator-approved Docker-network
service. It is never loaded as Go code, never receives database credentials,
and cannot add itself to a role.

## Platform configuration

External registrations are disabled by default. Enable only known Docker DNS
hosts and use a unique, secret assertion key:

```yaml
integrations:
  assertion_secret: "at-least-32-secret-characters-kept-out-of-git"
  allowed_service_hosts: [reports-service]
```

`INTEGRATIONS_ASSERTION_SECRET` and
`INTEGRATIONS_ALLOWED_SERVICE_HOSTS=reports-service,other-service` provide the
same configuration. Registration accepts HTTP Docker-network URLs only, with
an explicit port, no credentials/query/fragment, and a hostname exactly in the
allowlist. Loopback addresses, IP literals, redirects, and arbitrary hosts are
rejected.

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

## Gateway routes and trust boundary

An external frontend route is `/apps/{application-id}` (or a descendant), and
external business API calls use `/api/apps/{application-id}`. Both routes are
authenticated, workspace-scoped, enabled-state checked, and entry-RBAC checked
by Core before proxying. The proxy removes `Cookie`, `Authorization`, and any
client-supplied identity headers; it adds only the encrypted assertion and
does not forward upstream cookies back to the browser.

An external frontend is still trusted same-origin code after an administrator
registers it. Keep the service image and its dependencies under change control,
serve only application assets, and do not treat this gateway as a sandbox. The
standard web gateway sends a restrictive baseline CSP and no-store responses
for proxied external content.

## Docker fixture and verification

`test/fixtures/external-module` is a non-product Docker service used by the
integration suite. It exposes a health endpoint and validates gateway
assertions by calling the real Core introspection endpoint with its own service
credential. The Docker integration command exercises registration, gateway
frontend/API access, RBAC denial, workspace disablement, credential rotation,
revocation, and retirement without any database access from the fixture.
