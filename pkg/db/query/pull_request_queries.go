package query

import (
	"time"

	"gorm.io/gorm"

	"github.com/openshift/sippy/pkg/apis/api"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/filter"
)

func PullRequestReport(dbc *db.DB, filterOpts *filter.FilterOptions, release string) ([]api.PullRequest, error) {
	// This finds each PR's first payload for each stream/arch combo, we use it below to join on so we can
	// find the first ci and nightly for each payload.
	firstPayloadsByStreamAndArch := dbc.DB.Table("release_pull_requests").
		Select("url, release_tags.stream, release_tags.architecture, MIN(release_tags.release_time) AS min_release_time, release_tags.release_tag, release_tags.phase, release_tags.release").
		Joins("JOIN release_tag_pull_requests ON release_tag_pull_requests.release_pull_request_id = release_pull_requests.id").
		Joins("INNER JOIN release_tags ON release_tags.id = release_tag_pull_requests.release_tag_id").
		Group("url, release_tags.stream, release_tags.architecture, release_tags.release_tag, release_tags.phase, release_tags.release")

	lookback90d := time.Now().UTC().Add(-90 * 24 * time.Hour)
	prs := dbc.DB.Table("pull_requests").
		Joins("LEFT JOIN (?) ci ON ci.url = pull_requests.link",
			dbc.DB.Table("(?) as ci", firstPayloadsByStreamAndArch).Where("ci.stream = 'ci' AND architecture = 'amd64'")).
		Joins("LEFT JOIN (?) nightly ON nightly.url = pull_requests.link",
			dbc.DB.Table("(?) as nightly", firstPayloadsByStreamAndArch).Where("nightly.stream = 'nightly' AND nightly.architecture = 'amd64'")).
		Joins("INNER JOIN ci_job_run_pull_requests ON ci_job_run_pull_requests.prow_pull_request_id = pull_requests.id").
		Joins("INNER JOIN ci_job_runs on ci_job_run_pull_requests.ci_job_run_id = ci_job_runs.id").
		Joins("INNER JOIN ci_jobs on ci_job_runs.ci_job_id = ci_jobs.id").
		Where("ci_jobs.release = ?", release).
		Where("ci_job_runs.ci_job_release = ?", release).
		Where("ci_job_runs.timestamp > ?", lookback90d).
		Where("ci_job_run_pull_requests.ci_job_run_release = ?", release).
		Where("ci_job_run_pull_requests.ci_job_run_timestamp > ?", lookback90d).
		Select("DISTINCT ON(pull_requests.link) pull_requests.*, ci.release_tag AS first_ci_payload, ci.phase AS first_ci_payload_phase, ci.release as first_ci_payload_release, nightly.release_tag as first_nightly_payload, nightly.phase as first_nightly_payload_phase, nightly.release as first_nightly_payload_release")

	results := make([]api.PullRequest, 0)
	q, err := filter.FilterableDBResult(dbc.DB.Table("(?) as prs", prs), filterOpts, api.PullRequest{})
	if err != nil {
		return results, err
	}
	if err := q.Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func PullRequestAveragePremergeFailures(dbc *db.DB, release string, start, end *time.Time) *gorm.DB {
	premergeFailures := dbc.DB.Table("ci_job_runs").
		Select("ci_jobs.id as ci_job_id, ci_jobs.name as ci_job_name, pull_requests.org, pull_requests.repo, pull_requests.link, COUNT(*) as total_runs").
		Joins("INNER JOIN ci_job_run_pull_requests on ci_job_run_pull_requests.ci_job_run_id = ci_job_runs.id AND ci_job_run_pull_requests.ci_job_run_release = ci_job_runs.ci_job_release").
		Joins("INNER JOIN pull_requests on pull_requests.id = ci_job_run_pull_requests.prow_pull_request_id").
		Joins("INNER JOIN ci_jobs ON ci_job_runs.ci_job_id = ci_jobs.id").
		Where("ci_job_runs.ci_job_release = ?", release).
		Where("ci_job_run_pull_requests.ci_job_run_release = ?", release).
		Where("ci_job_runs.overall_result != 'S'").
		Where("ci_job_runs.overall_result != 'A'").
		Where("pull_requests.merged_at IS NOT NULL").
		Group("ci_jobs.id, ci_jobs.name, pull_requests.org, pull_requests.repo, pull_requests.id, pull_requests.link")

	if start != nil {
		premergeFailures = premergeFailures.
			Where("pull_requests.merged_at >= ?", start).
			Where("ci_job_runs.timestamp >= ?", start).
			Where("ci_job_run_pull_requests.ci_job_run_timestamp >= ?", start)
	}

	if end != nil {
		premergeFailures = premergeFailures.
			Where("pull_requests.merged_at <= ?", end).
			Where("ci_job_runs.timestamp <= ?", end).
			Where("ci_job_run_pull_requests.ci_job_run_timestamp <= ?", end)
	}

	return dbc.DB.Table("(?) as premerge_failures", premergeFailures).
		Select("org, repo, ci_job_id, ci_job_name, AVG(total_runs) as average_premerge_job_failures").
		Group("ci_job_id, ci_job_name, org, repo")
}
