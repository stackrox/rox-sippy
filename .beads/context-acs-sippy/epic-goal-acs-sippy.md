

## Completed Tasks

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
