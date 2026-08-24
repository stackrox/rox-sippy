# Appendix: StackRox CI Infrastructure

## CI Systems

### GitHub Actions (Primary)

30+ workflow files in `.github/workflows/`. Key workflows:

| Workflow | Purpose |
|----------|---------|
| `build.yaml` (52KB) | Monolithic build + test orchestration |
| `scanner-e2e-test.yaml` | Scanner-specific E2E |
| `ci-failures-report.yml` | Weekly failure reports to Slack |
| `cut-rc.yml` | RC cutting automation |
| `finish-release.yml` | Release finishing |
| `nightly-tag.yml` | Nightly tag automation |
| `retest_comment.yml` / `retest_periodic.yml` | Automated PR retesting |
| `konflux-auto-retest.yml` | Konflux build retesting |

### OpenShift CI / Prow

`.openshift-ci/` contains Python-based E2E orchestration:
- `ci_tests.py` — test suite definitions and dispatch
- `base_qa_e2e_test.py` — base class for QA E2E tests
- `clusters.py` — cluster provisioning and management
- `flakechecker/flake-config.yml` — known flaky tests (regex patterns + ratio thresholds)

### Konflux / Tekton

`.tekton/` and `.konflux/` — container image builds, retagging, operator bundle pipelines. Supply-chain focused, less relevant for test analytics.

## Test Data Flow

```
Test Frameworks → JUnit XML → go-junit-report → junit-reports/report.xml
                                                          ↓
                                                    junit2jira → Jira tickets
                                                          ↓
                                                  scripts/ci/metrics.sh → BigQuery
                                                          ↓
                                              acs-san-stackroxci.ci_metrics
                                              ├── stackrox_tests
                                              └── stackrox_jobs
```

## Test Frameworks

| Framework | Location | Language | Test Type |
|-----------|----------|----------|-----------|
| Go test | Throughout `*_test.go` | Go | Unit |
| Spock/Groovy | `qa-tests-backend/` | Groovy | QA E2E |
| Go test | `tests/e2e/` | Go | Go E2E |
| Go test | `tests/upgrade/` | Go | Upgrade |
| Go test | `tests/performance/`, `scale/` | Go | Performance |
| Cypress | `ui/apps/platform/cypress/` | JS | UI E2E |
| Go test | `tests/complianceoperator/` | Go | Compliance |
| Python | `.openshift-ci/compatibility_test.py` | Python | Compatibility |

## BigQuery Tables

Project: `acs-san-stackroxci`, Dataset: `ci_metrics`

Tables:
- `stackrox_tests` — individual test results
- `stackrox_jobs` — job-level metadata

Data pushed via `scripts/ci/metrics.sh` using `bq` CLI.

**Schema investigation needed** — exact columns and data types not yet inspected. This determines how much of Sippy's data model maps directly vs needs adaptation.

## Existing Analytics

### Weekly Slack Report (`ci-failures-report.yml`)

Sends to `#acs-slack-ci-integration-testing`:
- Top-N failures by category (QA E2E, Operator E2E, UI E2E, Unit tests)
- Failure streaks ≥ 3 consecutive runs
- SQL: `scripts/ci/sql/test_failure_streaks.sql`

### Flake Tracking

`.openshift-ci/flakechecker/flake-config.yml` — static list of known flaky tests with:
- Regex pattern to match test name
- Ratio threshold (below which the test is considered flaky, not failing)

Manual maintenance — no automated flake detection.

### junit2jira

`.github/actions/junit2jira/` — automatically creates Jira tickets from JUnit failures. Configured to use `stackrox/junit2jira` v0.0.27 against `redhat.atlassian.net`.

## Release Structure

- Branch naming: `release-4.X` (observed: 4.2 through 4.10+)
- Versioning: `4.X.Y` with named-release-patch scheme
- RC cutting: automated cherry-pick from main, Jira issue check
- Nightly tags: automated via GHA
- No payload acceptance/rejection model (unlike OpenShift)

## ACS Variant Dimensions (Proposed)

ACS's CI is simpler than OpenShift's. Likely variant dimensions:

| Dimension | Values | Source |
|-----------|--------|--------|
| Test Type | unit, qa-e2e, go-e2e, upgrade, perf, ui-e2e, compliance, scanner | Job name / workflow |
| Cloud Provider | gcp, aws, azure, (none for unit) | Job configuration |
| Release Branch | main, release-4.X | Branch name |
| Framework | go-test, spock, cypress, python | Directory / job name |
| Architecture | amd64, arm64 (if applicable) | Job configuration |

5–8 dimensions vs OpenShift's 28. Parsing logic would be a few hundred lines, not 1,437.

## Key Insight

The hardest part of building a CI analytics tool — getting structured test data into a queryable store — is **already done** for ACS. The BigQuery tables exist, JUnit XML is standardized, and the data pipeline is running. What's missing is the analytics layer (statistical regression detection, trend visualization, self-service dashboards).
