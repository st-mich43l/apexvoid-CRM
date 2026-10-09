# External module integration

ApexVoid CRM is the platform of record for users, opaque sessions, workspaces,
and RBAC. An external module is a trusted Docker service contract; it is never
loaded as Go code and it must not connect to the ApexVoid database.

## Registering a service

A platform administrator with `core.application.manage` registers a module at
`POST /api/v1/applications/external`. The request includes the application ID,
display metadata, `vN` API contract, Docker-network service and health
endpoints, optional gateway route, access policy, and namespaced permissions.
Every declared permission must begin with `<application-id>.`; ApexVoid rejects
duplicate IDs, service identities, and permission names, and it never grants a
new permission to a role automatically.

The success response returns a service credential exactly once. Store it in a
container secret. It is hashed at rest, is absent from read APIs, and can be
revoked with `POST /api/v1/applications/external/{application}/credentials/revoke`.

Workspace availability is managed independently of registration:

```text
PUT /api/v1/applications/external/{application}/workspaces/{workspace-id}
{"enabled": true}
```

An application is enabled by default unless an explicit workspace record
disables it. Disabled applications are omitted from discovery and their service
introspection requests fail closed.

## Service-to-service contract

Services call the versioned integration API with:

```text
X-ApexVoid-Application-ID: reports
X-ApexVoid-Service-Credential: <credential returned at registration>
```

To validate a session and receive one authorization decision, call:

```text
POST /api/v1/integrations/v1/session/introspect
{
  "access_token": "<opaque ApexVoid access token>",
  "workspace_id": "<workspace UUID>",
  "permission": "reports.report.read"
}
```

The response includes only `user_id`, `workspace_id`, the requested permission,
and `allowed`. Revoked/expired sessions, initial-password-change sessions,
missing membership, disabled modules, unknown permissions, and invalid service
credentials are rejected. Refresh tokens must never be sent to a module.

Availability can be checked by the service itself at
`GET /api/v1/integrations/v1/applications/{application}/availability?workspace_id=<UUID>`.

## Docker configuration

Run services on the existing ApexVoid Docker network:

```yaml
environment:
  APEXVOID_URL: http://backend:6868
  APEXVOID_APPLICATION_ID: reports
  APEXVOID_SERVICE_CREDENTIAL: ${REPORTS_APEXVOID_SERVICE_CREDENTIAL}
```

Use short request timeouts. Health is read from the registered health endpoint;
outages never remove permission catalog entries or role grants.

## Frontend gateway

An external frontend route must be `/apps/{application-id}` or a descendant.
ApexVoid proxies it only after user authentication, workspace resolution, and
enablement checks. The gateway strips browser cookies and Authorization headers
before forwarding, so an external service must use introspection. Registration
metadata cannot load arbitrary JavaScript into the ApexVoid frontend.

`test/fixtures/external-module` is a health-only Docker fixture for integration
tests. It is not a product application and has no database access.
