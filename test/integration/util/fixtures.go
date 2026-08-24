package util

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	apitype "github.com/openshift/sippy/pkg/apis/api"
	v1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/db/models"
)

func CreateCIJob(t *testing.T, dbc *db.DB, name, release string, variants []string) models.CIJob {
	t.Helper()
	job := models.CIJob{
		Name:     name,
		Release:  release,
		Variants: pq.StringArray(variants),
	}
	require.NoError(t, dbc.DB.Create(&job).Error, "creating CIJob %q", name)
	return job
}

type CIJobOption func(*models.CIJob)

func WithKind(kind models.ProwKind) CIJobOption {
	return func(j *models.CIJob) {
		j.Kind = kind
	}
}

// WithVariantCombination sets both Variants and VariantCombinationID from an existing
// VariantCombination. In production, VariantCombinationID is populated by a database
// trigger; the integration schema intentionally skips triggers (see SetupIntegrationSchema),
// so tests must set it explicitly via this option.
func WithVariantCombination(vc models.VariantCombination) CIJobOption {
	return func(j *models.CIJob) {
		j.Variants = vc.Variants
		j.VariantCombinationID = &vc.ID
	}
}

// CreateVariantCombination creates a variant_combinations row. Pass the result to
// WithVariantCombination when creating a CIJob so cumulative-summary-based reports
// (which join through ci_jobs.variant_combination_id) can find it.
func CreateVariantCombination(t *testing.T, dbc *db.DB, variants []string) models.VariantCombination {
	t.Helper()
	vc := models.VariantCombination{Variants: pq.StringArray(variants)}
	require.NoError(t, dbc.DB.Create(&vc).Error, "creating VariantCombination %v", variants)
	return vc
}

func CreateCIJobWithOptions(t *testing.T, dbc *db.DB, name, release string, variants []string, opts ...CIJobOption) models.CIJob {
	t.Helper()
	job := models.CIJob{
		Name:     name,
		Release:  release,
		Variants: pq.StringArray(variants),
	}
	for _, opt := range opts {
		opt(&job)
	}
	require.NoError(t, dbc.DB.Create(&job).Error, "creating CIJob %q", name)
	return job
}

type CIJobRunOption func(*models.CIJobRun)

func WithURL(url string) CIJobRunOption {
	return func(r *models.CIJobRun) { r.URL = url }
}

// WithLabels sets the job run's Labels array (e.g. infrafailure.LabelInfraFailure).
// Read-time summary queries exclude runs carrying the InfraFailure label.
func WithLabels(labels ...string) CIJobRunOption {
	return func(r *models.CIJobRun) { r.Labels = pq.StringArray(labels) }
}

func CreateCIJobRun(t *testing.T, dbc *db.DB, ciJobID uint, release string, timestamp time.Time, succeeded bool, overallResult v1.JobOverallResult, opts ...CIJobRunOption) models.CIJobRun {
	t.Helper()
	run := models.CIJobRun{
		CIJobID:      ciJobID,
		CIJobRelease: release,
		Timestamp:      timestamp,
		Succeeded:      succeeded,
		Failed:         !succeeded,
		OverallResult:  overallResult,
	}
	for _, opt := range opts {
		opt(&run)
	}
	require.NoError(t, dbc.DB.Create(&run).Error, "creating CIJobRun")
	return run
}

func CreateTest(t *testing.T, dbc *db.DB, name string) models.Test {
	t.Helper()
	test := models.Test{Name: name}
	require.NoError(t, dbc.DB.Create(&test).Error, "creating Test %q", name)
	return test
}

func CreateSuite(t *testing.T, dbc *db.DB, name string) models.Suite {
	t.Helper()
	suite := models.Suite{Name: name}
	require.NoError(t, dbc.DB.Create(&suite).Error, "creating Suite %q", name)
	return suite
}

// CreateJiraComponent creates a jira_components row. Test report queries resolve
// TestOwnership.JiraComponent (a plain string) against JiraComponent.Name to populate
// jira_component_id, so the two must match for that join to succeed.
func CreateJiraComponent(t *testing.T, dbc *db.DB, name string) models.JiraComponent {
	t.Helper()
	jc := models.JiraComponent{Name: name}
	require.NoError(t, dbc.DB.Create(&jc).Error, "creating JiraComponent %q", name)
	return jc
}

type TestOwnershipOption func(*models.TestOwnership)

// WithTestOwnershipJiraComponent sets the string joined against JiraComponent.Name.
// Note this is distinct from TestOwnership.JiraComponentID, which test report queries
// don't use for the jira_components join.
func WithTestOwnershipJiraComponent(name string) TestOwnershipOption {
	return func(to *models.TestOwnership) {
		to.JiraComponent = name
	}
}

func WithTestOwnershipJiraComponentID(id *uint) TestOwnershipOption {
	return func(to *models.TestOwnership) {
		to.JiraComponentID = id
	}
}

func WithTestOwnershipCapabilities(caps []string) TestOwnershipOption {
	return func(to *models.TestOwnership) {
		to.Capabilities = pq.StringArray(caps)
	}
}

// CreateTestOwnership creates a test_ownerships row scoped to a specific suite. Pass
// suiteID as nil for a suite-agnostic ownership row.
func CreateTestOwnership(t *testing.T, dbc *db.DB, testID uint, suiteID *uint, uniqueID, component string, opts ...TestOwnershipOption) models.TestOwnership {
	t.Helper()
	to := models.TestOwnership{
		TestID:    testID,
		SuiteID:   suiteID,
		UniqueID:  uniqueID,
		Name:      uniqueID,
		Component: component,
	}
	for _, opt := range opts {
		opt(&to)
	}
	require.NoError(t, dbc.DB.Create(&to).Error, "creating TestOwnership for test %d", testID)
	return to
}

type CIJobRunTestOption func(*models.CIJobRunTest)

func WithSuiteID(suiteID uint) CIJobRunTestOption {
	return func(pjrt *models.CIJobRunTest) {
		pjrt.SuiteID = &suiteID
	}
}

func WithLifecycle(lifecycle string) CIJobRunTestOption {
	return func(pjrt *models.CIJobRunTest) {
		pjrt.Lifecycle = lifecycle
	}
}

// WithDuration sets the test result's duration (seconds). TestDurations averages
// this column per day.
func WithDuration(duration float64) CIJobRunTestOption {
	return func(pjrt *models.CIJobRunTest) {
		pjrt.Duration = duration
	}
}

func CreateCIJobRunTest(t *testing.T, dbc *db.DB, ciJobRunID, ciJobID, testID uint, release string, timestamp time.Time, status int, opts ...CIJobRunTestOption) models.CIJobRunTest {
	t.Helper()
	pjrt := models.CIJobRunTest{
		CIJobRunID:        ciJobRunID,
		CIJobID:           ciJobID,
		TestID:              testID,
		CIJobRunRelease:   release,
		CIJobRunTimestamp: timestamp,
		Status:              status,
	}
	for _, opt := range opts {
		opt(&pjrt)
	}
	require.NoError(t, dbc.DB.Create(&pjrt).Error, "creating CIJobRunTest")
	return pjrt
}

// CreateCIJobRunTestOutput creates the ci_job_run_test_outputs row for a given
// CIJobRunTest. It wires the composite (id, timestamp, release) key that the
// read-time TestOutputs query joins on, denormalizing the timestamp and release
// from the parent test result.
func CreateCIJobRunTestOutput(t *testing.T, dbc *db.DB, pjrt models.CIJobRunTest, output string) models.CIJobRunTestOutput {
	t.Helper()
	o := models.CIJobRunTestOutput{
		CIJobRunTestID:        pjrt.ID,
		Output:                  output,
		CIJobRunTestTimestamp: pjrt.CIJobRunTimestamp,
		CIJobRunTestRelease:   pjrt.CIJobRunRelease,
	}
	require.NoError(t, dbc.DB.Create(&o).Error, "creating CIJobRunTestOutput for test %d", pjrt.ID)
	return o
}

func CreateReleaseDefinition(t *testing.T, dbc *db.DB, release string, major, minor int) models.ReleaseDefinition {
	t.Helper()
	rd := models.ReleaseDefinition{
		Release: release,
		Major:   major,
		Minor:   minor,
		Product: "OCP",
	}
	require.NoError(t, dbc.DB.Create(&rd).Error, "creating ReleaseDefinition %q", release)
	return rd
}

func CreateBug(t *testing.T, dbc *db.DB, key, status, summary string, lastChangeTime time.Time, jobs []models.CIJob) models.Bug {
	t.Helper()
	bug := models.Bug{
		Key:            key,
		Status:         status,
		Summary:        summary,
		LastChangeTime: lastChangeTime,
		Jobs:           jobs,
	}
	require.NoError(t, dbc.DB.Create(&bug).Error, "creating Bug %q", key)
	return bug
}

// CreateBugForTests creates a bug associated with tests via the bug_tests join table
// (as opposed to CreateBug, which associates via bug_jobs). Test report queries
// (openBugsSubquery) count open bugs through this association.
func CreateBugForTests(t *testing.T, dbc *db.DB, key, status, summary string, lastChangeTime time.Time, tests []models.Test) models.Bug {
	t.Helper()
	bug := models.Bug{
		Key:            key,
		Status:         status,
		Summary:        summary,
		LastChangeTime: lastChangeTime,
		Tests:          tests,
	}
	require.NoError(t, dbc.DB.Create(&bug).Error, "creating Bug %q", key)
	return bug
}

type CumulativeSummaryOption func(*models.TestCumulativeSummary)

func WithCumulativeSummaryLifecycle(lifecycle string) CumulativeSummaryOption {
	return func(tcs *models.TestCumulativeSummary) {
		tcs.Lifecycle = lifecycle
	}
}

func WithCumulativeSummaryFailures(failures int64) CumulativeSummaryOption {
	return func(tcs *models.TestCumulativeSummary) {
		tcs.PrefixSumFailures = failures
	}
}

func WithCumulativeSummaryLastFailure(t time.Time) CumulativeSummaryOption {
	return func(tcs *models.TestCumulativeSummary) {
		tcs.PrefixMaxLastFailure = &t
	}
}

func WithCumulativeSummaryLastSuccess(t time.Time) CumulativeSummaryOption {
	return func(tcs *models.TestCumulativeSummary) {
		tcs.PrefixMaxLastSuccess = &t
	}
}

// CreateCumulativeSummary creates a test_cumulative_summaries row directly with the given
// prefix sums, rather than deriving them from raw run data. Prefix sums are cumulative
// totals as of Date: callers computing a period count from two rows (e.g. for dates end
// and boundary) get period_count = end.PrefixSumX - boundary.PrefixSumX. Lifecycle
// defaults to "blocking" (matching the column's DB default) unless overridden.
func CreateCumulativeSummary(t *testing.T, dbc *db.DB, date civil.Date, release string, testID, ciJobID, suiteID uint, runs, successes, flakes int64, opts ...CumulativeSummaryOption) models.TestCumulativeSummary {
	t.Helper()
	tcs := models.TestCumulativeSummary{
		Date:               date,
		Release:            release,
		TestID:             testID,
		CIJobID:          ciJobID,
		SuiteID:            suiteID,
		Lifecycle:          "blocking",
		PrefixSumRuns:      runs,
		PrefixSumSuccesses: successes,
		PrefixSumFlakes:    flakes,
	}
	for _, opt := range opts {
		opt(&tcs)
	}
	require.NoError(t, dbc.DB.Create(&tcs).Error, "creating TestCumulativeSummary for test %d on %s", testID, date)
	return tcs
}

// ReleaseTagOption customizes a ReleaseTag before creation.
type ReleaseTagOption func(*models.ReleaseTag)

func WithPhase(phase string) ReleaseTagOption {
	return func(rt *models.ReleaseTag) { rt.Phase = phase }
}

func WithPreviousOSVersion(version string) ReleaseTagOption {
	return func(rt *models.ReleaseTag) { rt.PreviousOSVersion = version }
}

func WithCurrentOSVersion(version string) ReleaseTagOption {
	return func(rt *models.ReleaseTag) { rt.CurrentOSVersion = version }
}

func WithPreviousReleaseTag(prev string) ReleaseTagOption {
	return func(rt *models.ReleaseTag) { rt.PreviousReleaseTag = prev }
}

func WithForced(forced bool) ReleaseTagOption {
	return func(rt *models.ReleaseTag) { rt.Forced = forced }
}

func CreateReleaseTag(t *testing.T, dbc *db.DB, releaseTag, release, stream, arch string, releaseTime time.Time, opts ...ReleaseTagOption) models.ReleaseTag {
	t.Helper()
	rt := models.ReleaseTag{
		ReleaseTag:   releaseTag,
		Release:      release,
		Stream:       stream,
		Architecture: arch,
		Phase:        apitype.PayloadAccepted,
		ReleaseTime:  releaseTime,
	}
	for _, opt := range opts {
		opt(&rt)
	}
	require.NoError(t, dbc.DB.Create(&rt).Error, "creating ReleaseTag %q", releaseTag)
	return rt
}

type ReleasePullRequestOption func(*models.ReleasePullRequest)

func WithPullRequestID(id string) ReleasePullRequestOption {
	return func(pr *models.ReleasePullRequest) { pr.PullRequestID = id }
}

func WithBugURL(url string) ReleasePullRequestOption {
	return func(pr *models.ReleasePullRequest) { pr.BugURL = url }
}

func CreateReleasePullRequest(t *testing.T, dbc *db.DB, url, name, description string, opts ...ReleasePullRequestOption) models.ReleasePullRequest {
	t.Helper()
	pr := models.ReleasePullRequest{
		URL:         url,
		Name:        name,
		Description: description,
	}
	for _, opt := range opts {
		opt(&pr)
	}
	require.NoError(t, dbc.DB.Create(&pr).Error, "creating ReleasePullRequest %q", url)
	return pr
}

func CreateReleaseJobRun(t *testing.T, dbc *db.DB, releaseTagID, ciJobRunID uint, jobName, kind, state, url string) models.ReleaseJobRun {
	t.Helper()
	rjr := models.ReleaseJobRun{
		ReleaseTagID: fmt.Sprintf("%d", releaseTagID),
		Name:         ciJobRunID,
		JobName:      jobName,
		Kind:         kind,
		State:        state,
		URL:          url,
	}
	require.NoError(t, dbc.DB.Create(&rjr).Error, "creating ReleaseJobRun for tag %d", releaseTagID)
	return rjr
}

func LinkReleaseTagPullRequests(t *testing.T, dbc *db.DB, tag *models.ReleaseTag, prs ...models.ReleasePullRequest) {
	t.Helper()
	prPtrs := make([]*models.ReleasePullRequest, len(prs))
	for i := range prs {
		prPtrs[i] = &prs[i]
	}
	require.NoError(t, dbc.DB.Model(tag).Association("PullRequests").Append(prPtrs), "linking PRs to ReleaseTag %q", tag.ReleaseTag)
}
