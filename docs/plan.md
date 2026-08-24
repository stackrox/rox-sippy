# Implementation Plan: ACS Sippy

## Setup

- [ ] Fork Sippy repo (`openshift/sippy`) into ACS org
  - AC: Forked repo exists in ACS org, cloned locally, builds with `go build ./...`
- [ ] Verify Go version matches Sippy's `go.mod` requirement
  - AC: `go version` output matches or exceeds the version in `go.mod`
- [ ] Verify Node.js version for `sippy-ng/` React frontend
  - AC: `node --version` meets the version requirement in `sippy-ng/package.json`
- [ ] Set up local PostgreSQL instance for development
  - AC: `psql` connects to local instance, can create and drop a test database
- [ ] Obtain read-only BQ service account credentials for `acs-san-stackroxci.ci_metrics`
  - AC: Service account JSON key file exists, `GOOGLE_APPLICATION_CREDENTIALS` points to it
- [ ] Verify BQ access: run a test query against `stackrox_tests` and `stackrox_jobs`
  - AC: `bq query 'SELECT COUNT(*) FROM acs-san-stackroxci.ci_metrics.stackrox_tests'` returns a row count

## Phase 1: Strip OpenShift Code [parallel]

Goal: Remove all OCP-specific code to create a clean, compilable shell.

- [ ] Task 1: Remove OCP-specific backend packages
  - Spec: spec.md §Architecture — Files Removed from Sippy Fork
  - [ ] Delete `pkg/variantregistry/ocp.go` and `pkg/variantregistry/snapshot.yaml`
  - [ ] Delete `pkg/testidentification/ocp_variants.go` and `ocp_never_stable.txt`
  - [ ] Delete `pkg/synthetictests/` entirely
  - [ ] Delete `pkg/dataloader/prowloader/`, `releaseloader/`, `featuregateloader/`
  - [ ] Delete `config/openshift.yaml`, `config/openshift-customizations.yaml`
  - [ ] Delete `cmd/sippy-daemon/`
  - [ ] Fix all compilation errors from removed packages (stub interfaces where needed)
  - AC: `go build ./...` succeeds with no OCP-specific packages remaining. `grep -r "openshift" pkg/` returns zero hits in non-test, non-vendor files.

- [ ] Task 2: Remove OCP-specific frontend pages
  - Spec: spec.md §UI Pages
  - [ ] Delete `sippy-ng/src/` directories: `build_clusters/`, `payloads/`, `feature_gates/`, `install/`, `chat/`
  - [ ] Remove routes for deleted pages from router config
  - [ ] Remove navigation links to deleted pages
  - [ ] Update landing page redirect to Component Readiness
  - [ ] Fix all TypeScript/ESLint errors from removed components
  - AC: `npm run build` succeeds in `sippy-ng/`. No routes or nav links to deleted pages. Landing page redirects to Component Readiness.

- [ ] Task 3: Rename data models from Prow to CI
  - Spec: spec.md §Data Model — GORM Models
  - [ ] Rename `ProwJob` → `CIJob`, `ProwJobRun` → `CIJobRun`, `ProwJobRunTest` → `CIJobRunTest` across all Go files
  - [ ] Update GORM table name annotations to match new schema names
  - [ ] Update all API handlers referencing old model names
  - [ ] Update frontend API client type names
  - AC: `go build ./...` succeeds. `grep -r "ProwJob" pkg/` returns zero hits (excluding comments explaining the rename).

- [ ] Task 4: Phase 1 integration test — clean build
  - AC: Both `go build ./...` and `cd sippy-ng && npm run build` succeed. The application starts (`acs-sippy serve`) without crash, serving an empty dashboard (no data loaded yet).

## Phase 2: ACS Data Pipeline

Goal: Build the BQ→PostgreSQL sync pipeline and ACS variant classification.

- [ ] Task 5: Implement BQ data loader
  - Spec: spec.md §Integrations — BigQuery, §Data Model — PostgreSQL Schema
  - [ ] Create `pkg/dataloader/bqloader/loader.go` — connects to BQ, runs incremental queries
  - [ ] Implement test sync: query `stackrox_tests` since last sync timestamp, upsert into `ci_job_run_tests` + `tests` + `suites`
  - [ ] Implement job sync: query `stackrox_jobs` since last sync timestamp, upsert into `ci_jobs` + `ci_job_runs`
  - [ ] Handle BQ field name casing (PascalCase for tests, snake_case for jobs)
  - [ ] Track last sync timestamp in a `sync_state` table
  - [ ] Implement on-demand sync with rate limiting (15-minute cooldown)
  - Contract: `func (l *BQLoader) Sync(ctx context.Context) error`
    - PRECONDITION: BQ credentials available via `GOOGLE_APPLICATION_CREDENTIALS`
    - POSTCONDITION: All new records since last sync are in PostgreSQL. `sync_state.last_sync` updated.
    - ERROR: Returns wrapped BQ client errors. Partial syncs are idempotent (re-running is safe).
  - Consumed by: Task 7 (CLI `load` subcommand), Task 9 (API refresh endpoint)
  - AC: Running the loader against the real BQ tables populates PostgreSQL with test and job data. Running it twice produces no duplicates. Incremental sync only fetches new records.

- [ ] Task 6: Implement ACS variant registry
  - Spec: spec.md §ACS Variant Registry
  - [ ] Create `pkg/variantregistry/acs.go` with `ClassifyJob(jobName, branch, ciSystem string) ACSVariant`
  - [ ] Implement pattern matching for all 6 dimensions (TestType, CloudProvider, Release, Framework, CISystem, Architecture)
  - [ ] Write classification rules per spec table
  - [ ] Integrate with BQ loader — classify jobs during sync
  - Contract: `func ClassifyJob(jobName, branch, ciSystem string) ACSVariant`
    - PRECONDITION: jobName is non-empty
    - POSTCONDITION: All 6 variant fields populated (defaults for unrecognized patterns)
    - ERROR: Never errors — unknown patterns get default values
  - Consumed by: Task 5 (BQ loader assigns variants during sync)
  - AC: Given ACS job names from real BQ data, the classifier correctly assigns TestType, CloudProvider, Release, Framework, CISystem, and Architecture. Unknown job patterns get sensible defaults (not empty strings).
  - TDD recommended — write test cases from real job names before implementing classifier.

- [ ] Task 7: Wire up CLI subcommands
  - Spec: spec.md §Architecture — Package Layout
  - [ ] Update `cmd/acs-sippy/` entry point — rename binary, update help text
  - [ ] Wire `load` subcommand to `bqloader.Sync()`
  - [ ] Wire `migrate` subcommand to GORM auto-migrate with new schema
  - [ ] Wire `serve` subcommand to start server with new routes
  - AC: `acs-sippy migrate` creates the new schema. `acs-sippy load` runs BQ sync. `acs-sippy serve` starts the HTTP server.

- [ ] Task 8: Create ACS config file
  - Spec: spec.md §Integrations — Component Ownership
  - [ ] Create `config/acs.yaml` with component→team ownership mapping
  - [ ] Create release definitions for active ACS releases (release-4.7 through release-4.10, main)
  - [ ] Load config at startup, make available to variant registry and API
  - AC: Config file loads without error. Component ownership mapping is queryable by API handlers.

- [ ] Task 9: Phase 2 integration test — data pipeline end-to-end
  - AC: `acs-sippy migrate && acs-sippy load` populates PostgreSQL from real BQ data. The `ci_jobs`, `ci_job_runs`, `ci_job_run_tests`, and `tests` tables contain data. Variant assignments are populated on `ci_jobs`. Test against live BQ — do not mock.

## Phase 3: Materialized Views and API

Goal: Build the analytics layer — materialized views and API endpoints.

- [ ] Task 10: Create materialized views
  - Spec: spec.md §Data Model — Materialized Views
  - [ ] Implement `test_daily_summary` materialized view
  - [ ] Implement `test_release_summary` materialized view
  - [ ] Add `REFRESH MATERIALIZED VIEW CONCURRENTLY` logic (requires unique indexes)
  - [ ] Wire refresh into the sync pipeline (refresh after data load)
  - AC: After a data load, materialized views contain aggregated data. Views refresh without blocking concurrent reads. Query against views returns correct pass/fail/flake counts matching raw data.

- [ ] Task 11: Implement API endpoints
  - Spec: spec.md §API Surface
  - [ ] Implement `GET /api/releases` — list active releases from config
  - [ ] Implement `GET /api/jobs` — job list with variant filtering, sorting, pagination
  - [ ] Implement `GET /api/tests` — test list with component filtering, status filtering
  - [ ] Implement `GET /api/tests/:id` — test detail with history and linked bugs
  - [ ] Implement `GET /api/pull_requests` — PR test impact
  - [ ] Implement `GET /api/refresh` — trigger on-demand sync with rate limiting
  - [ ] Implement `GET /api/health` — system health check
  - Contract: All endpoints return JSON matching the response formats in spec.md §API Surface.
    - PRECONDITION: PostgreSQL connection available, materialized views populated
    - POSTCONDITION: JSON response with correct `items`, `total`, `release` fields
    - ERROR: 404 for unknown releases, 400 for invalid parameters, 429 for rate-limited refresh
  - AC: Each endpoint returns valid JSON matching the spec response format. Filtering, sorting, and pagination work correctly. The refresh endpoint rate-limits to once per 15 minutes.

- [ ] Task 12: Wire Component Readiness for ACS
  - Spec: spec.md §API Surface — `/api/component_readiness`
  - [ ] Adapt existing Sippy Component Readiness to use ACS data model
  - [ ] Configure basis/sample release comparison using ACS release branches
  - [ ] Wire variant dimensions to ACS taxonomy (6 dimensions instead of 28)
  - [ ] Verify Fisher exact test produces correct p-values with ACS data
  - AC: `GET /api/component_readiness?basis=release-4.8&sample=release-4.9` returns a regression grid with per-component, per-variant pass rate comparisons and statistical significance. Results match manual calculation for a known test.

- [ ] Task 13: Phase 3 integration test — API end-to-end
  - AC: With populated PostgreSQL (from Phase 2), all API endpoints return meaningful data. Component Readiness correctly identifies known regressions between two releases. Test against live PostgreSQL with real BQ-sourced data — do not mock.

## Phase 4: Frontend Adaptation

Goal: Adapt the React frontend to use ACS data and remove OCP references.

- [ ] Task 14: Update frontend API client
  - Spec: spec.md §API Surface
  - [ ] Update API client to use renamed endpoints and response types
  - [ ] Remove references to OCP-specific API fields (payload, feature gates, etc.)
  - [ ] Update TypeScript types to match new API response shapes
  - AC: Frontend API client correctly calls all ACS Sippy endpoints. No TypeScript compilation errors.

- [ ] Task 15: Adapt Component Readiness page
  - Spec: spec.md §UI Pages
  - [ ] Update release selector to use ACS releases (release-4.X branches)
  - [ ] Update variant filter to use ACS dimensions (6 instead of 28)
  - [ ] Update labels and descriptions (remove OCP terminology)
  - [ ] Verify regression grid renders correctly with ACS data
  - AC: Component Readiness page loads, shows ACS releases in selectors, allows filtering by ACS variant dimensions, and renders the regression grid with real data.

- [ ] Task 16: Adapt Jobs and Tests pages
  - Spec: spec.md §UI Pages
  - [ ] Update Jobs page variant filters to ACS dimensions
  - [ ] Update Tests page component filter to ACS components
  - [ ] Update labels, column headers, and descriptions
  - [ ] Verify data grids render correctly with ACS data
  - AC: Jobs and Tests pages load with ACS data. Variant and component filters work. Sorting and pagination work.

- [ ] Task 17: Adapt remaining pages (Releases, PRs, Upgrades)
  - Spec: spec.md §UI Pages
  - [ ] Simplify Release Overview — remove payload concepts, show branch-based release health
  - [ ] Update Pull Requests page labels
  - [ ] Update Upgrades page for ACS upgrade test structure
  - [ ] Update global navigation bar — only show kept pages
  - AC: All remaining pages load without errors. Navigation only shows ACS-relevant pages.

- [ ] Task 18: Phase 4 integration test — UI end-to-end
  - AC: Full application (`acs-sippy serve`) serves the React UI. All pages load, display real data from PostgreSQL, and interactive features (filtering, sorting, pagination, drill-down) work. On-demand refresh button triggers a sync and data updates after refresh.

## Phase 5: Deployment

Goal: Package and deploy to the AWS K8s cluster.

- [ ] Task 19: Create container image
  - [ ] Write multi-stage Dockerfile (Go build + React build + runtime)
  - [ ] Configure image to run `acs-sippy serve` as default command
  - [ ] Verify image builds and runs locally
  - AC: `docker build` produces a working image. `docker run` starts the server, serves the UI.

- [ ] Task 20: Write Kubernetes manifests
  - Spec: spec.md §Architecture — System Boundaries
  - [ ] Deployment for acs-sippy (Go backend + React UI)
  - [ ] StatefulSet for PostgreSQL with PersistentVolumeClaim
  - [ ] CronJob for daily BQ sync (`acs-sippy load`)
  - [ ] Service + Ingress for external access
  - [ ] Secret for BQ service account credentials
  - [ ] ConfigMap for `config/acs.yaml`
  - AC: `kubectl apply` deploys all resources. Pod starts, connects to PostgreSQL, serves the dashboard. CronJob runs daily sync successfully.

- [ ] Task 21: Phase 5 integration test — deployed end-to-end
  - AC: Dashboard is accessible via the cluster's ingress. Data is populated from BQ. Component Readiness shows regression analysis. On-demand refresh works from the UI.

## Dependencies

```
Phase 1 (Strip OCP) → Phase 2 (Data Pipeline) → Phase 3 (Views + API) → Phase 4 (Frontend) → Phase 5 (Deploy)
                                                                           ↑
Within Phase 1: Tasks 1, 2, 3 are parallel → Task 4 (integration test)    │
Within Phase 2: Task 6 → Task 5 (loader uses classifier)                  │
                Task 8 → Task 5 (loader uses config)                       │
                Tasks 5, 6, 7, 8 → Task 9 (integration test)              │
Within Phase 3: Task 10 → Tasks 11, 12 (API reads from views)             │
                Tasks 10, 11, 12 → Task 13                                │
Within Phase 4: Task 14 → Tasks 15, 16, 17 (all use updated client)       │
                Tasks 15, 16, 17 → Task 18                                │
Phase 3 API must be complete before Phase 4 frontend adaptation ───────────┘
```

## Cross-worker Invariants

1. **Model naming**: All Go code uses `CIJob`/`CIJobRun`/`CIJobRunTest` — never `ProwJob`. Table names match: `ci_jobs`, `ci_job_runs`, `ci_job_run_tests`.
2. **Variant dimensions**: Exactly 6 dimensions (TestType, CloudProvider, Release, Framework, CISystem, Architecture). The JSONB `variants` column stores these as a string array. The variant registry is the single source of truth for classification.
3. **Status codes**: Test status uses integer codes: 0=pass, 1=fail, 12=flake. These match Sippy's existing convention.
4. **Incremental sync**: All BQ queries filter by `Timestamp > @last_sync_timestamp` or `started_at > @last_sync_timestamp`. Never full-table-scan BQ.
5. **Materialized view refresh**: Always use `REFRESH MATERIALIZED VIEW CONCURRENTLY` — never the blocking variant.
6. **No OCP references**: No file outside `vendor/` should contain "openshift", "ocp", or "ProwJob" (except comments explaining the fork origin).

## Interface Contracts

| Producer Task | Consumer Task | Contract |
|---|---|---|
| Task 6: Variant registry | Task 5: BQ loader | `ClassifyJob(jobName, branch, ciSystem) ACSVariant` — always returns all 6 fields populated |
| Task 5: BQ loader | Task 7: CLI subcommands | `BQLoader.Sync(ctx) error` — idempotent, incremental |
| Task 5: BQ loader | Task 9: API refresh endpoint | `BQLoader.Sync(ctx) error` — same contract |
| Task 10: Materialized views | Task 11: API endpoints | Views `test_daily_summary` and `test_release_summary` exist with columns per spec |
| Task 10: Materialized views | Task 12: Component Readiness | `test_release_summary` view provides per-test, per-variant aggregated pass/fail counts |
| Task 11: API endpoints | Task 14: Frontend API client | JSON response shapes per spec.md §API Surface |

## Spec Coverage

| Spec Section | Task(s) |
|---|---|
| Architecture — System Boundaries | Task 19: Create container image, Task 20: Write Kubernetes manifests |
| Architecture — Package Layout (forked from Sippy, restructured) | Task 1: Remove OCP-specific backend packages, Task 7: Wire up CLI subcommands |
| Architecture — Files Removed from Sippy Fork | Task 1: Remove OCP-specific backend packages, Task 2: Remove OCP-specific frontend pages |
| Data Model — PostgreSQL Schema | Task 5: Implement BQ data loader, Task 7: Wire up CLI subcommands (migrate) |
| Data Model — Materialized Views | Task 10: Create materialized views |
| Data Model — GORM Models | Task 3: Rename data models from Prow to CI, Task 5: Implement BQ data loader |
| ACS Variant Registry | Task 6: Implement ACS variant registry |
| API Surface — Core Endpoints (kept from Sippy, adapted) | Task 11: Implement API endpoints |
| API Surface — Component Readiness | Task 12: Wire Component Readiness for ACS |
| API Surface — Response Formats | Task 11: Implement API endpoints, Task 14: Update frontend API client |
| API Surface — Error Format | Task 11: Implement API endpoints |
| UI Pages (kept vs removed) | Task 2: Remove OCP-specific frontend pages, Task 15: Adapt Component Readiness page, Task 16: Adapt Jobs and Tests pages, Task 17: Adapt remaining pages |
| Integrations — BigQuery (data source) | Task 5: Implement BQ data loader |
| Integrations — Jira (bug linking) | (kept from Sippy, minimal changes) |
| Integrations — Component Ownership (seeded from ci-fixing-factory) | Task 8: Create ACS config file |
| Security & Privacy | Task 20: Write Kubernetes manifests (K8s Secret) |
| Known Gotchas | Task 3: Rename data models (renaming), Task 5: Implement BQ data loader (BQ casing), Task 10: Create materialized views (concurrent refresh) |

## Risks

1. **Sippy internals deeper than expected** — The "keep" components may have hidden dependencies on OCP-specific code not visible from the package structure. Mitigation: Phase 1's integration test (Task 4) validates clean compilation early. Budget extra time for Task 1.

2. **BQ schema drift** — If the `stackrox_tests` or `stackrox_jobs` schema changes upstream, the BQ loader breaks. Mitigation: Version the expected schema in `bqloader/` and add a startup health check that validates BQ table columns.

3. **Component Readiness adaptation complexity** — The statistical engine may be more tightly coupled to OCP data structures than the package layout suggests. Mitigation: Task 12 is scoped generously; if adaptation is harder than expected, consider reimplementing just the Fisher exact test comparison (the algorithm is straightforward).

4. **Low test coverage in fork** — Sippy may have tests that assume OCP data. After stripping OCP code, test coverage drops. Mitigation: Phase integration tests (Tasks 4, 9, 13, 18) validate end-to-end behavior against real data, compensating for lost unit tests.
