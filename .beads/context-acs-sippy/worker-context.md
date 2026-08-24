# ACS Sippy — Worker Context

## Overview

Forking openshift/sippy to build ACS Sippy — a CI analytics dashboard for ACS (Advanced Cluster Security). We strip OCP-specific code, add ACS-specific data loaders (BigQuery → PostgreSQL), variant classification, and adapt the React frontend.

## Tech Stack

- **Language/Runtime**: Go 1.25, TypeScript/React
- **Framework**: Go stdlib HTTP + gorilla/mux, React (Create React App in sippy-ng/)
- **Key libraries**: GORM (ORM), BigQuery client, Material-UI, Nivo charts
- **Testing**: Go test + gotestsum, Jest/React Testing Library
- **Build**: `go build -mod=vendor ./cmd/sippy/...` (backend), `npm run build` in sippy-ng/ (frontend)
- **ORM**: GORM with PostgreSQL

## Build Environment

- **Build backend**: `go build -mod=vendor ./cmd/sippy/...`
- **Build frontend**: `cd sippy-ng && npm run build`
- **Test backend**: `go test -mod=vendor ./pkg/...`
- **Test frontend**: `cd sippy-ng && CI=true npm test -- --coverage`
- **Lint**: `./hack/go-lint.sh run ./...` (Go), `cd sippy-ng && npx eslint .` (JS)
- **Full build**: `make build`

## Repo Structure

```
cmd/
  sippy/            # Main binary entry point (rename to acs-sippy)
  sippy-daemon/     # OCP-specific daemon (to be deleted)
pkg/
  api/              # API types and handlers
  apis/             # API route registration
  bigquery/         # BQ client utilities
  cache/            # Caching layer
  componentreadiness/ # Component readiness analysis (Fisher exact test)
  dataloader/       # Data loading (prowloader/ to be replaced with bqloader/)
  db/               # Database utilities
  filter/           # Query filtering
  sippyserver/      # HTTP server setup
  synthetictests/   # OCP synthetic tests (to be deleted)
  testidentification/ # Test classification (OCP variants to be replaced)
  variantregistry/  # Variant definitions (OCP to be replaced with ACS)
sippy-ng/           # React frontend
  src/              # React components and pages
config/             # Configuration files (openshift.yaml to be replaced)
vendor/             # Go vendor directory
docs/               # Project documentation, spec, plan
```

## Conventions

- **Commit messages**: conventional format (`feat:`, `fix:`, `chore:`) — enforced by `tf.py worker-close`. Never include task/bead numbers (e.g., do NOT write "feat: Task 9: ...")
- **Module path**: `github.com/openshift/sippy` (keeping original module path to avoid mass import rewrite)
- **Vendor mode**: Always use `-mod=vendor` for Go builds. Run `go mod vendor` after adding dependencies.
- **Logging**: Use `slog` or `klog` for structured logging (match existing Sippy patterns)
- **Error handling**: Wrap errors with `fmt.Errorf("context: %w", err)`
- **Test files**: Every new production file should have a corresponding `_test.go` file. Table-driven tests for classifiers/converters are mandatory.
- **Frontend**: Material-UI components, follow existing sippy-ng patterns
- **Non-interactive shell**: Always use `-f` flags with `cp`, `mv`, `rm` to avoid interactive prompts

## Security

- Any value interpolated into SQL must use parameterized queries (GORM handles this)
- BQ credentials via `GOOGLE_APPLICATION_CREDENTIALS` env var, never hardcoded
- No secrets in config files — use K8s Secrets for deployment

## Key Specs

- Full spec: `docs/spec.md`
- Implementation plan: `docs/plan.md`
- Sippy architecture reference: `docs/appendix-sippy-architecture.md`
- StackRox CI reference: `docs/appendix-stackrox-ci.md`

## Cross-worker Invariants

1. **Model naming**: All Go code uses `CIJob`/`CIJobRun`/`CIJobRunTest` — never `ProwJob`. Table names match: `ci_jobs`, `ci_job_runs`, `ci_job_run_tests`.
2. **Variant dimensions**: Exactly 6 dimensions (TestType, CloudProvider, Release, Framework, CISystem, Architecture). The JSONB `variants` column stores these as a string array. The variant registry is the single source of truth for classification.
3. **Status codes**: Test status uses integer codes: 0=pass, 1=fail, 12=flake. These match Sippy's existing convention.
4. **Incremental sync**: All BQ queries filter by `Timestamp > @last_sync_timestamp` or `started_at > @last_sync_timestamp`. Never full-table-scan BQ.
5. **Materialized view refresh**: Always use `REFRESH MATERIALIZED VIEW CONCURRENTLY` — never the blocking variant.
6. **No OCP references**: No file outside `vendor/` should contain "openshift", "ocp", or "ProwJob" (except comments explaining the fork origin and the Go module path which stays as `github.com/openshift/sippy`).

## Known Gotchas

- The Go module path is `github.com/openshift/sippy` — do NOT change this. Changing it would require rewriting every import in every file including vendor/. The module path stays as-is.
- `vendor/` directory is committed — always use `-mod=vendor` and run `go mod vendor` after dependency changes
- sippy-ng uses Create React App — `npm run build` produces static files embedded into the Go binary via `embed.go`
- GORM auto-migration: table names derive from struct names. Renaming `ProwJob` → `CIJob` changes the table name unless `TableName()` is overridden.
- sendmessage: available (worker reuse enabled)

### Build Failures

If the build command fails due to platform-specific issues, use the most targeted verification command available for your language/framework, and note the limitation in your worker-close summary.

### Transient Diagnostics

LSP diagnostics during active worker runs (unresolved imports, undefined symbols in partially-written files) are almost always transient. Use the build tool as ground truth, not LSP. Do not act on these until the responsible worker completes.

- Workers never call `bd` directly — all bead operations go through `tf.py` subcommands (`claim`, `block`, `discover`, `worker-close`)
- **Never run `git stash -u` or `git stash --include-untracked`** — this stashes `.beads/context-*/` files and breaks orchestration state. Use `git stash` (tracked files only) or `git stash push <specific-files>` instead.
