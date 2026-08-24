package bqloader

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	bq "github.com/openshift/sippy/pkg/bigquery"
	"github.com/openshift/sippy/pkg/db/models"
)

const (
	// FirstSyncLookbackDays is the number of days to sync on the first run
	FirstSyncLookbackDays = 90
	// RateLimitCooldown is the minimum time between on-demand sync requests
	RateLimitCooldown = 15 * time.Minute
	// BatchSize for database inserts
	BatchSize = 500
)

// bqJobRow represents a row from the stackrox_jobs BigQuery table.
type bqJobRow struct {
	ID        string     `bigquery:"id"`
	Name      string     `bigquery:"name"`
	Repo      string     `bigquery:"repo"`
	Branch    string     `bigquery:"branch"`
	PRNumber  *int       `bigquery:"pr_number"`
	CommitSHA string     `bigquery:"commit_sha"`
	StartedAt time.Time  `bigquery:"started_at"`
	StoppedAt *time.Time `bigquery:"stopped_at"`
	CISystem  string     `bigquery:"ci_system"`
	Outcome   string     `bigquery:"outcome"`
}

// bqTestRow represents a row from the stackrox_tests BigQuery table.
type bqTestRow struct {
	Name      string    `bigquery:"Name"`
	Classname string    `bigquery:"Classname"`
	JobName   string    `bigquery:"JobName"`
	Status    string    `bigquery:"Status"`
	Timestamp time.Time `bigquery:"Timestamp"`
	BuildTag  string    `bigquery:"BuildTag"`
	Duration  *float64  `bigquery:"Duration"`
}

// JobClassifier provides variant classification for CI jobs.
// This interface allows the BQ loader to remain independent of the specific
// classification logic, which is implemented in pkg/variantregistry.
type JobClassifier interface {
	ClassifyJob(jobName, branch, ciSystem string) map[string]string
}

// BQLoader syncs data from BigQuery to PostgreSQL.
type BQLoader struct {
	db          *gorm.DB
	bqClient    *bq.Client
	projectID   string
	dataset     string
	classifier  JobClassifier
	lastRefresh time.Time
}

// SyncState tracks the last successful sync timestamp for each BQ table.
type SyncState struct {
	ID        uint      `gorm:"primaryKey"`
	TableName string    `gorm:"uniqueIndex;not null"`
	LastSync  time.Time `gorm:"not null"`
	UpdatedAt time.Time
}

// NewBQLoader creates a new BigQuery data loader.
func NewBQLoader(ctx context.Context, db *gorm.DB, bqClient *bq.Client, projectID, dataset string) (*BQLoader, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if bqClient == nil {
		return nil, fmt.Errorf("bqClient cannot be nil")
	}

	// Ensure sync_state table exists
	if err := db.AutoMigrate(&SyncState{}); err != nil {
		return nil, fmt.Errorf("failed to migrate sync_state table: %w", err)
	}

	return &BQLoader{
		db:        db,
		bqClient:  bqClient,
		projectID: projectID,
		dataset:   dataset,
	}, nil
}

// SetClassifier sets the job classifier for variant assignment.
// If not set, jobs will not have variants assigned.
func (l *BQLoader) SetClassifier(classifier JobClassifier) {
	l.classifier = classifier
}

// Sync performs an incremental sync of data from BigQuery to PostgreSQL.
// It enforces rate limiting for on-demand sync requests.
func (l *BQLoader) Sync(ctx context.Context) error {
	// Rate limiting check
	if time.Since(l.lastRefresh) < RateLimitCooldown {
		nextAllowed := l.lastRefresh.Add(RateLimitCooldown)
		return fmt.Errorf("rate limited: next sync allowed at %s", nextAllowed.Format(time.RFC3339))
	}

	log.Info("Starting BigQuery data sync")
	startTime := time.Now()

	// Sync jobs first (tests depend on jobs)
	if err := l.syncJobs(ctx); err != nil {
		return fmt.Errorf("failed to sync jobs: %w", err)
	}

	// Sync tests
	if err := l.syncTests(ctx); err != nil {
		return fmt.Errorf("failed to sync tests: %w", err)
	}

	l.lastRefresh = time.Now()
	log.Infof("BigQuery sync completed in %s", time.Since(startTime))
	return nil
}

// syncJobs syncs job data from stackrox_jobs table.
func (l *BQLoader) syncJobs(ctx context.Context) error {
	log.Info("Syncing jobs from BigQuery")

	// Get last sync timestamp
	lastSync, err := l.getLastSync(ctx, "stackrox_jobs")
	if err != nil {
		return fmt.Errorf("failed to get last sync time: %w", err)
	}

	// Build query for incremental sync
	query := l.bqClient.BQ.Query(fmt.Sprintf(`
		SELECT
			id,
			name,
			repo,
			branch,
			pr_number,
			commit_sha,
			started_at,
			stopped_at,
			ci_system,
			outcome
		FROM %s.%s.stackrox_jobs
		WHERE started_at > @last_sync
		ORDER BY started_at ASC
	`, l.projectID, l.dataset))

	query.Parameters = []bigquery.QueryParameter{
		{Name: "last_sync", Value: lastSync},
	}

	it, err := bq.LoggedRead(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query jobs: %w", err)
	}

	runsProcessed := 0

	for {
		var row bqJobRow

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate jobs: %w", err)
		}

		// Process job and job run
		if err := l.processJobRun(ctx, &row); err != nil {
			log.WithError(err).WithField("job_id", row.ID).Error("Failed to process job run")
			continue
		}

		runsProcessed++
		if runsProcessed%100 == 0 {
			log.Infof("Processed %d job runs", runsProcessed)
		}
	}

	// Update last sync timestamp
	if err := l.updateLastSync(ctx, "stackrox_jobs", time.Now()); err != nil {
		return fmt.Errorf("failed to update last sync time: %w", err)
	}

	log.Infof("Synced %d job runs", runsProcessed)
	return nil
}

// processJobRun creates or updates a job and its associated job run.
func (l *BQLoader) processJobRun(ctx context.Context, row *bqJobRow) error {
	// Determine release from branch
	release := determineRelease(row.Branch)

	// Find or create the CI job
	var job models.CIJob
	result := l.db.Where("name = ? AND release = ?", row.Name, release).First(&job)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new job
			job = models.CIJob{
				Name:     row.Name,
				Release:  release,
				CISystem: row.CISystem,
			}

			// Apply variant classification if classifier is available
			if l.classifier != nil {
				variants := l.classifier.ClassifyJob(row.Name, row.Branch, row.CISystem)
				for k, v := range variants {
					job.Variants = append(job.Variants, fmt.Sprintf("%s:%s", k, v))
				}
			}

			if err := l.db.Create(&job).Error; err != nil {
				return fmt.Errorf("failed to create job: %w", err)
			}
		} else {
			return fmt.Errorf("failed to query job: %w", result.Error)
		}
	}

	// Determine outcome booleans
	succeeded := row.Outcome == "passed" || row.Outcome == "success"
	failed := row.Outcome == "failed" || row.Outcome == "failure"
	infraFailure := row.Outcome == "infra_failure"

	// Create or update job run
	jobRun := models.CIJobRun{
		CIJobID:               job.ID,
		CIJobRelease:          release,
		BQID:                  row.ID,
		CommitSHA:             row.CommitSHA,
		Branch:                row.Branch,
		PRNumber:              row.PRNumber,
		StartedAt:             row.StartedAt,
		StoppedAt:             row.StoppedAt,
		Timestamp:             row.StartedAt, // For compatibility with existing Sippy code
		Repo:                  row.Repo,
		Succeeded:             succeeded,
		Failed:                failed,
		InfrastructureFailure: infraFailure,
	}

	// Upsert job run (ON CONFLICT DO UPDATE on bq_id)
	if err := l.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "bq_id"}},
		UpdateAll: true,
	}).Create(&jobRun).Error; err != nil {
		return fmt.Errorf("failed to upsert job run: %w", err)
	}

	return nil
}

// syncTests syncs test data from stackrox_tests table.
func (l *BQLoader) syncTests(ctx context.Context) error {
	log.Info("Syncing tests from BigQuery")

	// Get last sync timestamp
	lastSync, err := l.getLastSync(ctx, "stackrox_tests")
	if err != nil {
		return fmt.Errorf("failed to get last sync time: %w", err)
	}

	// Build query for incremental sync
	query := l.bqClient.BQ.Query(fmt.Sprintf(`
		SELECT
			Name,
			Classname,
			JobName,
			Status,
			Timestamp,
			BuildTag,
			Duration
		FROM %s.%s.stackrox_tests
		WHERE Timestamp > @last_sync
		ORDER BY Timestamp ASC
	`, l.projectID, l.dataset))

	query.Parameters = []bigquery.QueryParameter{
		{Name: "last_sync", Value: lastSync},
	}

	it, err := bq.LoggedRead(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query tests: %w", err)
	}

	testsProcessed := 0
	batch := make([]*models.CIJobRunTest, 0, BatchSize)

	for {
		var row bqTestRow

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate tests: %w", err)
		}

		// Process test result
		testResult, err := l.processTestResult(ctx, &row)
		if err != nil {
			log.WithError(err).WithField("test_name", row.Name).Error("Failed to process test result")
			continue
		}

		if testResult != nil {
			batch = append(batch, testResult)
		}

		// Flush batch if it reaches batch size
		if len(batch) >= BatchSize {
			if err := l.flushTestBatch(ctx, batch); err != nil {
				return fmt.Errorf("failed to flush test batch: %w", err)
			}
			testsProcessed += len(batch)
			batch = batch[:0]
			log.Infof("Processed %d test results", testsProcessed)
		}
	}

	// Flush remaining batch
	if len(batch) > 0 {
		if err := l.flushTestBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to flush test batch: %w", err)
		}
		testsProcessed += len(batch)
	}

	// Update last sync timestamp
	if err := l.updateLastSync(ctx, "stackrox_tests", time.Now()); err != nil {
		return fmt.Errorf("failed to update last sync time: %w", err)
	}

	log.Infof("Synced %d test results", testsProcessed)
	return nil
}

// processTestResult processes a single test result from BigQuery.
func (l *BQLoader) processTestResult(ctx context.Context, row *bqTestRow) (*models.CIJobRunTest, error) {
	// Find or create test
	var test models.Test
	result := l.db.Where("name = ?", row.Name).First(&test)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			test = models.Test{Name: row.Name}
			if err := l.db.Create(&test).Error; err != nil {
				return nil, fmt.Errorf("failed to create test: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to query test: %w", result.Error)
		}
	}

	// Find or create suite from Classname
	var suite models.Suite
	var suiteID *uint
	if row.Classname != "" {
		result = l.db.Where("name = ?", row.Classname).First(&suite)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				suite = models.Suite{Name: row.Classname}
				if err := l.db.Create(&suite).Error; err != nil {
					return nil, fmt.Errorf("failed to create suite: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to query suite: %w", result.Error)
			}
		}
		suiteID = &suite.ID
	}

	// Find the job run by BuildTag (which should match the BQ ID)
	var jobRun models.CIJobRun
	if err := l.db.Where("bq_id = ?", row.BuildTag).First(&jobRun).Error; err != nil {
		// If we can't find the job run, log and skip this test result
		log.WithError(err).WithField("build_tag", row.BuildTag).Warn("Job run not found for test result")
		return nil, nil
	}

	// Map status string to integer code
	statusCode := mapTestStatus(row.Status)

	// Create test result
	var duration float64
	if row.Duration != nil {
		duration = *row.Duration
	}

	testResult := &models.CIJobRunTest{
		CIJobRunID:        jobRun.ID,
		CIJobID:           jobRun.CIJobID,
		CIJobRunTimestamp: jobRun.Timestamp,
		CIJobRunRelease:   jobRun.CIJobRelease,
		TestID:            test.ID,
		SuiteID:           suiteID,
		Status:            statusCode,
		Duration:          duration,
	}

	return testResult, nil
}

// flushTestBatch inserts a batch of test results into the database.
func (l *BQLoader) flushTestBatch(ctx context.Context, batch []*models.CIJobRunTest) error {
	if len(batch) == 0 {
		return nil
	}

	// Use GORM's CreateInBatches for efficient insertion
	if err := l.db.CreateInBatches(batch, BatchSize).Error; err != nil {
		return fmt.Errorf("failed to create test results batch: %w", err)
	}

	return nil
}

// getLastSync retrieves the last sync timestamp for a table.
// If no sync has occurred, returns a timestamp N days ago.
func (l *BQLoader) getLastSync(ctx context.Context, tableName string) (time.Time, error) {
	var state SyncState
	result := l.db.Where("table_name = ?", tableName).First(&state)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// First sync - go back N days
			return time.Now().AddDate(0, 0, -FirstSyncLookbackDays), nil
		}
		return time.Time{}, fmt.Errorf("failed to query sync state: %w", result.Error)
	}

	return state.LastSync, nil
}

// updateLastSync updates the last sync timestamp for a table.
func (l *BQLoader) updateLastSync(ctx context.Context, tableName string, syncTime time.Time) error {
	state := SyncState{
		TableName: tableName,
		LastSync:  syncTime,
	}

	// Upsert sync state
	if err := l.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "table_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_sync", "updated_at"}),
	}).Create(&state).Error; err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	return nil
}

// determineRelease extracts the release from a branch name.
func determineRelease(branch string) string {
	if branch == "main" || branch == "master" || branch == "" {
		return "main"
	}
	// Handle release-X.Y format
	if len(branch) > 8 && branch[:8] == "release-" {
		return branch
	}
	return "main"
}

// mapTestStatus converts BQ status strings to Sippy status codes.
func mapTestStatus(status string) int {
	switch status {
	case "pass", "passed", "success":
		return 0 // pass
	case "fail", "failed", "failure":
		return 1 // fail
	case "flake", "flaky":
		return 12 // flake
	default:
		log.Warnf("Unknown test status: %s, defaulting to fail", status)
		return 1 // default to fail
	}
}
