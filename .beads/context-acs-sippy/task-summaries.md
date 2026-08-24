

## Completed Tasks

### BD-acs-sippy-2g6: Task 3: Rename data models from Prow to CI
**Worker**: backend-2 | **Files**: pkg/db/models/prow.go,pkg/api,pkg/sippyserver,sippy-ng/src
Renamed ProwJob→CIJob across 117 files. Structs, table names, migrations, API handlers, frontend types all updated. Remaining ProwJob refs only in pkg/apis/prow/ (external API types, acceptable).

### BD-acs-sippy-edx: Task 2: Remove OCP-specific frontend pages
**Worker**: frontend-1 | **Files**: sippy-ng/src/App.jsx,sippy-ng/src/Sidebar.jsx
Stripped OCP frontend: deleted 65 files (~15.8K lines) - build_clusters, chat, prow_job_runs, repositories, payload/install/feature_gate pages. Updated routes and nav. Landing redirects to Component R

### BD-acs-sippy-0q3: Task 1: Remove OCP-specific backend packages
**Worker**: backend-1 | **Files**: pkg/synthetictests,pkg/dataloader/prowloader,pkg/dataloader/releaseloader,pkg/dataloader/featuregateloader,pkg/variantregistry/ocp.go,pkg/testidentification/ocp_variants.go,config/openshift.yaml,cmd/sippy-daemon
Stripped OCP backend: deleted 49 files (412K lines) - synthetictests, prowloader, releaseloader, featuregateloader, ocp variants, openshift configs, sippy-daemon. Fixed 15+ files with compilation erro
