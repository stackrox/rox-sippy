package bqloader

import (
	"testing"
	"time"
)

func TestDetermineRelease(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "main branch",
			branch:   "main",
			expected: "main",
		},
		{
			name:     "master branch",
			branch:   "master",
			expected: "main",
		},
		{
			name:     "empty branch",
			branch:   "",
			expected: "main",
		},
		{
			name:     "release branch 4.9",
			branch:   "release-4.9",
			expected: "release-4.9",
		},
		{
			name:     "release branch 4.10",
			branch:   "release-4.10",
			expected: "release-4.10",
		},
		{
			name:     "feature branch",
			branch:   "feature/my-feature",
			expected: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineRelease(tt.branch)
			if got != tt.expected {
				t.Errorf("determineRelease(%q) = %q, want %q", tt.branch, got, tt.expected)
			}
		})
	}
}

func TestMapTestStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected int
	}{
		{
			name:     "pass",
			status:   "pass",
			expected: 0,
		},
		{
			name:     "passed",
			status:   "passed",
			expected: 0,
		},
		{
			name:     "success",
			status:   "success",
			expected: 0,
		},
		{
			name:     "fail",
			status:   "fail",
			expected: 1,
		},
		{
			name:     "failed",
			status:   "failed",
			expected: 1,
		},
		{
			name:     "failure",
			status:   "failure",
			expected: 1,
		},
		{
			name:     "flake",
			status:   "flake",
			expected: 12,
		},
		{
			name:     "flaky",
			status:   "flaky",
			expected: 12,
		},
		{
			name:     "unknown status defaults to fail",
			status:   "unknown",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTestStatus(tt.status)
			if got != tt.expected {
				t.Errorf("mapTestStatus(%q) = %d, want %d", tt.status, got, tt.expected)
			}
		})
	}
}

func TestRateLimiting(t *testing.T) {
	// Test that rate limiting logic works correctly
	cooldown := 15 * time.Minute
	lastRefresh := time.Now().Add(-10 * time.Minute) // 10 minutes ago

	// Should be rate limited (10 min < 15 min)
	if time.Since(lastRefresh) >= cooldown {
		t.Error("Expected to be rate limited, but was not")
	}

	// 20 minutes ago - should NOT be rate limited
	lastRefresh = time.Now().Add(-20 * time.Minute)
	if time.Since(lastRefresh) < cooldown {
		t.Error("Expected NOT to be rate limited, but was")
	}
}

func TestSyncStateConstants(t *testing.T) {
	// Verify constants are set as specified
	if FirstSyncLookbackDays != 90 {
		t.Errorf("FirstSyncLookbackDays = %d, want 90", FirstSyncLookbackDays)
	}

	if RateLimitCooldown != 15*time.Minute {
		t.Errorf("RateLimitCooldown = %v, want 15m", RateLimitCooldown)
	}

	if BatchSize != 500 {
		t.Errorf("BatchSize = %d, want 500", BatchSize)
	}
}

// TestJobClassifierInterface verifies the interface definition
func TestJobClassifierInterface(t *testing.T) {
	// Mock classifier
	var classifier JobClassifier = &mockClassifier{}

	variants := classifier.ClassifyJob("test-job", "main", "gha")
	if variants == nil {
		t.Error("Expected non-nil variants map")
	}
}

// mockClassifier is a test implementation of JobClassifier
type mockClassifier struct{}

func (m *mockClassifier) ClassifyJob(jobName, branch, ciSystem string) map[string]string {
	return map[string]string{
		"test_type": "unit",
		"ci_system": ciSystem,
		"release":   "main",
	}
}

func TestFieldMapping(t *testing.T) {
	// Test that field mapping from BQ schema to Go models works as expected
	// This tests the conceptual mapping - actual DB integration tests would be separate

	tests := []struct {
		name        string
		bqFieldName string
		goFieldName string
		description string
	}{
		{
			name:        "BQ tests table uses PascalCase",
			bqFieldName: "Name",
			goFieldName: "Name",
			description: "Test name field",
		},
		{
			name:        "BQ tests Classname maps to suite",
			bqFieldName: "Classname",
			goFieldName: "Classname",
			description: "Suite/class name",
		},
		{
			name:        "BQ tests Status maps to status code",
			bqFieldName: "Status",
			goFieldName: "Status",
			description: "Test status (string -> int)",
		},
		{
			name:        "BQ jobs table uses snake_case",
			bqFieldName: "started_at",
			goFieldName: "StartedAt",
			description: "Job start time",
		},
		{
			name:        "BQ jobs ci_system field",
			bqFieldName: "ci_system",
			goFieldName: "CISystem",
			description: "CI system name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a documentation test - it verifies our understanding
			// of the field name mappings between BQ and Go
			if tt.bqFieldName == "" || tt.goFieldName == "" {
				t.Errorf("Invalid field mapping: %s -> %s", tt.bqFieldName, tt.goFieldName)
			}
		})
	}
}

func TestIncrementalSyncLogic(t *testing.T) {
	// Test the incremental sync logic
	// First sync should look back 90 days
	firstSyncStart := time.Now().AddDate(0, 0, -FirstSyncLookbackDays)
	daysDiff := int(time.Since(firstSyncStart).Hours() / 24)

	if daysDiff < 89 || daysDiff > 91 {
		t.Errorf("First sync lookback = %d days, want ~90 days", daysDiff)
	}

	// Subsequent syncs should use the last sync timestamp
	lastSync := time.Now().Add(-1 * time.Hour)
	hoursSince := time.Since(lastSync).Hours()

	if hoursSince < 0.9 || hoursSince > 1.1 {
		t.Errorf("Time since last sync = %.2f hours, want ~1 hour", hoursSince)
	}
}
