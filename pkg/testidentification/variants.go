package testidentification

import (
	"context"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// NeverStable is a variant for jobs that are known to be perpetually failing
	NeverStable = "never-stable"
)

// VariantManager identifies and describes different variants
type VariantManager interface {
	// AllPlatforms returns a set of all known platform variants
	AllPlatforms() sets.Set[string]

	// IdentifyVariants takes a job name and returns the list of variants that job belongs to.
	IdentifyVariants(jobName string) []string

	// IsJobNeverStable returns true if the job has been curated as never having passed more than 50ish% of the time.
	// This is used sparingly for jobs that are persistently failing and never taken stable.
	IsJobNeverStable(jobName string) bool
}

// NewOpenshiftVariantManager creates a variant manager (stub - returns empty manager for ACS)
// TODO(ACS): Replace with ACS-specific variant manager
func NewOpenshiftVariantManager(ctx context.Context, bqc interface{}) (VariantManager, error) {
	return NewEmptyVariantManager(), nil
}
