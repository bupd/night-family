# Security Foot-Gun Finder Report

**Date:** 2026-04-18
**Scope:** All local projects under `/var/home/bupd/code/`

---

## Executive Summary

Scanned harbor-satellite, harbor-scanner-trivy, ks (Kubernetes configs), arch-bootc-hetzner, and shell scripts for common security anti-patterns. Found **30+ issues** across 3 severity levels. Most critical: hardcoded credentials in Kubernetes manifests, privileged containers, and disabled TLS verification.

| Severity | Count | Key Areas |
|----------|-------|-----------|
| HIGH     | 16    | Hardcoded creds, privileged containers, host mounts, curl\|bash |
| MEDIUM   | 8     | sslmode=disable, NOPASSWD sudo, missing NetworkPolicy |
| LOW      | 3     | HTTP in examples, temp file cleanup, debug logging |

---

## HIGH Severity Findings

### 1. Hardcoded Credentials in Kubernetes/Helm Values

Plaintext passwords across multiple Helm value files. These should use Kubernetes Secrets or an external secrets manager.

| File | Detail |
|------|--------|
| `ks/gitea/values.yaml:63` | `password: gitea-db-pass-2026` |
| `ks/gc-chart/values.yaml:25` | `DB_PASSWORD: "password"` |
| `ks/gc-chart/values.yaml:30` | `HARBOR_PASSWORD: "Harbor12345"` |
| `ks/harbor/values.yaml:245,772,927,947` | Multiple default passwords |
| `ks/depl/harbor-sat.values.yaml:279` | `harborAdminPassword: "Harbor12345"` |
| `ks/depl/harbor-sat.values.yaml:365` | `secretKey: "not-a-secure-key"` |

### 2. Hardcoded Token in Docker Compose

**File:** `harbor-satellite/main/docker-compose.yml:8`
```yaml
- TOKEN=c78dc95cae68e73664a067cb8bc0c6d2
```

### 3. Hardcoded Demo Credentials

**File:** `harbor-satellite/main/master-demo.sh:12-19`
```bash
HARBOR_USERNAME="admin"
HARBOR_PASSWORD="Harbor12345"
ADMIN_PASSWORD="Harbor12345"
SAT_PASS="password"
```

### 4. Privileged Containers

**File:** `ks/harbor-debugging-psp.yaml`
- `privileged: true`, `runAsUser: RunAsAny`, `volumes: ["*"]`, `SYS_PTRACE` capability

**File:** `ks/depl/harbor-sat.values.yaml:477-487`
- `privileged: true`, `allowPrivilegeEscalation: true`
- AppArmor and seccomp both `Unconfined`
- `runAsNonRoot: false`, `SYS_PTRACE` capability

### 5. Host Filesystem Mounts

| File | Mount |
|------|-------|
| `ks/kube-registry.yaml:45-46` | `hostPath: /data/registry/` |
| `ks/kind-hostpath.yml:17-19` | `hostPath: /` (entire host root!) |

### 6. Overly Permissive Directory Permissions

**File:** `harbor-scanner-trivy/pkg/etc/checker.go:61`
```go
os.MkdirAll(path, 0777)
```
Cache and report directories created world-writable.

### 7. InsecureSkipVerify Enabled

**File:** `harbor-scanner-trivy/pkg/trivy/target.go:87`
```go
tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: config.Insecure}
```

**File:** `harbor-satellite/main/internal/state/registration_process.go:195-199`
```go
transport.TLSClientConfig.InsecureSkipVerify = useUnsecure
```

### 8. Curl Piped to Bash

| File | Command |
|------|---------|
| `arch-bootc-hetzner/Containerfile:163` | `curl -fsSL https://bun.sh/install \| bash` |
| `arch-bootc-hetzner/Containerfile:166` | `curl -fsSL https://claude.ai/install.sh \| bash` |
| `arch-bootc-hetzner/files/ensure-homebrew.sh:17` | Homebrew installer piped to bash |

---

## MEDIUM Severity Findings

### 9. Database SSL Disabled

**Files:** `harbor-satellite/main/ground-control/internal/server/server.go:81`, `ground-control/migrator/migrator.go:31`
```go
"postgres://...?sslmode=disable"
```

### 10. API Server Without TLS

**File:** `harbor-scanner-trivy/pkg/http/api/server.go:95`
- Server can run without TLS; credentials transmitted in plaintext.

### 11. NOPASSWD Sudo

**File:** `arch-bootc-hetzner/Containerfile:135`
```bash
echo "%wheel ALL=(ALL:ALL) NOPASSWD: ALL" > /etc/sudoers.d/wheel
```

### 12. Missing Network Policies

**Directory:** `ks/` — No `NetworkPolicy` resources found. All pod-to-pod traffic allowed by default.

### 13. Credentials in Environment Variables

**File:** `harbor-scanner-trivy/pkg/trivy/wrapper.go:257-268`
- Registry passwords and GitHub tokens passed via env vars, visible in `/proc/[pid]/environ`.

---

## LOW Severity Findings

### 14. HTTP in Example Configs

**Files:** `harbor-satellite/main/config.example.json`, `docker-compose.byo.yml`
- Example configs use `http://` instead of `https://`, setting poor precedent.

### 15. URL Construction Without Encoding

**File:** `harbor-satellite/main/internal/state/registration_process.go:155`
- Token interpolated into URL without `url.PathEscape()`.

---

## Positive Findings (No Issues)

- **harbor-satellite**: Uses `crypto/rand` correctly, parameterized SQL queries (sqlc), proper file permissions (0600/0700), no `unsafe` package
- **harbor-scanner-trivy**: No `crypto/md5`/`crypto/sha1`, no hardcoded secrets in Go code, Dockerfile runs as non-root user (`scanner`)
- **harbor-satellite**: Proper TLS defaults (`MinVersion: tls.VersionTLS12`, mTLS support)

---

## Recommendations

1. **Secrets Management**: Migrate all hardcoded credentials to Sealed Secrets, External Secrets Operator, or Vault
2. **Container Security**: Remove `privileged: true`, set `runAsNonRoot: true`, drop all capabilities, use restricted seccomp/AppArmor profiles
3. **Volume Security**: Replace `hostPath` with PVCs; never mount host root
4. **TLS Enforcement**: Use `sslmode=require` for Postgres, enforce TLS on API servers
5. **File Permissions**: Use `0755` for cache dirs, `0700` for report dirs
6. **Network Policies**: Add default-deny NetworkPolicy with explicit allow rules
7. **Install Scripts**: Verify checksums/signatures for curl-piped installers
