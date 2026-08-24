-- Drop the GIN index on ci_jobs.variants. All variant filtering now
-- goes through variant_combination_id lookups against the small
-- variant_combinations table, so the GIN index is unused.
DROP INDEX IF EXISTS idx_ci_jobs_variants;
