CREATE INDEX IF NOT EXISTS idx_ci_jobs_variants ON ci_jobs USING gin (variants);
