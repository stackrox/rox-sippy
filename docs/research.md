# Research: ACS Sippy

## Problem Statement

ACS (StackRox) has no unified CI health dashboard. Test results flow to BigQuery tables and weekly Slack reports, but there's no tool to:
- Detect statistical regressions across releases
- Surface flaky tests systematically (beyond a static YAML config)
- Track release health with pass/fail trends over time
- Give engineers a self-service view of "what's broken and why"

OpenShift has Sippy for this. The question: can we reuse Sippy for ACS, or do we need to build something new?

## Landscape

### Sippy (OpenShift's CI Private Investigator)

**What it does**: Analyzes CI job/test results, detects regressions with statistical significance (Fisher exact test), tracks release health, surfaces flaky tests, and supports release decisions. The flagship feature — Component Readiness — compares test pass rates between a "basis" release and a "sample" release to detect regressions.

**Architecture**: Go backend + React/MUI frontend + PostgreSQL + BigQuery. Data pipeline: Prow API → GCS (JUnit XML) → data loaders → PostgreSQL → materialized views → REST API → React UI.

**OpenShift coupling**: Deep. 204 of the Go source files reference "openshift" or "OCP". Key coupling points:

| Layer | What's coupled | Severity |
|-------|---------------|----------|
| Variant registry | `pkg/variantregistry/ocp.go` (1,437 lines) parses OCP job naming conventions to extract Platform, Arch, Network, Topology, etc. — 28 variant dimensions | Critical |
| Job config | `config/openshift.yaml` (2MB, auto-generated) maps every release to its Prow jobs | Critical |
| Variant snapshot | `pkg/variantregistry/snapshot.yaml` (9MB) pre-computed variant assignments | Critical |
| Test identification | `pkg/testidentification/ocp_variants.go` + `ocp_never_stable.txt` (70KB) | Critical |
| Release model | Payload tags with streams (nightly/ci), phases (Accepted/Rejected), architecture | High |
| Synthetic tests | `pkg/synthetictests/` generates OpenShift-specific install/upgrade health tests | High |
| Test suites | `pkg/db/suites.go` hardcodes recognized suite names | Medium |
| Component ownership | Maps tests to OCP components/capabilities | Medium |
| Data loaders | 13 specialized loaders assume Prow+GCS+BQ data sources | High |

**What IS generic**: The core data model (ProwJob → ProwJobRun → ProwJobRunTest → Test), the statistical engine (Fisher exact test, regression detection), the API framework, the React UI patterns, and the Jira integration.

Detailed architecture analysis in [appendix-sippy-architecture.md](appendix-sippy-architecture.md).

### StackRox CI Infrastructure (current state)

**CI systems**: Three — GitHub Actions (primary, 30+ workflows), OpenShift CI/Prow (E2E orchestration), Konflux/Tekton (container image builds).

**Test data already available**:
- JUnit XML via `go-junit-report` → `junit-reports/report.xml`
- BigQuery tables: `acs-san-stackroxci.ci_metrics.stackrox_tests` and `stackrox_jobs`
- Weekly Slack reports to `#acs-slack-ci-integration-testing` (top-N failures, streaks ≥ 3)
- `junit2jira` auto-creates Jira tickets from JUnit failures

**Test categories**: Unit (Go), QA E2E (Groovy/Spock), Go E2E, Upgrade, Performance, UI E2E (Cypress), Compliance, Scanner E2E, Compatibility.

**Release structure**: `release-4.X` branches (4.2–4.10+), RC cutting and release finishing automated via GHA workflows, nightly tags.

**Known pain points**: Flaky tests (static YAML config in `.openshift-ci/flakechecker/`), stale required checks, no merge queue, `wait-for-images` bottleneck, three CI systems creating complexity.

Detailed CI analysis in [appendix-stackrox-ci.md](appendix-stackrox-ci.md).

### TestGrid / Other Tools

**TestGrid**: Kubernetes-ecosystem tool for visualizing CI job results in a grid. Simpler than Sippy — shows pass/fail per job over time, no statistical regression detection. Already sunset/declining in the broader community.

**Prow Dashboard**: Basic job status view. StackRox has Prow access but it's view-only, no analytics.

**Custom BQ dashboards (Looker, Grafana)**: Possible to build on existing BQ data. Lower development cost but limited interactivity and no statistical engine.

## Available Resources

### Data sources (StackRox already has)
- **BigQuery tables** (`stackrox_tests`, `stackrox_jobs`) — structured test result data already flowing. This is the single biggest head start — Sippy also uses BQ.
- **JUnit XML** artifacts from all test frameworks — standard format, parseable.
- **GitHub Actions API** — job/workflow run metadata.
- **Jira** — bug tracking, already integrated via `junit2jira`.

### Infrastructure
- **GCP project** (`acs-san-stackroxci`) — BQ, GCS, compute already provisioned.
- **Existing SQL queries** — `scripts/ci/sql/test_failure_streaks.sql` and `metrics.sh` for BQ operations.
- **Slack integration** — weekly failure reports already wired up.

### Codebase
- **Sippy source** — Apache 2.0 licensed, forkable.
- **StackRox CI scripts** — `scripts/ci/`, `.github/workflows/`, `.openshift-ci/` for understanding data flow.

## Approach Options

### Option A: Generic Sippy (abstract the OpenShift coupling)

**Description**: Fork Sippy and refactor it into a generic CI analytics platform with pluggable "project profiles" — each profile defines variant parsing, job naming, release semantics, test suite recognition, and data source bindings.

**Effort estimate**: 3–6 months, 2–3 engineers.

**What it requires**:
1. Extract a `ProjectProfile` interface from the 28-dimension variant registry, job config, test identification, release model, synthetic tests, and component ownership
2. Move all OpenShift-specific code behind this interface as the first implementation
3. Define and implement an ACS profile as the second implementation
4. Refactor 13 data loaders to be composable (not all projects use Prow+GCS)
5. Generalize the 2MB config format and 9MB variant snapshot mechanism
6. Abstract the release model (OCP's payload acceptance/rejection vs ACS's branch-based releases)
7. Refactor the React frontend to be profile-aware (labels, navigation, default views)

**Trade-offs**:
- (+) Reusable across Red Hat — other products could adopt
- (+) Upstream contribution path — community benefits
- (−) Massive refactoring effort — touching 200+ files with OCP references
- (−) Risk of breaking Sippy for its primary users during refactor
- (−) Requires deep understanding of both Sippy internals AND each target project's CI
- (−) Maintenance burden: keeping the generic framework aligned with OpenShift's rapid Sippy evolution
- (−) The variant taxonomy is the core value prop, and it's inherently domain-specific — a "generic" version may be generic in form but still require deep per-project customization

### Option B: ACS-specific fork (strip OpenShift, add ACS)

**Description**: Fork Sippy, gut the OpenShift-specific code, and rebuild the data pipeline for ACS's CI infrastructure. Keep the statistical engine, API patterns, and React frontend shell.

**Effort estimate**: 6–10 weeks, 1–2 engineers.

**What it requires**:
1. Fork the Sippy repo
2. Replace variant registry with ACS-specific variant parsing (much simpler — ACS has fewer dimensions: test-type, cloud-provider, release-branch, maybe 5–8 dimensions vs OCP's 28)
3. Replace data loaders — ACS uses GHA (not Prow) as primary CI. Write a GHA loader + keep the existing BQ/JUnit loaders
4. Replace job config with ACS job definitions (much smaller — dozens of workflows vs thousands of OCP jobs)
5. Simplify release model (branch-based releases, no payload acceptance/rejection)
6. Remove synthetic tests, OCP component ownership, OCP test identification
7. Adapt the React UI — relabel, remove OCP-specific pages (payload health, build clusters), keep the core views (jobs, tests, component readiness)
8. Deploy to existing ACS GCP infrastructure

**Trade-offs**:
- (+) Much smaller scope — clear what to cut, clear what to keep
- (+) ACS's BQ data is already there — data pipeline mostly exists
- (+) ACS's CI is simpler — fewer variant dimensions, fewer CI systems to integrate
- (+) Can ship an MVP in weeks, iterate from there
- (+) Team can maintain it without understanding Sippy's full complexity
- (−) No reuse by other products — ACS-only
- (−) Diverges from upstream Sippy — can't easily pull improvements
- (−) Still requires understanding Sippy's internals well enough to gut them cleanly

### Option C: Build from scratch on existing BQ data

**Description**: Skip Sippy entirely. Build a lightweight CI health dashboard from scratch, reading from StackRox's existing BigQuery tables. Use the statistical concepts from Sippy (Fisher exact test) but implement them in a simpler architecture.

**Effort estimate**: 4–8 weeks for MVP, 1–2 engineers.

**What it requires**:
1. Define ACS variant taxonomy (test-type, cloud-provider, release, framework — ~5–8 dimensions)
2. Build a BQ query layer for test pass rates, flake detection, regression detection
3. Implement Fisher exact test for release-over-release comparison (the core Sippy algorithm — straightforward to reimplement, it's standard statistics)
4. Build a web UI (React or even a simpler stack) for job/test views and component readiness
5. Wire up data ingestion — either read existing BQ tables directly or add a lightweight ETL
6. Deploy to ACS GCP

**Trade-offs**:
- (+) Simplest architecture — no legacy code to understand or strip
- (+) Designed for ACS from day one — no abstraction tax
- (+) Can choose modern tooling (e.g., Go + htmx, or Next.js, vs Sippy's older React setup)
- (+) Fastest path to MVP if scoped tightly (just regression detection + test health)
- (−) Loses Sippy's years of UI polish and edge-case handling
- (−) No upstream community — you maintain everything
- (−) Risk of re-learning lessons Sippy already encoded (e.g., how to handle infrastructure failures vs real test failures, how to classify flakes)
- (−) Fisher exact test is the easy part — the hard part is data quality, variant classification, and knowing what to show users

### Recommendation

**Option B (ACS-specific fork)** is the sweet spot.

- Option A (generic Sippy) is a multi-month platform play with unclear demand beyond "it would be nice." The coupling is too deep and too OCP-specific for a clean abstraction — you'd be building a framework for a problem only a handful of teams have.
- Option C (from scratch) throws away too much. Sippy's UI, its handling of infrastructure failures vs test failures, its flake detection, and its materialized view patterns represent years of hard-won lessons.
- Option B gives you the valuable parts (statistical engine, UI shell, API patterns, deployment model) without the baggage (OCP variant taxonomy, payload acceptance, Prow-specific loaders). ACS's CI is simple enough that the replacement code is much smaller than what you're removing.

The critical advantage: **ACS already has test data in BigQuery**. The hardest part of any CI analytics tool is getting clean data in — and that's already done.

## Out of Scope

- **Replacing or modifying StackRox's existing CI infrastructure** — this consumes existing data, doesn't change how tests run
- **Generic multi-project platform** — solving for ACS only (Option A explicitly deferred)
- **PR commenting daemon** — Sippy has one (`sippy-daemon`), but ACS already has `junit2jira`; PR commenting is a later enhancement
- **Chat/LLM interface** — Sippy has an experimental chat feature; not needed for MVP
- **Build cluster analytics** — OCP-specific feature, ACS doesn't need it

## Open Questions

1. **What's in the BQ tables today?** — Resolved from codebase analysis. See details below.

### BigQuery Schema (from codebase)

**Project**: `acs-san-stackroxci`, Dataset: `ci_metrics`

**`stackrox_jobs` table** (written by `scripts/ci/metrics.sh` via `bq_save_job_record`):

| Column | Type | Source |
|--------|------|--------|
| `id` | STRING | `BUILD_ID` (Prow) or `GITHUB_RUN_ID.ATTEMPT.JOB.RANDOM` (GHA) |
| `name` | STRING | Job name |
| `repo` | STRING | Repository full name |
| `branch` | STRING | Git branch |
| `pr_number` | INTEGER | PR number (nullable) |
| `commit_sha` | STRING | Git commit SHA |
| `started_at` | TIMESTAMP | Job start time |
| `stopped_at` | TIMESTAMP | Job stop time |
| `ci_system` | STRING | CI system identifier |
| (additional fields) | various | Extensible via field-value pairs in `save_job_record` |

**`stackrox_tests` table** (CSV uploaded by `junit2jira` → GCS → batch load):

| Column | Type | Source (inferred from SQL queries) |
|--------|------|--------|
| `Name` | STRING | Test name |
| `Classname` | STRING | Test suite/package (e.g., `github.com/stackrox/rox/...`) |
| `JobName` | STRING | CI job name |
| `Status` | STRING | `passed`, `failed` |
| `Timestamp` | TIMESTAMP | When the test ran |
| `BuildTag` | STRING | Build identifier (e.g., `PR_NUMBER/merge@SHA` for PRs) |

**`stackrox_tests__extended_view`** — A BQ view joining tests with job metadata. Used by the Slack failure reports. Adds fields like `ShortJobName` and `IsPullRequest`.

**Additional tables**:
- `stackrox_central_metrics` — Central component metrics
- `stackrox_image_prefetches` — Image prefetch timing data

**Data pipeline**: `junit2jira` parses JUnit XML → writes CSV → uploads to GCS (`gs://stackrox-ci-artifacts/test-metrics/upload/`) → batch workflow loads CSVs into BQ in batches of 20 (to avoid GCP quota issues). Job records are written directly to BQ via parameterized SQL inserts.

**ci-fixing-factory** (`~/work/code/ci-fixing-factory`): A separate project for automated CI failure analysis. Not a data source — it's a consumer. Its `ci-analyze` skill config references BQ (`project: stackrox-ci-metrics, dataset: test_failures, table: failure_history`) for historical failure lookups. Also maps StackRox component owners (Scanner→team-vulnerability-mgmt, Central→team-core, etc.) and defines classification rules for failures (KNOWN_FLAKE, INFRASTRUCTURE, REGRESSION, etc.). The component ownership mapping and classification taxonomy could inform ACS Sippy's variant/component model.

*This question is now resolved — the existing BQ schema covers the essential fields for Sippy-style analytics (test name, suite, job, status, timestamp, build metadata). No additional data pipeline work needed for MVP.*

2. **Who would use this?** — All three: RIT (release health, regression detection), engineering teams (flaky test identification, test health trends), and TRT-equivalent stakeholders (release readiness decisions). The UI should serve engineers doing daily triage as well as leads making release calls.
3. **Where would it run?** — On a Kubernetes cluster on AWS. Data lives in GCP BigQuery (`acs-san-stackroxci`). Cross-cloud data transfer cost is minimal for this use case: Sippy's pattern is a daily batch load from BQ into PostgreSQL — the transferred volume is aggregated query results (megabytes), not raw test data (gigabytes). GCP egress charges at this scale would be under $1/month. The app itself (Go backend + PostgreSQL + React frontend) runs entirely on the AWS cluster after the daily sync.
4. **How often should data refresh?** — Daily automatic refresh, plus an on-demand refresh option in the UI. The daily cadence covers normal usage, but engineers investigating a fresh failure need current data without waiting for the next scheduled sync. The on-demand refresh triggers an immediate BQ query and materialized view update. Rate-limiting (e.g., max once per 15 minutes) prevents abuse and BQ cost spikes.
5. **Is the weekly Slack report sufficient for some use cases?** — The weekly Slack report and the dashboard coexist. The Slack report serves as a push notification (top failures, streaks) while the dashboard serves as a pull interface (self-service investigation, drill-down, historical trends). The dashboard could eventually generate the Slack report, but that's an optimization, not a requirement.

## Sources

- Sippy source code: `~/code/sippy/` (Go backend, React frontend, config files)
- StackRox source code: `~/work/code/stackrox/` (CI workflows, test infrastructure, BQ scripts)
- Sippy README and architecture docs
- StackRox `.github/workflows/`, `.openshift-ci/`, `scripts/ci/`
- BigQuery project: `acs-san-stackroxci.ci_metrics`
