# Knowledge Silo Report

**Date:** 2026-04-30
**Repos analyzed:** harbor-satellite, harbor-cli, harbor-scanner-trivy

## Executive Summary

This report identifies knowledge silos across three Harbor ecosystem repositories by computing the **bus factor** (number of unique human contributors) for each directory and source file. A bus factor of 1 means only one person has ever committed to that area -- a single point of failure for project continuity.

**Key findings:**

- **harbor-satellite** is heavily concentrated: `bupd` + `Prasanth Baskar` (same person, two git identities) account for ~73% of all commits. Several subsystems (`docker/`, `internal/watcher/`, `internal/crypto/`, `internal/spiffe/`) have a bus factor of 1.
- **harbor-cli** has strong single-owner silos: Patrick Eschenbach solely owns replication, robot accounts, scan-all, and configurations subsystems (~40+ files). Rizul Gupta solely owns scanner, webhook, CVE allowlist, and immutable tag features (~30+ files).
- **harbor-scanner-trivy** is dominated by Daniel Pacak (139 commits, 35% of total). Several test fixture directories and helm templates have a bus factor of 1.

---

## harbor-satellite

**Total unique contributors:** 27 (excluding bots)
**Top contributors by commit count:**

| Author | Commits |
|--------|---------|
| bupd | 592 |
| Prasanth Baskar | 103 |
| Vadim Bauer | 45 |
| meethereum | 27 |
| Roald Brunell | 25 |
| Mehul-Kumar-27 | 22 |
| M Viswanath Sai | 20 |
| Narhari Motivaras | 17 |

### Directories with Bus Factor = 1

| Directory | Sole Author |
|-----------|-------------|
| `docker/` | Prasanth Baskar |
| `docker/e2e/` | Prasanth Baskar |
| `docker/e2e/spiffe/` | Prasanth Baskar |
| `internal/watcher/` | M Viswanath Sai |
| `test/e2e/testconfig/config/core/` | Narhari Motivaras |
| `test/e2e/testconfig/config/jobservice/` | Narhari Motivaras |
| `test/e2e/testconfig/config/registry/` | Narhari Motivaras |
| `test/e2e/testconfig/config/registryctl/` | Narhari Motivaras |
| `website/layouts/docs/` | Prasanth Baskar |
| `website/layouts/landing/` | Prasanth Baskar |

### Directories with Bus Factor = 2

| Directory | Authors |
|-----------|---------|
| `deploy/helm/` | bupd, Prasanth Baskar |
| `internal/crypto/` | bupd, Prasanth Baskar |
| `internal/identity/` | bupd, Prasanth Baskar |
| `internal/secure/` | bupd, Prasanth Baskar |
| `internal/spiffe/` | bupd, Prasanth Baskar |
| `internal/token/` | bupd, Prasanth Baskar |
| `internal/version/` | Mehul-Kumar-27, Prasanth Baskar |
| `ground-control/internal/auth/` | bupd, Prasanth Baskar |
| `ground-control/internal/middleware/` | bupd, Prasanth Baskar |
| `ground-control/internal/spiffe/` | bupd, Prasanth Baskar |
| `ground-control/pkg/crypto/` | bupd, Prasanth Baskar |

> **Note:** `bupd` and `Prasanth Baskar` appear to be the same person. Directories listed as bus factor 2 with only these two identities effectively have a **true bus factor of 1**.

### Single-Author Source Files (selected)

| File | Sole Author |
|------|-------------|
| `internal/watcher/watcher.go` | M Viswanath Sai |
| `internal/server/middleware.go` | Mehul Kumar |
| `internal/state/artifact.go` | Mehul Kumar |
| `internal/scheduler/scheduler_test.go` | Anurag Ojha |
| `ground-control/internal/harborhealth/types.go` | Narhari Motivaras |
| `ground-control/sql/schema/006_config.sql` | M Viswanath Sai |
| `ground-control/sql/schema/007_satellites_config.sql` | M Viswanath Sai |

### Risk Assessment

- **Critical:** `internal/crypto/`, `internal/spiffe/`, `internal/identity/`, `internal/token/` -- security-sensitive code with effective bus factor of 1.
- **High:** `internal/watcher/` -- entire subsystem known only to M Viswanath Sai.
- **Medium:** E2E test infrastructure known only to Narhari Motivaras.

---

## harbor-cli

**Total unique contributors:** 69 (excluding bots)
**Top contributors by commit count:**

| Author | Commits |
|--------|---------|
| bupd | 158 |
| Vadim Bauer | 90 |
| amands98 | 45 |
| NucleoFusion | 42 |
| Prasanth Baskar | 41 |
| Patrick Eschenbach | 34 |
| Rizul Gupta | 28 |
| Tyler Auerbeck | 15 |

### Directories with Bus Factor = 1

| Directory | Sole Author |
|-----------|-------------|
| `doc/cli-config/` | Patrick Eschenbach |
| `doc/cli-encryption/` | Patrick Eschenbach |
| `examples/` (all subdirs) | Patrick Eschenbach |
| `test/helper/` | Patrick Eschenbach |
| `pkg/views/configurations/` | Patrick Eschenbach |
| `pkg/views/scan-all/` | Patrick Eschenbach |
| `pkg/views/cveallowlist/` | Rizul Gupta |
| `pkg/views/immutable/` | Rizul Gupta |
| `pkg/views/password/` | Chayan Das |

### Single-Author Source Files (major clusters)

**Patrick Eschenbach (40+ sole-authored files):**
- Entire `cmd/harbor/root/configurations/` subsystem
- Entire `cmd/harbor/root/replication/` subsystem (8 files)
- Entire `cmd/harbor/root/robot/` subsystem (5 files)
- Entire `cmd/harbor/root/scan_all/` subsystem (6 files)
- `pkg/api/configurations_handler.go`, `pkg/api/robot_handler.go`, `pkg/api/scan_all_handler.go`
- All replication views, robot views, scan-all views
- `test/helper/helpers.go`

**Rizul Gupta (30+ sole-authored files):**
- Entire `cmd/harbor/root/scanner/` subsystem (6 files)
- Entire `cmd/harbor/root/webhook/` subsystem (4 files)
- Entire `cmd/harbor/root/cve/` subsystem
- `cmd/harbor/root/instance/cmd.go`, `cmd/harbor/root/tag/cmd.go`
- All scanner views, webhook views, CVE allowlist views, immutable views

**Nucleo Fusion (8 sole-authored files):**
- `cmd/harbor/root/artifact/scan/` subsystem (3 files)
- `cmd/harbor/root/artifact/tags/` subsystem (4 files)
- `.dagger/archive.go`, `.dagger/checksum.go`, `.dagger/sbom.go`, `.dagger/utils.go`

### Risk Assessment

- **Critical:** Patrick Eschenbach is the sole knowledge holder for replication, robot accounts, configurations, and scan-all -- four major feature areas. If this contributor becomes unavailable, maintaining these features would require significant ramp-up.
- **High:** Rizul Gupta solely owns scanner, webhook, CVE, and immutable tag management.
- **Medium:** Nucleo Fusion solely owns artifact scan/tag commands and most Dagger CI pipeline code.

---

## harbor-scanner-trivy

**Total unique contributors:** 31 (excluding bots)
**Top contributors by commit count:**

| Author | Commits |
|--------|---------|
| Daniel Pacak | 139 |
| chenk | 58 |
| Daniel Jiang | 57 |
| Wang Yan | 9 |
| Teppei Fukuda | 9 |
| Prasanth Baskar | 5 |

### Directories with Bus Factor = 1

| Directory | Sole Author |
|-----------|-------------|
| `docs/` | Daniel Pacak |
| `docs/images/` | Daniel Pacak |
| `test/component/data/` (all subdirs) | Daniel Pacak |
| `test/component/scanner/` | Daniel Pacak |
| `test/integration/api/testdata/` (all subdirs) | Teppei Fukuda |
| `test/integration/persistence/` | Daniel Pacak |

### Directories with Bus Factor = 2

| Directory | Authors |
|-----------|---------|
| `pkg/job/` | Daniel Pacak, Teppei Fukuda |
| `pkg/persistence/` | Daniel Pacak, Teppei Fukuda |
| `pkg/queue/` | Daniel Pacak, Teppei Fukuda |
| `pkg/ext/` | Daniel Pacak, guangwu, Teppei Fukuda |

### Single-Author Source Files

| File | Sole Author |
|------|-------------|
| `helm/harbor-scanner-trivy/templates/secret-tls.yaml` | Daniel Pacak |
| `helm/harbor-scanner-trivy/templates/secret.yaml` | Daniel Pacak |
| `helm/harbor-scanner-trivy/templates/service.yaml` | Daniel Pacak |
| `pkg/http/api/server_test.go` | Daniel Pacak |
| `helm/harbor-scanner-trivy/templates/trivy-ignore-policy-cm.yaml` | Peter Jakobsen |
| `pkg/trivy/errors.go` | Prasanth Baskar |
| `pkg/trivy/target_test.go` | Prasanth Baskar |
| `skaffold.yaml` | Teppei Fukuda |

### Risk Assessment

- **Critical:** Daniel Pacak is the original author and sole contributor to documentation, persistence layer, job queue, and most test infrastructure. He accounts for 35% of all commits.
- **High:** `pkg/job/`, `pkg/persistence/`, `pkg/queue/` have only 2 contributors (Daniel Pacak + Teppei Fukuda).
- **Medium:** Test fixtures and component test data are single-author areas.

---

## Cross-Repo Observations

### Authors Who Are Single Points of Failure

| Author | Repo | Sole-Owned Areas |
|--------|------|------------------|
| Patrick Eschenbach | harbor-cli | Replication, robot accounts, configurations, scan-all, test helpers, examples |
| Rizul Gupta | harbor-cli | Scanner, webhook, CVE allowlist, immutable tags, instance management |
| Daniel Pacak | harbor-scanner-trivy | Docs, persistence, job queue, component test infra |
| Prasanth Baskar / bupd | harbor-satellite | Crypto, SPIFFE, identity, token, container runtime, docker e2e |
| M Viswanath Sai | harbor-satellite | Watcher subsystem, DB schema migrations |
| Narhari Motivaras | harbor-satellite | E2E test config infrastructure |

### Most At-Risk Subsystems (Bus Factor = 1, Production-Critical)

1. **harbor-satellite `internal/crypto/`, `internal/spiffe/`, `internal/token/`** -- security-critical, single owner
2. **harbor-cli replication subsystem** -- core feature, single owner (Patrick Eschenbach)
3. **harbor-scanner-trivy `pkg/persistence/`, `pkg/job/`** -- core runtime, 2 owners max
4. **harbor-satellite `internal/watcher/`** -- runtime component, single owner

---

## Recommendations

### Immediate Actions (High Priority)

1. **Pair programming / code review rotation:** Require at least one reviewer from outside the original author for PRs touching single-owner areas.
2. **Cross-training sessions:** Schedule knowledge-sharing sessions for:
   - Patrick Eschenbach to walk through replication/robot/scan-all in harbor-cli
   - Daniel Pacak to document persistence and job queue architecture in harbor-scanner-trivy
   - Prasanth Baskar to document SPIFFE/crypto/identity flows in harbor-satellite
3. **Second-author commits:** Encourage contributors to make small improvements in single-owner areas to build familiarity.

### Medium-Term Actions

4. **Architecture documentation:** Create developer guides for subsystems with bus factor = 1, especially security-sensitive ones (crypto, SPIFFE, token).
5. **Onboarding paths:** Define "good first issues" in sole-owned areas to attract new contributors.
6. **Automated alerts:** Set up CODEOWNERS files and monitor for PRs that only the original author can review.

### Long-Term Actions

7. **Feature ownership rotation:** Periodically rotate maintainer responsibilities so knowledge spreads.
8. **Bus factor dashboard:** Integrate bus factor computation into CI to track trends over time.
