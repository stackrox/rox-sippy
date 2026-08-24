# Project Instructions for AI Agents

This file provides instructions and context for AI coding agents working on this project.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:7510c1e2 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->


## Build & Test

```bash
go build -mod=vendor ./cmd/sippy/...       # Backend
cd sippy-ng && npm run build               # Frontend
go test -mod=vendor ./pkg/...              # Backend tests
cd sippy-ng && CI=true npm test -- --coverage  # Frontend tests
```

## Cross-worker Invariants

1. **Model naming**: All Go code uses `CIJob`/`CIJobRun`/`CIJobRunTest` — never `ProwJob`. Table names: `ci_jobs`, `ci_job_runs`, `ci_job_run_tests`.
2. **Variant dimensions**: Exactly 6 (TestType, CloudProvider, Release, Framework, CISystem, Architecture).
3. **Status codes**: 0=pass, 1=fail, 12=flake.
4. **Module path**: Keep `github.com/openshift/sippy` — do NOT rename.
5. **Vendor mode**: Always `-mod=vendor`. Run `go mod vendor` after dep changes.
6. **No OCP references**: No "openshift", "ocp", or "ProwJob" outside vendor/ (except module path and fork-origin comments).
