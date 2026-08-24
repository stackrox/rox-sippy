-- Create materialized views for ACS Sippy analytics
--
-- test_daily_summary: Daily aggregated test pass rates (partitioned by release + date)
-- test_release_summary: Cumulative test summary per release (for Component Readiness)
--
-- Both views require unique indexes for REFRESH MATERIALIZED VIEW CONCURRENTLY

-- ============================================================================
-- 1. test_daily_summary
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS test_daily_summary AS
SELECT
    t.id AS test_id,
    j.release,
    DATE(jr.timestamp AT TIME ZONE 'UTC') AS day,
    j.ci_system,
    COUNT(*) FILTER (WHERE jrt.status = 1) AS passes,
    COUNT(*) FILTER (WHERE jrt.status = 12) AS failures,
    COUNT(*) FILTER (WHERE jrt.status = 13) AS flakes,
    COUNT(*) AS total_runs
FROM ci_job_run_tests jrt
JOIN ci_job_runs jr ON jr.id = jrt.ci_job_run_id
JOIN ci_jobs j ON j.id = jr.ci_job_id
JOIN tests t ON t.id = jrt.test_id
GROUP BY t.id, j.release, DATE(jr.timestamp AT TIME ZONE 'UTC'), j.ci_system
WITH NO DATA;

-- Required unique index for REFRESH CONCURRENTLY
CREATE UNIQUE INDEX IF NOT EXISTS idx_test_daily_summary_unique
    ON test_daily_summary (test_id, release, day, ci_system);

-- Additional indexes for query performance
CREATE INDEX IF NOT EXISTS idx_test_daily_summary_day
    ON test_daily_summary (day);

CREATE INDEX IF NOT EXISTS idx_test_daily_summary_release
    ON test_daily_summary (release);

-- ============================================================================
-- 2. test_release_summary
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS test_release_summary AS
SELECT
    t.id AS test_id,
    t.name AS test_name,
    j.release,
    j.variants,
    COUNT(*) FILTER (WHERE jrt.status = 1) AS passes,
    COUNT(*) FILTER (WHERE jrt.status = 12) AS failures,
    COUNT(*) FILTER (WHERE jrt.status = 13) AS flakes,
    COUNT(*) AS total_runs,
    MIN(jr.timestamp) AS first_run,
    MAX(jr.timestamp) AS last_run
FROM ci_job_run_tests jrt
JOIN ci_job_runs jr ON jr.id = jrt.ci_job_run_id
JOIN ci_jobs j ON j.id = jr.ci_job_id
JOIN tests t ON t.id = jrt.test_id
GROUP BY t.id, t.name, j.release, j.variants
WITH NO DATA;

-- Required unique index for REFRESH CONCURRENTLY
CREATE UNIQUE INDEX IF NOT EXISTS idx_test_release_summary_unique
    ON test_release_summary (test_id, release, variants);

-- Additional indexes for query performance
CREATE INDEX IF NOT EXISTS idx_test_release_summary_release
    ON test_release_summary (release);
