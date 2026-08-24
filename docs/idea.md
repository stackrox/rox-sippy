# ACS Sippy

## One-liner

A CI health dashboard for ACS that detects test regressions across releases, surfaces flaky tests, and gives engineers self-service visibility into what's broken and why — built by forking Sippy and replacing its OpenShift-specific layers with ACS equivalents.

## Problem

ACS has test data flowing to BigQuery and a weekly Slack failure report, but no analytics layer on top. Engineers can't answer basic questions without writing ad-hoc BQ queries:

- "Did the 4.9 release regress compared to 4.8?"
- "Which tests are flaky vs genuinely failing?"
- "What's the pass rate trend for scanner E2E tests over the last month?"
- "Are we ready to cut a release?"

OpenShift solved this with Sippy. ACS needs the same capability, adapted to its simpler CI landscape.

## Who Is It For

**RIT (Release & Integration Team)**: Release health dashboards, regression detection between release branches, release readiness signals. Primary power users.

**Engineering teams**: Flaky test identification, test health trends for their components, self-service investigation when their tests break. Daily triage use case.

**Release stakeholders**: High-level release readiness views, pass/fail trends across releases, confidence signals for release decisions.

## What Does Success Look Like

1. Engineers stop writing ad-hoc BQ queries to understand CI health — the dashboard answers their questions directly.
2. RIT uses the regression detection view (Component Readiness) to make release decisions, replacing manual analysis.
3. Flaky tests are identified automatically with statistical confidence, replacing the static YAML config in `.openshift-ci/flakechecker/`.
4. Test failure investigations start at the dashboard and drill down, rather than starting from raw GHA logs.
5. The weekly Slack report continues alongside the dashboard — push (Slack) for awareness, pull (dashboard) for investigation.

## Core Concept

### Fork and Replace

Fork Sippy's codebase and surgically replace the OpenShift-coupled layers while keeping the generic foundation:

**Keep** (the valuable parts):
- Statistical engine — Fisher exact test for regression detection, Component Readiness comparison logic
- Data model structure — Job → Run → Test hierarchy (rename from ProwJob but same shape)
- API framework — Go REST endpoints, caching, pagination
- React/MUI frontend shell — data grids, charts, navigation patterns
- PostgreSQL + materialized views — the performance optimization layer
- Jira integration — bug linking (already connects to `redhat.atlassian.net`)

**Replace** (the OpenShift-specific parts):
- Variant registry — from 28 OCP dimensions to ~6 ACS dimensions (see below)
- Data loaders — from Prow+GCS to BQ-first (ACS data is already in BigQuery)
- Job config — from 2MB auto-generated OCP config to a lightweight ACS job registry
- Release model — from payload acceptance/rejection to branch-based releases
- Test identification — from OCP suite recognition to ACS test categorization
- Synthetic tests — removed entirely (OCP-specific concept)
- Component ownership — seeded from ci-fixing-factory's mapping (Scanner→team-vulnerability-mgmt, Central→team-core, etc.)

### ACS Variant Taxonomy

ACS CI is much simpler than OpenShift's. The proposed variant dimensions:

| Dimension | Example Values | Parsed From |
|-----------|---------------|-------------|
| Test Type | unit, qa-e2e, go-e2e, upgrade, perf, ui-e2e, compliance, scanner | Job name / workflow file |
| Cloud Provider | gcp, aws, azure, none | Job configuration |
| Release | main, release-4.8, release-4.9 | Branch name |
| Framework | go-test, spock, cypress | Directory / job name |
| CI System | gha, prow, konflux | `ci_system` field in BQ |
| Architecture | amd64, arm64 | Job configuration (if applicable) |

6 dimensions vs OpenShift's 28. The variant parser would be a few hundred lines, not 1,437.

### Data Pipeline

```
Existing (no changes needed):
  JUnit XML → junit2jira → CSV → GCS → BQ batch load → stackrox_tests
  CI scripts → bq INSERT → stackrox_jobs

New (ACS Sippy adds):
  stackrox_tests + stackrox_jobs (BQ) → Daily sync → PostgreSQL → Materialized views
                                              ↓
                                     REST API → React UI
```

The critical insight: ACS's data pipeline already gets test results into BigQuery. ACS Sippy reads from BQ and loads into PostgreSQL for fast querying — the same pattern Sippy uses, but without needing to build the data ingestion layer.

### Data Refresh

- **Daily automatic sync**: Scheduled job queries BQ, loads new results into PostgreSQL, refreshes materialized views
- **On-demand refresh**: UI button triggers an immediate sync. Rate-limited to once per 15 minutes to prevent BQ cost spikes
- **Incremental loads**: Only fetch data since the last sync timestamp, not full table scans

### Deployment

Runs on an AWS Kubernetes cluster. Components:
- Go backend (single binary, serves API + static frontend assets)
- PostgreSQL (persistent volume)
- CronJob for daily BQ sync

Cross-cloud data transfer (GCP BQ → AWS app) is negligible — aggregated query results are megabytes, not gigabytes.

## Key Design Decisions

1. **Fork Sippy, don't build from scratch.** Sippy's statistical engine, UI patterns, and materialized view architecture represent years of hard-won lessons about CI analytics. Reimplementing these from scratch would mean relearning how to handle infrastructure failures vs test failures, how to classify flakes, and how to present regression data usefully. The fork approach inherits all of this.

2. **ACS-specific, not generic.** A generic Sippy would require abstracting 200+ files of OCP-specific code into pluggable interfaces — a 3–6 month project with unclear demand. ACS-specific means we can ship in weeks and maintain without deep Sippy internals knowledge. If other RH products want this later, they can fork our fork.

3. **BQ as the source of truth, PostgreSQL as the query layer.** ACS data already flows to BQ. Rather than building a parallel data ingestion pipeline, we read from BQ and materialize into PostgreSQL for fast UI queries. This is exactly Sippy's existing architecture — we're just changing what it reads from.

4. **Component ownership seeded from ci-fixing-factory.** The `ci-fixing-factory` project already maps StackRox components to owning teams and classifies failures (KNOWN_FLAKE, INFRASTRUCTURE, REGRESSION). This taxonomy seeds ACS Sippy's component model, avoiding a cold start. See [appendix-stackrox-ci.md](appendix-stackrox-ci.md).

5. **Daily refresh + on-demand.** CI analytics is a trends tool, not a real-time monitor. Daily refresh is sufficient for most use cases. On-demand refresh (rate-limited) covers the "I just merged a fix, did it help?" scenario without overloading BQ.

## What This Is NOT

- **Not a CI system** — it reads existing test data, doesn't run tests or manage jobs
- **Not a replacement for junit2jira** — PR-level failure→ticket automation continues as-is
- **Not a generic platform** — ACS-only; the generic Sippy vision is explicitly deferred
- **Not a real-time monitor** — daily refresh + on-demand, not streaming
- **Not a replacement for the Slack report** — they coexist; the dashboard is the drill-down complement
- **Not a build system dashboard** — no Konflux/Tekton build analytics (that's a different concern)

## Language / Runtime

**Go backend + React/MUI frontend + PostgreSQL** — inherited from Sippy. No reason to change the stack:
- Go is ACS's primary language — team can maintain it
- React/MUI is mature and well-understood
- PostgreSQL with materialized views handles the analytics query patterns efficiently

## Open Questions

All resolved — see spec.md for details.

1. **Which Sippy UI pages to keep vs cut?** — Resolved: 12 pages kept (Component Readiness, Jobs, Job Analysis, Tests, Test Analysis, Release Overview, Pull Requests, Upgrades, Variant Status, Home), 11 pages removed (Payload Streams/Tags/Details, Feature Gates, Build Clusters, Install, Intervals/Events Charts, Chat, Repositories). See spec.md §UI Pages.
2. **How to handle the three CI systems in variant classification?** — Resolved: same pattern as Sippy. Separate data loaders per CI source, all normalizing into the common Job→Run→Test model. The variant registry classifies jobs after loading, independent of source. The `ci_system` field in `stackrox_jobs` distinguishes the origin.
3. **Authentication/authorization model?** — Resolved: no app-level auth. Access controlled at K8s cluster level. Matches Sippy's model (open within Red Hat network).
