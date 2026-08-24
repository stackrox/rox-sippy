package jobrunintervals

import (
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/sippy/pkg/api"
	apitype "github.com/openshift/sippy/pkg/apis/api"
	"github.com/openshift/sippy/pkg/db"
)

// JobRunIntervals fetches intervals for a given job run by fetching from the prow job's GCS bucket path
// constructed using the prow job name and jobID using one of these methods:
// 1) using a GCS path that was calculated and passed in (we can retrieve intervals immediately)
// 2) looking up the url given the jobRunID and extracting the prow job name (we need to wait until the sippyDB is populated)
// If the GCS path could not be calculated, it will be empty.
func JobRunIntervals(gcsClient *storage.Client, dbc *db.DB, jobRunID int64, gcsBucket, gcsPath, intervalFile string, logger *log.Entry) (*apitype.EventIntervalList, error) {

	jobRunURL := fmt.Sprintf("https://gcsweb.acs.redhat.com/gcs/%s/%s", gcsBucket, gcsPath)

	jobRun, err := api.FetchJobRun(dbc, jobRunID, false, nil, logger)
	if err != nil {
		// some jobs are not in the DB, and usually we have bucket/path without looking them up
		logger.WithError(err).Debugf("failed to fetch job run %d", jobRunID)
		if gcsPath == "" {
			return nil, errors.New("no GCS path given and no job run found in DB")
		}
	} else {
		jobRunURL = jobRun.URL
		gcsBucket = jobRun.GCSBucket // in theory jobs might someday come from more than one bucket
		_, path, found := strings.Cut(jobRunURL, "/"+gcsBucket+"/")
		if !found {
			return nil, fmt.Errorf("job run URL %q does not contain bucket %q", jobRun.URL, gcsBucket)
		}
		gcsPath = path
	}

	// TODO(ACS): GCS interval extraction removed (OCP-specific prowloader/gcs package deleted)
	logger.Info("GCS interval extraction not available (OCP-specific feature removed)")
	intervals := &apitype.EventIntervalList{}
	return intervals, nil
}
