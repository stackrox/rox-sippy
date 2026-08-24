SELECT
    r.timestamp,
    md.metadata->>'alert' AS "alert",
    md.metadata->>'namespace' AS "namespace",
    md.metadata->>'result' AS "result",
    r.test_failures,
    r.url
FROM
    ci_job_run_test_output_metadata md,
    ci_job_run_test_outputs o,
    ci_job_run_tests rt,
    ci_job_runs r,
    ci_jobs j,
    tests t
WHERE
    md.created_at > NOW() - INTERVAL '14 days' AND
    md.metadata->>'result' != 'allow' AND
    md.ci_job_run_test_output_id = o.id AND
    o.ci_job_run_test_id = rt.id AND
    (rt.status = 12 OR rt.status = 13) AND
    rt.ci_job_run_id = r.id AND
    r.ci_job_id = j.id AND
    (j.release = '4.13' OR j.release = 'Presubmits') AND
    rt.test_id = t.id AND (
        t.name LIKE '%unexpected alerts in firing or pending%' OR
        t.name LIKE '%during or after upgrade%') AND
    /* replace with the alert you're interested in */
    md.metadata->>'alert' = 'TargetDown' AND
    md.metadata->>'namespace' = 'openshift-dns'
ORDER BY r.timestamp DESC
LIMIT 30;

