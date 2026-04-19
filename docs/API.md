# API

night-family exposes a REST + JSON API described by OpenAPI 3.1
(`api/openapi.yaml`). The spec is the source of truth; server handlers are
validated against it at request time.

Base URL (default, local): `http://127.0.0.1:7337/api/v1`

## Authentication

- **Local (default)**: unix socket; filesystem permissions are the ACL.
- **Network**: bearer token in `Authorization` header. Token is generated at
  daemon install time and stored in `~/.config/night-family/token`.

## Endpoints (v1)

### Health & meta

```
GET  /healthz                    → 200 ok
GET  /readyz                     → 200 ready | 503
GET  /version                    → {version, commit, built_at}
GET  /metrics                    → Prometheus text format
```

### Family

```
GET    /family                   → [FamilyMember]
GET    /family/{name}            → FamilyMember
POST   /family                   ← FamilyMemberInput
PUT    /family/{name}            ← FamilyMemberInput
DELETE /family/{name}
POST   /family/validate          ← FamilyMemberInput  → {ok, errors[]}
```

### Duties

```
GET    /duties                   → [DutyInfo]      (catalog)
GET    /duties/{type}            → DutyInfo
POST   /duties/{type}/plan       ← PlanRequest     → Plan (dry-run)
```

### Runs

```
GET    /runs?member=&duty=&status=&since=  → PagedRuns
GET    /runs/{id}                → Run (full)
GET    /runs/{id}/logs?tail=     → text/event-stream
POST   /runs                     ← RunRequest       → Run (queued)
POST   /runs/{id}/cancel         → Run
```

### Nights (a night = one scheduled window)

```
GET    /nights                   → [NightSummary]
GET    /nights/{id}              → Night
GET    /nights/current           → Night | 404
POST   /nights/trigger           → Night (starts off-schedule night)
POST   /nights/current/stop      → Night
```

### Budget & schedule

```
GET    /budget                   → BudgetSnapshot
GET    /schedule                 → Schedule
PUT    /schedule                 ← ScheduleInput
```

### PRs (index of what we've opened)

```
GET    /prs?state=&since=        → [PR]
GET    /prs/{id}                 → PR
```

### Config

```
GET    /config                   → Config
PUT    /config                   ← ConfigInput
GET    /config/reviewers         → [string]
PUT    /config/reviewers         ← [string]
```

## Schemas (excerpt)

### FamilyMember

```jsonc
{
  "name": "rick",
  "role": "Senior security engineer",
  "system_prompt": "...",
  "duties": [
    {
      "type": "vuln-scan",
      "interval": "24h",
      "priority": "high",
      "args": {"include_dependencies": true}
    }
  ],
  "risk_tolerance": "medium",
  "max_prs_per_night": 2,
  "cost_tier": "high",
  "reviewers": ["coderabbitai", "cubic-dev-ai"],
  "provider": {"name": "claude", "model": "claude-opus-4-7"},
  "created_at": "2026-04-17T22:00:00Z",
  "updated_at": "2026-04-17T22:00:00Z"
}
```

### Run

```jsonc
{
  "id": "run_01H...",
  "night_id": "night_01H...",
  "member": "rick",
  "duty": "vuln-scan",
  "status": "succeeded",     // queued | running | succeeded | failed | cancelled
  "started_at": "...",
  "finished_at": "...",
  "tokens_in": 12450,
  "tokens_out": 3100,
  "branch": "night-family/rick/vuln-scan-a1b2c3",
  "pr_url": "https://github.com/.../pull/42",
  "summary": "Found 1 high, 3 medium...",
  "error": null
}
```

### BudgetSnapshot

```jsonc
{
  "taken_at": "2026-04-17T22:05:00Z",
  "provider": "claude",
  "remaining_tokens_estimate": 82000,
  "window_ends_at": "2026-04-18T00:30:00Z",
  "reserved_for_tonight": 60000
}
```

### Night

```jsonc
{
  "id": "night_01H...",
  "started_at": "2026-04-17T22:00:00Z",
  "finished_at": null,
  "plan": [ /* ordered list of planned runs */ ],
  "runs": [ /* actual runs, in order */ ],
  "summary": null
}
```

## Error model

Problem-Details (RFC 9457) shape:

```jsonc
{
  "type": "https://night-family.dev/errors/validation",
  "title": "Invalid family member",
  "status": 400,
  "detail": "system_prompt must not be empty",
  "instance": "/api/v1/family"
}
```

## Versioning

- Major version in URL (`/api/v1`).
- Breaking changes bump the major; bug fixes do not.
- OpenAPI spec committed under `api/openapi.yaml`; schema files under
  `api/schemas/*.json`.

## HTMX surface

The web UI uses a parallel `/ui/*` surface that returns HTML fragments. These
are **not** part of the public API and may change without notice. They share
the same domain model.
