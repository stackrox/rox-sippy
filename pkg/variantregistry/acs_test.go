package variantregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyJob(t *testing.T) {
	tests := []struct {
		name     string
		jobName  string
		branch   string
		ciSystem string
		expected ACSVariant
	}{
		// Unit tests
		{
			name:     "go unit test",
			jobName:  "go-unit-tests",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "unit",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		{
			name:     "unit test with underscore",
			jobName:  "scanner_unit_tests",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "unit",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// QA E2E tests
		{
			name:     "qa-e2e on GCP",
			jobName:  "qa-e2e-tests-gcp",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "qa-e2e",
				CloudProvider: "gcp",
				Release:       "main",
				Framework:     "spock",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		{
			name:     "qa_e2e on AWS",
			jobName:  "qa_e2e_tests_aws",
			branch:   "release-4.9",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "qa-e2e",
				CloudProvider: "aws",
				Release:       "release-4.9",
				Framework:     "spock",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// Go E2E tests
		{
			name:     "go-e2e on Azure",
			jobName:  "e2e-tests-azure",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "azure",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		{
			name:     "go-e2e on GKE",
			jobName:  "e2e-gcp-nightly",
			branch:   "main",
			ciSystem: "prow",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "gcp",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "prow",
				Architecture:  "amd64",
			},
		},
		// Upgrade tests
		{
			name:     "upgrade test",
			jobName:  "upgrade-tests-gcp",
			branch:   "release-4.8",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "upgrade",
				CloudProvider: "gcp",
				Release:       "release-4.8",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// Performance tests
		{
			name:     "perf test",
			jobName:  "perf-tests-large-scale",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "perf",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		{
			name:     "scale test",
			jobName:  "scale-tests-aws",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "perf",
				CloudProvider: "aws",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// UI E2E tests
		{
			name:     "cypress ui test",
			jobName:  "cypress-e2e-tests",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "ui-e2e",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "cypress",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		{
			name:     "ui-e2e test",
			jobName:  "ui-e2e-smoke-tests",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "ui-e2e",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "cypress",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// Compliance tests
		{
			name:     "compliance test",
			jobName:  "compliance-operator-tests",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "compliance",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// Scanner tests
		{
			name:     "scanner test",
			jobName:  "scanner-e2e-tests",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "scanner",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// Architecture variants
		{
			name:     "arm64 test",
			jobName:  "e2e-tests-arm64-gcp",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "gcp",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "arm64",
			},
		},
		{
			name:     "aarch64 test",
			jobName:  "e2e-tests-aarch64-aws",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "aws",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "arm64",
			},
		},
		// Edge cases and defaults
		{
			name:     "unknown job pattern",
			jobName:  "some-random-ci-job",
			branch:   "main",
			ciSystem: "jenkins",
			expected: ACSVariant{
				TestType:      "unknown",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "jenkins",
				Architecture:  "amd64",
			},
		},
		{
			name:     "empty branch defaults to main",
			jobName:  "e2e-tests",
			branch:   "",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		{
			name:     "master branch maps to main",
			jobName:  "e2e-tests",
			branch:   "master",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "none",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// Real-world patterns from StackRox CI
		{
			name:     "qa tests backend pattern",
			jobName:  "qa-tests-backend-nightly-gcp",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "qa-e2e",
				CloudProvider: "gcp",
				Release:       "main",
				Framework:     "spock",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		{
			name:     "openshift-ci e2e pattern",
			jobName:  "periodic-ci-stackrox-e2e-aws",
			branch:   "release-4.7",
			ciSystem: "prow",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "aws",
				Release:       "release-4.7",
				Framework:     "go-test",
				CISystem:      "prow",
				Architecture:  "amd64",
			},
		},
		// Konflux tests
		{
			name:     "konflux ci system",
			jobName:  "e2e-tests-gcp",
			branch:   "main",
			ciSystem: "konflux",
			expected: ACSVariant{
				TestType:      "go-e2e",
				CloudProvider: "gcp",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "konflux",
				Architecture:  "amd64",
			},
		},
		// Multi-pattern matching (upgrade should take precedence)
		{
			name:     "upgrade e2e test",
			jobName:  "upgrade-e2e-tests-aws",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "upgrade",
				CloudProvider: "aws",
				Release:       "main",
				Framework:     "go-test",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
		// QA prefix for Spock framework
		{
			name:     "qa prefix implies Spock",
			jobName:  "qa-compatibility-test-aws",
			branch:   "main",
			ciSystem: "gha",
			expected: ACSVariant{
				TestType:      "qa-e2e",
				CloudProvider: "aws",
				Release:       "main",
				Framework:     "spock",
				CISystem:      "gha",
				Architecture:  "amd64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ClassifyJob(tt.jobName, tt.branch, tt.ciSystem)
			assert.Equal(t, tt.expected.TestType, actual.TestType, "TestType mismatch")
			assert.Equal(t, tt.expected.CloudProvider, actual.CloudProvider, "CloudProvider mismatch")
			assert.Equal(t, tt.expected.Release, actual.Release, "Release mismatch")
			assert.Equal(t, tt.expected.Framework, actual.Framework, "Framework mismatch")
			assert.Equal(t, tt.expected.CISystem, actual.CISystem, "CISystem mismatch")
			assert.Equal(t, tt.expected.Architecture, actual.Architecture, "Architecture mismatch")
		})
	}
}

func TestACSClassifierAdapter(t *testing.T) {
	classifier := NewACSClassifier()

	tests := []struct {
		name     string
		jobName  string
		branch   string
		ciSystem string
		expected map[string]string
	}{
		{
			name:     "basic classification",
			jobName:  "qa-e2e-tests-gcp",
			branch:   "main",
			ciSystem: "gha",
			expected: map[string]string{
				"TestType":      "qa-e2e",
				"CloudProvider": "gcp",
				"Release":       "main",
				"Framework":     "spock",
				"CISystem":      "gha",
				"Architecture":  "amd64",
			},
		},
		{
			name:     "release branch",
			jobName:  "e2e-tests-aws",
			branch:   "release-4.9",
			ciSystem: "prow",
			expected: map[string]string{
				"TestType":      "go-e2e",
				"CloudProvider": "aws",
				"Release":       "release-4.9",
				"Framework":     "go-test",
				"CISystem":      "prow",
				"Architecture":  "amd64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyJob(tt.jobName, tt.branch, tt.ciSystem)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyTestType(t *testing.T) {
	tests := []struct {
		jobName  string
		expected string
	}{
		{"go-unit-tests", "unit"},
		{"scanner_unit_tests", "unit"},
		{"qa-e2e-tests", "qa-e2e"},
		{"qa_e2e_tests", "qa-e2e"},
		{"e2e-tests", "go-e2e"},
		{"upgrade-tests", "upgrade"},
		{"perf-tests", "perf"},
		{"scale-tests", "perf"},
		{"cypress-tests", "ui-e2e"},
		{"ui-e2e-tests", "ui-e2e"},
		{"compliance-tests", "compliance"},
		{"scanner-tests", "scanner"},
		{"unknown-test", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.jobName, func(t *testing.T) {
			result := classifyTestType(tt.jobName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyCloudProvider(t *testing.T) {
	tests := []struct {
		jobName  string
		expected string
	}{
		{"e2e-tests-aws", "aws"},
		{"e2e-tests_aws", "aws"},
		{"e2e-tests-gcp", "gcp"},
		{"e2e-tests_gcp", "gcp"},
		{"e2e-tests-azure", "azure"},
		{"e2e-tests-local", "none"},
		{"unit-tests", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.jobName, func(t *testing.T) {
			result := classifyCloudProvider(tt.jobName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyRelease(t *testing.T) {
	tests := []struct {
		branch   string
		expected string
	}{
		{"main", "main"},
		{"master", "main"},
		{"", "main"},
		{"release-4.7", "release-4.7"},
		{"release-4.9", "release-4.9"},
		{"release-4.10", "release-4.10"},
		{"feature/test-branch", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			result := classifyRelease(tt.branch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyFramework(t *testing.T) {
	tests := []struct {
		jobName  string
		expected string
	}{
		{"qa-e2e-tests", "spock"},
		{"qa_tests_backend", "spock"},
		{"cypress-tests", "cypress"},
		{"ui-cypress-e2e", "cypress"},
		{"e2e-tests", "go-test"},
		{"unit-tests", "go-test"},
		{"integration-tests", "go-test"},
	}

	for _, tt := range tests {
		t.Run(tt.jobName, func(t *testing.T) {
			result := classifyFramework(tt.jobName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyArchitecture(t *testing.T) {
	tests := []struct {
		jobName  string
		expected string
	}{
		{"e2e-tests-arm64", "arm64"},
		{"e2e-tests-aarch64", "arm64"},
		{"e2e-tests-amd64", "amd64"},
		{"e2e-tests-x86_64", "amd64"},
		{"e2e-tests", "amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.jobName, func(t *testing.T) {
			result := classifyArchitecture(tt.jobName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
