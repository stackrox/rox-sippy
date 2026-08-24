package variantregistry

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/openshift/sippy/pkg/bigquery/bqlabel"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/openshift/sippy/pkg/apis/api/componentreport/crview"
	v1 "github.com/openshift/sippy/pkg/apis/config/v1"
	"github.com/openshift/sippy/pkg/releaseoverride"
)

// JobVariants is a map of jobs to variant key/value pairs
type JobVariants map[string]map[string]string

// OCPVariantLoader is a stub for the deleted OCP variant loader
type OCPVariantLoader struct {
	config                       *v1.SippyConfig
	views                        []crview.View
	syntheticReleaseJobOverrides *releaseoverride.SyntheticReleaseOverrides
}

// CalculateVariantsForJob calculates variants for a job (stub - returns empty map)
func (o *OCPVariantLoader) CalculateVariantsForJob(log logrus.FieldLogger, job string, variants interface{}) map[string]string {
	// TODO(ACS): Replace with ACS-specific variant calculation
	return map[string]string{}
}

// validateComponentCapabilityVariants validates component capability variants (stub - always returns nil)
func validateComponentCapabilityVariants(job string, variants map[string]string) error {
	// TODO(ACS): Add ACS-specific validation if needed
	return nil
}

type VariantSnapshot struct {
	config                       *v1.SippyConfig
	views                        []crview.View
	syntheticReleaseJobOverrides *releaseoverride.SyntheticReleaseOverrides
	log                          logrus.FieldLogger
}

func NewVariantSnapshot(config *v1.SippyConfig, views []crview.View, syntheticReleaseJobOverrides *releaseoverride.SyntheticReleaseOverrides, log logrus.FieldLogger) *VariantSnapshot {
	return &VariantSnapshot{
		config:                       config,
		views:                        views,
		syntheticReleaseJobOverrides: syntheticReleaseJobOverrides,
		log:                          log,
	}
}

func (s *VariantSnapshot) Identify() (JobVariants, error) {
	newVariants := map[string]map[string]string{}
	variantSyncer := OCPVariantLoader{config: s.config, views: s.views, syntheticReleaseJobOverrides: s.syntheticReleaseJobOverrides}
	var errs []string
	for _, releaseCfg := range s.config.Releases {
		for job := range releaseCfg.Jobs {
			if isIgnoredJob(job) {
				continue
			}
			if _, done := newVariants[job]; done {
				continue
			}
			variants := variantSyncer.CalculateVariantsForJob(s.log, job, nil)
			newVariants[job] = variants
			if err := validateComponentCapabilityVariants(job, variants); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return nil, fmt.Errorf("variant registry validation failed:\n%s", strings.Join(errs, "\n"))
	}

	return newVariants, nil
}

func (s *VariantSnapshot) Load(path string) (JobVariants, error) {
	oldVariants := map[string]map[string]string{}
	oldVariantsYAML, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(oldVariantsYAML, &oldVariants); err != nil {
		return nil, err
	}

	return oldVariants, nil
}

func (s *VariantSnapshot) Save(path string) error {
	newVariants, err := s.Identify()
	if err != nil {
		return err
	}
	y, err := yaml.Marshal(newVariants)
	if err != nil {
		return err
	}
	return os.WriteFile(path, y, 0o600)
}

func isIgnoredJob(jobName string) bool {
	// Some CI jobs don't have stable variants because they move between OCP release versions, or other
	// reasons
	ignoredJobSubstrings := []string{
		"periodic-ci-openshift-hive-master-",
	}
	for _, entry := range ignoredJobSubstrings {
		if strings.Contains(jobName, entry) {
			return true
		}
	}

	return false
}

// NewOCPVariantLoader creates a new OCP variant loader (stub for ACS)
// TODO(ACS): Replace with ACS-specific variant loader implementation
func NewOCPVariantLoader(
	bigQueryClient *bigquery.Client,
	opCtx bqlabel.OperationalContext,
	bigQueryProject, bigQueryDataSet, bigQueryTable string,
	gcsClient *storage.Client,
	config *v1.SippyConfig,
	views []crview.View,
	syntheticReleaseJobOverrides *releaseoverride.SyntheticReleaseOverrides,
) *OCPVariantLoader {
	return &OCPVariantLoader{
		config:                       config,
		views:                        views,
		syntheticReleaseJobOverrides: syntheticReleaseJobOverrides,
	}
}

// LoadExpectedJobVariants loads expected job variants (stub - returns empty map)
// TODO(ACS): Implement ACS-specific job variant loading from BigQuery
func (o *OCPVariantLoader) LoadExpectedJobVariants(ctx context.Context) (map[string]map[string]string, error) {
	return map[string]map[string]string{}, nil
}
