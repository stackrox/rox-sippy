

## Completed Tasks

### BD-acs-sippy-4q4: Task 20: Write Kubernetes manifests
**Worker**: devops-2 | **Files**: deploy/deployment.yaml,deploy/postgres-statefulset.yaml,deploy/cronjob.yaml,deploy/service.yaml,deploy/ingress.yaml,deploy/configmap.yaml,deploy/secret-template.yaml,deploy/kustomization.yaml
Created K8s manifests: Deployment, PostgreSQL StatefulSet, daily CronJob, Service, Ingress, ConfigMap, Secret template, kustomization.yaml, README. All valid YAML with placeholder credentials.

### BD-acs-sippy-cpj: Task 19: Create container image
**Worker**: devops-1 | **Files**: Dockerfile
Updated Dockerfile: multi-stage build (Go+React+runtime), binary renamed to acs-sippy, ACS config files included, UBI9 runtime, 287MB image, builds and runs successfully.

### BD-acs-sippy-qxb: Task 17: Adapt remaining pages (Releases, PRs, Upgrades)
**Worker**: frontend-5 | **Files**: sippy-ng/src/App.jsx,sippy-ng/src/components/Sidebar.jsx,sippy-ng/src/pull_requests/PullRequestsTable.jsx,sippy-ng/src/releases/ReleaseOverview.jsx,sippy-ng/src/releases/Upgrades.jsx
Adapted remaining pages: updated Sidebar nav (ACS prioritization), removed payload columns from PRs, updated Upgrades title, changed app title to ACS Sippy, updated GitHub links to stackrox.

### BD-acs-sippy-0c5: Task 16: Adapt Jobs and Tests pages
**Worker**: frontend-4 | **Files**: sippy-ng/src/jobs/JobTable.jsx,sippy-ng/src/tests/TestTable.jsx
Adapted Jobs/Tests pages: replaced OCP variant priority keys with ACS 6 dimensions, removed OCP-specific logic (JobTier styling, OpenShift search links, Find Bugs button).

### BD-acs-sippy-co9: Task 15: Adapt Component Readiness page
**Worker**: frontend-3 | **Files**: sippy-ng/src/component_readiness/CompReadyVars.jsx,sippy-ng/src/component_readiness/CompReadyMainInputs.jsx,sippy-ng/src/component_readiness/ComponentReadinessHelp.jsx
Adapted CR page: replaced 28 OCP variants with 6 ACS dims, updated column grouping (CloudProvider,Architecture,TestType), simplified variant filters, removed OCP terminology from help.

### BD-acs-sippy-bq2: Task 14: Update frontend API client
**Worker**: frontend-2 | **Files**: sippy-ng/src/App.jsx,sippy-ng/src/components/RefreshButton.jsx,sippy-ng/src/component_readiness/CompReadyUtils.jsx,sippy-ng/src/component_readiness/ReleaseSelector.jsx
Updated frontend API client: removed OCP payload references from Component Readiness, added RefreshButton component for /api/refresh, verified all API field names use ci_job_* naming.

### BD-acs-sippy-6kl: Task 12: Wire Component Readiness for ACS
**Worker**: backend-7 | **Files**: pkg/api/componentreadiness/component_report.go,pkg/api/componentreadiness/dataprovider/postgres/provider.go,pkg/sippyserver/metrics/metrics.go,config/acs-views.yaml,config/acs.yaml
Wired Component Readiness for ACS: updated variant dimensions (6 ACS dims vs 28 OCP), updated DefaultColumnGroupBy/DefaultDBGroupBy, CacheVariants, fieldMap, Prometheus metrics labels. Created acs-vie

### BD-acs-sippy-11r: Task 11: Implement API endpoints
**Worker**: api-1 | **Files**: pkg/api/refresh.go,pkg/api/refresh_test.go,pkg/sippyserver/server.go,cmd/sippy/serve.go,cmd/sippy/component_readiness.go
Implemented API endpoints: most existing endpoints work as-is, added new /api/refresh with 15-min rate limiting wired to BQLoader.Sync(), updated NewServer() to accept dataSyncer. Tests passing.

### BD-acs-sippy-2r1: Task 10: Create materialized views
**Worker**: backend-6 | **Files**: pkg/db/migrations/000013_create_materialized_views.up.sql,pkg/db/migrations/000013_create_materialized_views.down.sql,pkg/db/views.go
Created materialized views: test_daily_summary and test_release_summary with unique indexes for CONCURRENTLY refresh, migration 000013, wired into RefreshData pipeline.

### BD-acs-sippy-khn: Task 8: Create ACS config file
**Worker**: config-1 | **Files**: config/acs.yaml
Created config/acs.yaml: 5 releases (main, 4.7-4.10), job regexp patterns, component readiness config. Loads via --config flag.

### BD-acs-sippy-a9f: Task 7: Wire up CLI subcommands
**Worker**: backend-5 | **Files**: cmd/sippy/main.go,cmd/sippy/load.go,pkg/flags/bigquery.go
Wired CLI: load subcommand uses BQLoader+ACSClassifier with acs-san-stackroxci defaults, updated binary name to acs-sippy, all three subcommands (load/migrate/serve) working.

### BD-acs-sippy-89z: Task 6: Implement ACS variant registry
**Worker**: backend-4 | **Files**: pkg/variantregistry/acs.go,pkg/variantregistry/acs_test.go
Implemented ACS variant classifier: 6 dimensions (TestType, CloudProvider, Release, Framework, CISystem, Architecture), pattern-based classification, ACSClassifier adapter for BQ loader, 30+ table-dri

### BD-acs-sippy-s7w: Task 5: Implement BQ data loader
**Worker**: backend-3 | **Files**: pkg/dataloader/bqloader/loader.go,pkg/dataloader/bqloader/loader_test.go,pkg/db/models/prow.go
Implemented BQ data loader: Sync() for jobs and tests, 90-day initial lookback, incremental sync, 15-min rate limiting, JobClassifier interface for Task 6, batch upserts, sync_state tracking. Tests pa

### BD-acs-sippy-2g6: Task 3: Rename data models from Prow to CI
**Worker**: backend-2 | **Files**: pkg/db/models/prow.go,pkg/api,pkg/sippyserver,sippy-ng/src
Renamed ProwJob→CIJob across 117 files. Structs, table names, migrations, API handlers, frontend types all updated. Remaining ProwJob refs only in pkg/apis/prow/ (external API types, acceptable).

### BD-acs-sippy-edx: Task 2: Remove OCP-specific frontend pages
**Worker**: frontend-1 | **Files**: sippy-ng/src/App.jsx,sippy-ng/src/Sidebar.jsx
Stripped OCP frontend: deleted 65 files (~15.8K lines) - build_clusters, chat, prow_job_runs, repositories, payload/install/feature_gate pages. Updated routes and nav. Landing redirects to Component R

### BD-acs-sippy-0q3: Task 1: Remove OCP-specific backend packages
**Worker**: backend-1 | **Files**: pkg/synthetictests,pkg/dataloader/prowloader,pkg/dataloader/releaseloader,pkg/dataloader/featuregateloader,pkg/variantregistry/ocp.go,pkg/testidentification/ocp_variants.go,config/openshift.yaml,cmd/sippy-daemon
Stripped OCP backend: deleted 49 files (412K lines) - synthetictests, prowloader, releaseloader, featuregateloader, ocp variants, openshift configs, sippy-daemon. Fixed 15+ files with compilation erro
