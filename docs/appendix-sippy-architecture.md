# Appendix: Sippy Architecture Deep Dive

## Overview

Sippy is a Go + React application with PostgreSQL (GORM) as the primary store and BigQuery as a secondary data source. It runs as a single binary with subcommands (`serve`, `load`, `migrate`, `snapshot`).

## Data Pipeline

```
Prow API → GCS (JUnit XML) → Data Loaders → PostgreSQL → Materialized Views → REST API → React UI
                                    ↑
                              BigQuery (historical data, variant resolution)
```

### Data Loaders (`pkg/dataloader/`)

13 specialized loaders, each responsible for one data source:

| Loader | Source | What it loads |
|--------|--------|---------------|
| `prowloader` | Prow API + GCS | Jobs, runs, test results (JUnit XML) |
| `bugloader` | Jira | Bugs linked to tests |
| `jiraloader` | Jira | Incidents, status |
| `releaseloader` | Release controller | Payload tags, acceptance status |
| `featuregateloader` | Config | Feature gate definitions |
| `variantloader` | BigQuery + config | Variant assignments per job |
| Others | Various | Build clusters, disruption data, etc. |

### Core Data Models (`pkg/db/models/`)

```
ProwJob (name, release, variants[], kind)
  └── ProwJobRun (timestamp, cluster, succeeded, duration, infra_failure, labels[])
        └── ProwJobRunTest (status, duration, lifecycle, suite)
              └── Test (name, bugs[], ownership)

ReleaseTag (version, stream, arch, phase, forced, reject_reason)
ReleaseDefinition (release metadata, GA date, capabilities)
Bug (jira_key, status, affects/fix versions, components)
TestDailyTotal / TestCumulativeSummary (pre-aggregated statistics)
```

## Variant Registry — The Heart of the Coupling

`pkg/variantregistry/ocp.go` (1,437 lines) defines 28 variant dimensions:

Platform, Architecture, Network, Topology, Upgrade, Installer, SecurityMode, Suite, FeatureSet, CGroupMode, ContainerRuntime, Scheduler, NetworkAccess, NetworkStack, Owner, and more.

Each OpenShift Prow job name encodes these dimensions. Example:
```
periodic-ci-openshift-release-master-nightly-4.16-e2e-aws-ovn-single-node
```
Parses to: Platform=AWS, Network=OVN, Topology=Single, Release=4.16, etc.

This is the single largest coupling point. ACS jobs don't follow this convention, so the entire registry would be replaced, not adapted.

## Component Readiness — The Core Feature

`pkg/api/componentreadiness/` + `pkg/componentreadiness/`

Compares test pass rates between a "basis" release (e.g., 4.15) and a "sample" release (e.g., 4.16). For each test × variant combination, runs a Fisher exact test to determine if the pass rate change is statistically significant.

The algorithm:
1. Query test results for basis and sample windows
2. Group by test × variant combination
3. For each group: compute pass/fail counts, run Fisher exact test
4. Apply confidence threshold and minimum sample size
5. Classify as: Significant Regression, Missing Data, Not Enough Data, Acceptable, or No Change

This is **domain-agnostic**. The statistics don't care what the tests are — they just need pass/fail counts grouped by dimensions.

## Server (`pkg/sippyserver/server.go`)

3,200 lines. Handles:
- HTTP route registration (80+ endpoints)
- Materialized view refresh scheduling
- Health checks and metrics
- API middleware (auth, caching, CORS)

## Frontend (`sippy-ng/`)

React + Material UI. Major pages:
- Component Readiness (the flagship — statistical regression grid)
- Jobs (pass rate, infra failure %, variant filtering)
- Tests (pass/fail/flake rates)
- Releases/payload health
- Pull requests
- Build clusters
- Chat (LLM-based query)

## What's Reusable for ACS

| Component | Reusable? | Notes |
|-----------|-----------|-------|
| Statistical engine (Fisher exact test) | Yes | Pure math, no domain coupling |
| Data model structure (Job→Run→Test) | Yes | Rename "ProwJob" but same shape |
| API framework and patterns | Yes | Standard Go REST |
| React UI shell | Mostly | Remove OCP pages, relabel |
| Component Readiness page | Yes | Core visualization |
| Jobs/Tests list pages | Yes | Generic data grid |
| Materialized view pattern | Yes | Performance optimization |
| Payload/release health page | No | OCP-specific concept |
| Build cluster page | No | OCP-specific |
| Variant registry | No | Must rewrite for ACS |
| Data loaders | Partial | BQ/JUnit reusable, Prow/GCS need replacement |
| Config format | No | 2MB OCP-specific, replace with ACS config |
