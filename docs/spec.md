# Technical Spec: ACS Sippy

## Architecture

### System Boundaries

```
┌─────────────────────────────────────────────────────┐
│ AWS Kubernetes Cluster                               │
│                                                      │
│  ┌──────────────┐    ┌──────────────┐               │
│  │ Go Backend   │◄──►│ PostgreSQL   │               │
│  │ (API + UI)   │    │ (data store) │               │
│  └──────┬───────┘    └──────────────┘               │
│         │                                            │
│  ┌──────┴───────┐                                   │
│  │ CronJob      │                                   │
│  │ (daily sync) │                                   │
│  └──────┬───────┘                                   │
│         │                                            │
└─────────┼────────────────────────────────────────────┘
          │ BQ query (daily + on-demand)
          ▼
┌─────────────────────┐
│ GCP BigQuery         │
│ acs-san-stackroxci   │
│ ├── stackrox_tests   │
│ └── stackrox_jobs    │
└─────────────────────┘
```

### Package Layout (forked from Sippy, restructured)

```
acs-sippy/
├── cmd/
│   └── acs-sippy/          # CLI entry point (serve, load, migrate)
├── pkg/
│   ├── api/                # REST API handlers
│   │   ├── jobs/           # Job endpoints
│   │   ├── tests/          # Test endpoints
│   │   ├── componentreadiness/  # Regression detection (kept from Sippy)
│   │   └── releases/       # Release health endpoints
│   ├── db/
│   │   ├── models/         # GORM models (renamed from ProwJob → CIJob)
│   │   ├── migrations/     # Schema migrations
│   │   └── query/          # Materialized view definitions, query helpers
│   ├── dataloader/
│   │   └── bqloader/       # BigQuery → PostgreSQL sync (replaces prowloader)
│   ├── variantregistry/
│   │   └── acs.go          # ACS variant parser (~200 lines, replaces ocp.go's 1,437)
│   ├── componentreadiness/  # Statistical engine (Fisher exact test — kept as-is)
│   ├── testidentification/  # ACS test classification
│   └── server/             # HTTP server, routing, middleware
├── sippy-ng/               # React frontend (trimmed)
│   └── src/
│       ├── component_readiness/  # Keep
│       ├── jobs/                 # Keep
│       ├── tests/                # Keep
│       ├── releases/             # Keep (simplified)
│       ├── pull_requests/        # Keep
│       └── upgrades/             # Keep
└── config/
    └── acs.yaml            # ACS job registry (replaces 2MB openshift.yaml)
```

### Files Removed from Sippy Fork

| Sippy Path | Reason |
|------------|--------|
| `pkg/variantregistry/ocp.go` (1,437 lines) | Replaced by `acs.go` |
| `pkg/variantregistry/snapshot.yaml` (9MB) | Not needed — ACS variant assignment is simple enough to compute at load time |
| `config/openshift.yaml` (2MB) | Replaced by `acs.yaml` |
| `config/openshift-customizations.yaml` | Not needed |
| `pkg/testidentification/ocp_variants.go` | Replaced by ACS test identification |
| `pkg/testidentification/ocp_never_stable.txt` (70KB) | Not needed |
| `pkg/synthetictests/` | Removed entirely (OCP-specific) |
| `pkg/dataloader/prowloader/` | Replaced by `bqloader/` |
| `pkg/dataloader/releaseloader/` | OCP payload concept |
| `pkg/dataloader/featuregateloader/` | OCP feature gates |
| `sippy-ng/src/build_clusters/` | OCP-specific |
| `sippy-ng/src/payloads/` | OCP-specific |
| `sippy-ng/src/feature_gates/` | OCP-specific |
| `sippy-ng/src/install/` | OCP-specific |
| `sippy-ng/src/chat/` | Not needed for MVP |
| `cmd/sippy-daemon/` | PR commenter — ACS uses junit2jira |

## Data Model

### PostgreSQL Schema

```sql
CREATE TABLE ci_jobs (
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    release       TEXT NOT NULL,          -- e.g., 'main', 'release-4.9'
    variants      JSONB DEFAULT '[]',     -- computed variant assignments
    ci_system     TEXT NOT NULL,          -- 'gha', 'prow', 'konflux'
    UNIQUE(name, release)
);

CREATE TABLE ci_job_runs (
    id              SERIAL PRIMARY KEY,
    ci_job_id       INTEGER NOT NULL REFERENCES ci_jobs(id),
    bq_id           TEXT NOT NULL UNIQUE,  -- original ID from stackrox_jobs
    commit_sha      TEXT,
    branch          TEXT,
    pr_number       INTEGER,
    started_at      TIMESTAMP NOT NULL,
    stopped_at      TIMESTAMP,
    outcome         TEXT NOT NULL,         -- 'passed', 'failed', 'canceled'
    infra_failure   BOOLEAN DEFAULT FALSE,
    repo            TEXT,
    INDEX idx_job_runs_started (started_at),
    INDEX idx_job_runs_ci_job (ci_job_id)
);

CREATE TABLE ci_job_run_tests (
    id              SERIAL PRIMARY KEY,
    ci_job_run_id   INTEGER NOT NULL REFERENCES ci_job_runs(id),
    test_id         INTEGER NOT NULL REFERENCES tests(id),
    suite_id        INTEGER REFERENCES suites(id),
    status          SMALLINT NOT NULL,     -- 0=pass, 1=fail, 12=flake
    duration        REAL,
    INDEX idx_run_tests_run (ci_job_run_id),
    INDEX idx_run_tests_test (test_id)
);

CREATE TABLE tests (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    component       TEXT,                  -- owning component (e.g., 'scanner', 'central')
    team            TEXT                   -- owning team (e.g., 'team-vulnerability-mgmt')
);

CREATE TABLE suites (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE
);

CREATE TABLE releases (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,   -- e.g., 'release-4.9'
    ga_date         DATE,
    is_active       BOOLEAN DEFAULT TRUE
);

CREATE TABLE bugs (
    id              SERIAL PRIMARY KEY,
    jira_key        TEXT NOT NULL UNIQUE,   -- e.g., 'ROX-12345'
    status          TEXT,
    summary         TEXT,
    component       TEXT,
    affects_versions TEXT[],
    fix_versions    TEXT[],
    last_updated    TIMESTAMP
);

CREATE TABLE bug_tests (
    bug_id          INTEGER NOT NULL REFERENCES bugs(id),
    test_id         INTEGER NOT NULL REFERENCES tests(id),
    PRIMARY KEY (bug_id, test_id)
);
```

### Materialized Views

```sql
-- Daily aggregated test pass rates (partitioned by release + date)
CREATE MATERIALIZED VIEW test_daily_summary AS
SELECT
    t.id AS test_id,
    j.release,
    DATE(jr.started_at) AS day,
    j.ci_system,
    COUNT(*) FILTER (WHERE jrt.status = 0) AS passes,
    COUNT(*) FILTER (WHERE jrt.status = 1) AS failures,
    COUNT(*) FILTER (WHERE jrt.status = 12) AS flakes,
    COUNT(*) AS total_runs
FROM ci_job_run_tests jrt
JOIN ci_job_runs jr ON jr.id = jrt.ci_job_run_id
JOIN ci_jobs j ON j.id = jr.ci_job_id
JOIN tests t ON t.id = jrt.test_id
GROUP BY t.id, j.release, DATE(jr.started_at), j.ci_system;

-- Cumulative test summary per release (for Component Readiness)
CREATE MATERIALIZED VIEW test_release_summary AS
SELECT
    t.id AS test_id,
    t.name AS test_name,
    t.component,
    j.release,
    j.variants,
    COUNT(*) FILTER (WHERE jrt.status = 0) AS passes,
    COUNT(*) FILTER (WHERE jrt.status = 1) AS failures,
    COUNT(*) FILTER (WHERE jrt.status = 12) AS flakes,
    COUNT(*) AS total_runs,
    MIN(jr.started_at) AS first_run,
    MAX(jr.started_at) AS last_run
FROM ci_job_run_tests jrt
JOIN ci_job_runs jr ON jr.id = jrt.ci_job_run_id
JOIN ci_jobs j ON j.id = jr.ci_job_id
JOIN tests t ON t.id = jrt.test_id
GROUP BY t.id, t.name, t.component, j.release, j.variants;
```

### GORM Models

```go
type CIJob struct {
    ID       uint           `gorm:"primaryKey"`
    Name     string         `gorm:"not null"`
    Release  string         `gorm:"not null"`
    Variants pq.StringArray `gorm:"type:jsonb;default:'[]'"`
    CISystem string         `gorm:"column:ci_system;not null"`
    Runs     []CIJobRun
}

type CIJobRun struct {
    ID           uint      `gorm:"primaryKey"`
    CIJobID      uint      `gorm:"not null"`
    BQID         string    `gorm:"column:bq_id;uniqueIndex"`
    CommitSHA    string
    Branch       string
    PRNumber     *int
    StartedAt    time.Time `gorm:"not null"`
    StoppedAt    *time.Time
    Outcome      string    `gorm:"not null"` // passed, failed, canceled
    InfraFailure bool      `gorm:"default:false"`
    Repo         string
    Tests        []CIJobRunTest
}

type CIJobRunTest struct {
    ID         uint `gorm:"primaryKey"`
    CIJobRunID uint `gorm:"not null"`
    TestID     uint `gorm:"not null"`
    SuiteID    *uint
    Status     int  `gorm:"not null"` // 0=pass, 1=fail, 12=flake
    Duration   *float32
}

type Test struct {
    ID        uint   `gorm:"primaryKey"`
    Name      string `gorm:"uniqueIndex;not null"`
    Component string
    Team      string
    Bugs      []Bug `gorm:"many2many:bug_tests"`
}
```

## ACS Variant Registry

```go
// pkg/variantregistry/acs.go

type ACSVariant struct {
    TestType      string // unit, qa-e2e, go-e2e, upgrade, perf, ui-e2e, compliance, scanner
    CloudProvider string // gcp, aws, azure, none
    Release       string // main, release-4.8, release-4.9
    Framework     string // go-test, spock, cypress
    CISystem      string // gha, prow, konflux
    Architecture  string // amd64, arm64
}

// ClassifyJob assigns variant dimensions to a CI job based on its name,
// branch, and CI system metadata.
func ClassifyJob(jobName, branch, ciSystem string) ACSVariant {
    v := ACSVariant{
        Release:  classifyRelease(branch),
        CISystem: ciSystem,
    }
    v.TestType = classifyTestType(jobName)
    v.CloudProvider = classifyCloudProvider(jobName)
    v.Framework = classifyFramework(jobName)
    v.Architecture = classifyArchitecture(jobName)
    return v
}
```

Classification rules (pattern-matched from job names):

| Dimension | Pattern | Value |
|-----------|---------|-------|
| TestType | `*-unit-*`, `*_unit_*` | `unit` |
| TestType | `*qa-e2e*`, `*qa_e2e*` | `qa-e2e` |
| TestType | `*-e2e-*` (not qa) | `go-e2e` |
| TestType | `*upgrade*` | `upgrade` |
| TestType | `*perf*`, `*scale*` | `perf` |
| TestType | `*cypress*`, `*ui-e2e*` | `ui-e2e` |
| TestType | `*compliance*` | `compliance` |
| TestType | `*scanner*` | `scanner` |
| CloudProvider | `*-aws-*`, `*_aws_*` | `aws` |
| CloudProvider | `*-gcp-*`, `*_gcp_*` | `gcp` |
| CloudProvider | `*-azure-*` | `azure` |
| CloudProvider | (no match) | `none` |
| Release | `release-4.\d+` | `release-4.X` |
| Release | `main`, `master` | `main` |
| Framework | job in `qa-tests-backend/` or name contains `qa` | `spock` |
| Framework | job contains `cypress` | `cypress` |
| Framework | (default) | `go-test` |

## API Surface

### Core Endpoints (kept from Sippy, adapted)

```
GET  /api/releases                              # List active releases
GET  /api/releases/:release                     # Release overview (pass rates, trends)

GET  /api/jobs                                  # Job list with filters
     ?release=release-4.9
     &variant=test_type:qa-e2e
     &sortField=current_pass_percentage
     &sort=asc
     &limit=50&offset=0

GET  /api/jobs/:id/runs                         # Runs for a specific job

GET  /api/tests                                 # Test list with filters
     ?release=release-4.9
     &component=scanner
     &status=failing
     &limit=50&offset=0

GET  /api/tests/:id                             # Test detail (history, linked bugs)

GET  /api/component_readiness                   # Regression detection
     ?basis=release-4.8
     &sample=release-4.9
     &confidence=95
     &min_runs=10
     &group_by=component

GET  /api/pull_requests                         # PR test impact
     ?release=release-4.9
     &limit=50&offset=0

GET  /api/health                                # System health check
GET  /api/refresh                               # Trigger on-demand BQ sync (rate-limited)
```

### Response Formats

```json
// GET /api/tests?release=release-4.9&status=failing
{
  "items": [
    {
      "id": 1234,
      "name": "TestScannerVulnerabilityDetection",
      "component": "scanner",
      "team": "team-vulnerability-mgmt",
      "current_pass_percentage": 85.2,
      "previous_pass_percentage": 98.1,
      "current_runs": 120,
      "current_passes": 102,
      "current_failures": 15,
      "current_flakes": 3,
      "net_change": -12.9,
      "bugs": [{"key": "ROX-12345", "status": "In Progress"}],
      "variants": {"test_type": "scanner", "ci_system": "gha"}
    }
  ],
  "total": 42,
  "release": "release-4.9"
}
```

```json
// GET /api/component_readiness?basis=release-4.8&sample=release-4.9
{
  "rows": [
    {
      "component": "scanner",
      "columns": [
        {
          "variant": {"test_type": "scanner", "cloud_provider": "gcp"},
          "basis_pass_rate": 0.981,
          "sample_pass_rate": 0.852,
          "fisher_exact_p": 0.0023,
          "status": "SignificantRegression",
          "basis_runs": 200,
          "sample_runs": 120
        }
      ]
    }
  ],
  "basis_release": "release-4.8",
  "sample_release": "release-4.9",
  "generated_at": "2026-08-21T10:00:00Z"
}
```

```json
// GET /api/refresh
{
  "status": "started",           // or "rate_limited"
  "last_refresh": "2026-08-21T06:00:00Z",
  "next_allowed": "2026-08-21T10:15:00Z",
  "message": "Sync started. Data will be available in ~2-5 minutes."
}
```

### Error Format

```json
{
  "error": "invalid_parameter",
  "message": "release 'release-4.99' not found",
  "status": 404
}
```

## UI Pages (kept vs removed)

| Sippy Page | Decision | Rationale |
|------------|----------|-----------|
| Component Readiness | **Keep** | Flagship feature — statistical regression grid |
| Jobs | **Keep** | Job pass rates, filtering by variant |
| Job Analysis | **Keep** | Deep-dive into individual jobs |
| Tests | **Keep** | Test pass/fail/flake rates |
| Test Analysis | **Keep** | Deep-dive into individual tests |
| Release Overview | **Keep** (simplified) | Release-level summary, no payload concepts |
| Pull Requests | **Keep** | PR test impact view |
| Upgrades | **Keep** | ACS has upgrade tests |
| Variant Status | **Keep** | Status per variant dimension |
| Payload Streams/Tags/Details | **Remove** | OCP payload concept, no ACS equivalent |
| Feature Gates | **Remove** | OCP-specific |
| Build Clusters | **Remove** | OCP-specific |
| Install | **Remove** | OCP install tests, not applicable |
| Intervals/Events Charts | **Remove** | OCP job run introspection, too specialized |
| Chat | **Remove** | LLM feature, not needed for MVP |
| Repositories | **Remove** | ACS is monorepo, per-repo view not useful |
| Home/Landing | **Adapt** | Redirect to Component Readiness as default |

**12 pages kept/adapted, 11 pages removed.**

## Integrations

### BigQuery (data source)

**Project**: `acs-san-stackroxci`, **Dataset**: `ci_metrics`

Sync queries read from `stackrox_tests` and `stackrox_jobs`. See [appendix-stackrox-ci.md](appendix-stackrox-ci.md) for full BQ schema.

```sql
-- Incremental test sync (fetch since last sync)
SELECT Name, Classname, JobName, Status, Timestamp, BuildTag
FROM `acs-san-stackroxci.ci_metrics.stackrox_tests`
WHERE Timestamp > @last_sync_timestamp
ORDER BY Timestamp ASC;

-- Incremental job sync
SELECT id, name, repo, branch, pr_number, commit_sha,
       started_at, stopped_at, ci_system, outcome
FROM `acs-san-stackroxci.ci_metrics.stackrox_jobs`
WHERE started_at > @last_sync_timestamp
ORDER BY started_at ASC;
```

### Jira (bug linking)

Same integration as Sippy — queries `redhat.atlassian.net` for bugs matching ACS components. Kept from Sippy with minimal changes (update project filter from OCP to ROX/RHACS).

### Component Ownership (seeded from ci-fixing-factory)

Initial mapping:

| Component | Team |
|-----------|------|
| scanner | team-vulnerability-mgmt |
| central | team-core |
| sensor | team-sensor-ecosystem |
| collector | team-collector |
| ui | team-acs-ui |
| compliance | team-core-workflows |
| operator | team-automation |

Stored in `config/acs.yaml`, editable without code changes.

## Security & Privacy

- **No app-level authentication** — access controlled at K8s cluster level (cluster requires auth to reach the service)
- **No PII in test data** — test names, job names, pass/fail status are not sensitive
- **BQ access** — service account with read-only access to `ci_metrics` dataset. Credentials stored as K8s Secret, mounted into the backend pod
- **Rate limiting** — on-demand refresh endpoint rate-limited to once per 15 minutes per cluster (not per user, since there's no user identity)

## Known Gotchas

1. **GORM model renaming** — Sippy's models are named `ProwJob`, `ProwJobRun`, `ProwJobRunTest`. Renaming to `CIJob`/`CIJobRun`/`CIJobRunTest` requires updating every query, every API handler, and every test. Use find-and-replace but verify GORM table name annotations (`gorm:"table:..."`) match the new schema.

2. **Materialized view refresh locking** — PostgreSQL's `REFRESH MATERIALIZED VIEW` takes an exclusive lock. Use `CONCURRENTLY` to avoid blocking reads, but this requires a unique index on the materialized view. Sippy already handles this — don't break the pattern.

3. **BQ query costs** — BigQuery charges per bytes scanned. The `stackrox_tests` table grows daily. Use partitioned queries (filter by `Timestamp`) and avoid `SELECT *`. The incremental sync pattern (fetch since last timestamp) keeps costs bounded.

4. **Fisher exact test edge cases** — When sample sizes are very small (< 5 runs), the Fisher exact test produces unreliable p-values. Sippy handles this with a `min_runs` threshold (configurable, default 10). Keep this behavior.

5. **Frontend route cleanup** — React Router routes for removed pages must be deleted, not just hidden. Stale routes that 404 at the API layer create confusing user experiences.

6. **BQ field name casing** — `stackrox_tests` uses PascalCase (`Name`, `Classname`, `JobName`), while `stackrox_jobs` uses snake_case (`started_at`, `ci_system`). The BQ loader must handle both conventions when mapping to GORM models.
