package variantregistry

import (
	"regexp"
	"strings"
)

// ACSVariant represents the variant dimensions for an ACS CI job.
// All fields are guaranteed to be non-empty; unknown patterns receive default values.
type ACSVariant struct {
	TestType      string // unit, qa-e2e, go-e2e, upgrade, perf, ui-e2e, compliance, scanner, unknown
	CloudProvider string // gcp, aws, azure, none
	Release       string // main, release-4.X
	Framework     string // go-test, spock, cypress
	CISystem      string // gha, prow, konflux, jenkins, etc.
	Architecture  string // amd64, arm64
}

// ClassifyJob assigns variant dimensions to a CI job based on its name,
// branch, and CI system metadata.
//
// PRECONDITION: jobName is non-empty
// POSTCONDITION: All 6 variant fields populated (defaults for unrecognized patterns)
// ERROR: Never errors — unknown patterns get default values
func ClassifyJob(jobName, branch, ciSystem string) ACSVariant {
	v := ACSVariant{
		Release:  classifyRelease(branch),
		CISystem: ciSystem,
	}
	v.TestType = classifyTestType(jobName)
	v.CloudProvider = classifyCloudProvider(jobName)
	v.Framework = classifyFramework(jobName)
	v.Architecture = classifyArchitecture(jobName)
	return v
}

// classifyTestType determines the test type from the job name.
func classifyTestType(jobName string) string {
	lower := strings.ToLower(jobName)

	// Order matters: more specific patterns first
	if containsPattern(lower, "unit") {
		return "unit"
	}
	// QA tests: check for qa- prefix, qa_ prefix, or "qa" at start of job name
	if containsPattern(lower, "qa-e2e", "qa_e2e") || startsWithPattern(lower, "qa-", "qa_") {
		return "qa-e2e"
	}
	if containsPattern(lower, "upgrade") {
		return "upgrade"
	}
	if containsPattern(lower, "cypress", "ui-e2e") {
		return "ui-e2e"
	}
	if containsPattern(lower, "compliance") {
		return "compliance"
	}
	if containsPattern(lower, "scanner") {
		return "scanner"
	}
	if containsPattern(lower, "perf", "scale") {
		return "perf"
	}
	// Generic e2e (not qa-e2e)
	if containsPattern(lower, "e2e") {
		return "go-e2e"
	}

	return "unknown"
}

// classifyCloudProvider determines the cloud provider from the job name.
func classifyCloudProvider(jobName string) string {
	lower := strings.ToLower(jobName)

	if containsPattern(lower, "-aws", "_aws") {
		return "aws"
	}
	if containsPattern(lower, "-gcp", "_gcp", "-gke", "_gke") {
		return "gcp"
	}
	if containsPattern(lower, "-azure", "_azure") {
		return "azure"
	}

	return "none"
}

// classifyRelease extracts the release from the branch name.
func classifyRelease(branch string) string {
	if branch == "main" || branch == "master" || branch == "" {
		return "main"
	}

	// Match release-X.Y format
	releasePattern := regexp.MustCompile(`^release-\d+\.\d+$`)
	if releasePattern.MatchString(branch) {
		return branch
	}

	// Default to main for feature branches or other patterns
	return "main"
}

// classifyFramework determines the test framework from the job name.
func classifyFramework(jobName string) string {
	lower := strings.ToLower(jobName)

	// QA tests use Spock/Groovy framework
	if containsPattern(lower, "qa-", "qa_") || startsWithPattern(lower, "qa-", "qa_") {
		return "spock"
	}

	// Cypress for UI tests or ui-e2e
	if containsPattern(lower, "cypress", "ui-e2e") {
		return "cypress"
	}

	// Default to Go test framework
	return "go-test"
}

// classifyArchitecture determines the architecture from the job name.
func classifyArchitecture(jobName string) string {
	lower := strings.ToLower(jobName)

	if containsPattern(lower, "arm64", "aarch64") {
		return "arm64"
	}

	// Default to amd64 (most common)
	return "amd64"
}

// containsPattern checks if the string contains any of the given patterns.
func containsPattern(s string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

// startsWithPattern checks if the string starts with any of the given patterns.
func startsWithPattern(s string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.HasPrefix(s, pattern) {
			return true
		}
	}
	return false
}

// ACSClassifier implements the JobClassifier interface for use with the BQ loader.
// It wraps the ACSVariant return into the map[string]string format.
type ACSClassifier struct{}

// NewACSClassifier creates a new ACS job classifier.
func NewACSClassifier() *ACSClassifier {
	return &ACSClassifier{}
}

// ClassifyJob implements the JobClassifier interface.
func (c *ACSClassifier) ClassifyJob(jobName, branch, ciSystem string) map[string]string {
	variant := ClassifyJob(jobName, branch, ciSystem)
	return map[string]string{
		"TestType":      variant.TestType,
		"CloudProvider": variant.CloudProvider,
		"Release":       variant.Release,
		"Framework":     variant.Framework,
		"CISystem":      variant.CISystem,
		"Architecture":  variant.Architecture,
	}
}
