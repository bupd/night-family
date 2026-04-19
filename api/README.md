# API spec

The canonical night-family API spec and JSON Schemas live at:

- `internal/server/apiassets/openapi.yaml`
- `internal/server/apiassets/schemas/*.json`

They sit there (rather than under `api/`) because Go's `//go:embed`
directive cannot traverse `..`, and we want the spec embedded directly into
the `nfd` binary so there's no runtime filesystem dependency.

## Rules

1. **Breaking changes bump `/api/v1` to `/api/v2`.** Additive changes —
   new endpoints, new optional fields, new enum variants — don't.
2. **Handlers validate against the spec at runtime.** A request that
   doesn't match returns an RFC 9457 `application/problem+json` body.
3. **The spec is the source of truth.** Generated Go types and clients
   live under `internal/api/gen/` (once codegen lands) and are regenerated
   via `task api:gen`. CI asserts the committed output is current.
4. **Every endpoint has a stable `operationId`.** It doubles as the Go
   symbol stem.

## Live endpoints (once `nfd` is running)

- `GET /openapi.yaml` — raw spec
- `GET /openapi.json` — same spec re-encoded as JSON
- `GET /docs` — Swagger UI bound to the live server

## Previewing locally

Any OAS tool works, e.g.:

```
npx @redocly/cli preview-docs internal/server/apiassets/openapi.yaml
```
