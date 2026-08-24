package query

import (
	"time"

	"gorm.io/gorm"

	"github.com/openshift/sippy/pkg/apis/api"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/filter"
)

func RepositoryReport(dbc *db.DB, filterOpts *filter.FilterOptions, release string, reportEnd time.Time) ([]api.Repository, error) {
	end := reportEnd

	premergeFailureStart := reportEnd.Add(-14 * 24 * time.Hour)
	averageByJob := PullRequestAveragePremergeFailures(dbc, release, &premergeFailureStart, &end)

	revertCountStart := reportEnd.Add(-90 * 24 * time.Hour)
	revertCount := RepositoryRevertCount(dbc, &revertCountStart, &end)

	repos := dbc.DB.Table("pull_requests").
		Joins("INNER JOIN ci_job_run_pull_requests ON ci_job_run_pull_requests.prow_pull_request_id = pull_requests.id").
		Joins("INNER JOIN ci_job_runs on ci_job_run_pull_requests.ci_job_run_id = ci_job_runs.id").
		Joins("INNER JOIN ci_jobs on ci_job_runs.ci_job_id = ci_jobs.id").
		Joins("LEFT JOIN (?) revert_count ON revert_count.org = pull_requests.org AND revert_count.repo = pull_requests.repo", revertCount).
		Joins("LEFT JOIN (?) premerge_failures ON premerge_failures.ci_job_ID = ci_jobs.id", averageByJob).
		Where("ci_jobs.release = ?", release).
		Where("ci_job_runs.ci_job_release = ?", release).
		Where("ci_job_runs.timestamp >= ? AND ci_job_runs.timestamp < ?", revertCountStart, reportEnd).
		Where("ci_job_run_pull_requests.ci_job_run_release = ?", release).
		Where("ci_job_run_pull_requests.ci_job_run_timestamp >= ? AND ci_job_run_pull_requests.ci_job_run_timestamp < ?", revertCountStart, reportEnd).
		Group("pull_requests.org, pull_requests.repo").
		Select("ROW_NUMBER() OVER() as id, pull_requests.org, pull_requests.repo, max(revert_count) as revert_count, coalesce(max(average_premerge_job_failures), 0) as worst_premerge_job_failures, count(distinct(ci_jobs.id)) as job_count")

	results := make([]api.Repository, 0)
	q, err := filter.FilterableDBResult(dbc.DB.Table("(?) as repos", repos), filterOpts, api.Repository{})
	if err != nil {
		return results, err
	}
	if err := q.Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func RepositoryRevertCount(dbc *db.DB, start, end *time.Time) *gorm.DB {
	query := dbc.DB.Table("pull_requests").
		Where("title ILIKE '%Revert%'").
		Where("title NOT ILIKE '%Unrevert%'").
		Group("org, repo").
		Select("org, repo, COUNT(DISTINCT link) AS revert_count")

	if start != nil {
		query = query.Where("pull_requests.merged_at >= ?", start)
	}

	if end != nil {
		query = query.Where("pull_requests.merged_at <= ?", end)
	}

	return query
}
