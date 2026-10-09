# In-app documentation center

ApexVoid Enterprise includes a read-only documentation area under **Platform → Documentation** at the stable frontend route `/docs`. Each guide has its own URL and section anchors. Guides ship in the frontend Docker image—there is no separate documentation server, API, external CDN, or database migration.

## Source layout

- `web/src/modules/core/pages/documentation/content.ts` defines typed guides, categories, keywords, rich blocks and searchable content.
- `web/src/modules/core/pages/DocumentationPage.tsx` implements search, reading UI, guide navigation, code copy, deep links and responsive contents navigation.
- `web/src/modules/core/index.tsx` registers both documentation routes and the sidebar entry.
- `web/src/modules/core/pages/DocumentationPage.test.tsx` verifies links, search, metadata and integrity.

These UI guides are the approachable learning path. The canonical contract details remain in `docs/building-applications.md`, `docs/external-module-integration.md`, and `docs/operations.md`. If protocol behavior changes, review both the canonical references and every corresponding in-app guide.

## Editing a guide

1. Add or edit a guide in the typed `guides` registry, using a unique stable slug, category, meaningful summary, practical keywords and estimated reading time.
2. Organize each guide into a small number of sections with stable unique anchor IDs; keep the hierarchy suitable for a visible table of contents.
3. Use declarative block types for prose, numbered procedures, bullets, data tables, commands and security notes. Code examples must match the repository's implemented API and SDK signatures.
4. Update `related` with other existing guide slugs only. Do not silently introduce undeployed capabilities such as microfrontend integration, external SSO domains, or automatic application upgrades.
5. Run frontend typecheck, tests, lint and build, and verify both light and dark themes.

## Local checks

```bash
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run lint
npm --prefix web run build
```

Development Docker uses a Vite source bind mount so edits appear without rebuilding the backend. Production bundles the same guide sources in the built frontend image; deploy a new frontend image to publish a guide update.

## Security

Never put real enrollment codes, permanent service credentials, database passwords, cookies or identity assertions in the guide source. Explicitly mark placeholders, and remind readers that current browser frontend delivery uses the external app iframe route subject to browser frame restrictions. Documentation must not encourage bypassing gateway RBAC or disabling security headers.
