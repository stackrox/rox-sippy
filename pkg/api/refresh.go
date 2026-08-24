package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// DataSyncer represents a component that can sync data from an external source.
type DataSyncer interface {
	Sync(ctx context.Context) error
}

// RefreshResponse represents the JSON response for the /api/refresh endpoint.
type RefreshResponse struct {
	Status      string     `json:"status"`       // "started" or "rate_limited"
	LastRefresh *time.Time `json:"last_refresh"` // when last sync completed
	NextAllowed *time.Time `json:"next_allowed"` // when next sync is allowed (if rate-limited)
	Message     string     `json:"message"`      // human-readable message
}

// HandleRefreshRequest triggers an on-demand data sync with rate limiting.
// Returns 200 if sync started, 429 if rate-limited.
func HandleRefreshRequest(w http.ResponseWriter, req *http.Request, syncer DataSyncer) {
	if syncer == nil {
		RespondWithJSON(http.StatusServiceUnavailable, w, RefreshResponse{
			Status:  "unavailable",
			Message: "Data sync is not configured for this instance",
		})
		return
	}

	ctx := req.Context()
	err := syncer.Sync(ctx)

	if err != nil {
		// Check if it's a rate limit error
		if isRateLimitError(err) {
			// Extract next allowed time from error message if possible
			RespondWithJSON(http.StatusTooManyRequests, w, RefreshResponse{
				Status:  "rate_limited",
				Message: fmt.Sprintf("Sync is rate-limited. %s", err.Error()),
			})
			return
		}

		// Other errors
		log.WithError(err).Error("Failed to trigger data sync")
		RespondWithError(w, "Failed to trigger data sync", err)
		return
	}

	// Sync started successfully
	now := time.Now()
	RespondWithJSON(http.StatusOK, w, RefreshResponse{
		Status:      "started",
		LastRefresh: &now,
		Message:     "Sync started. Data will be available in ~2-5 minutes.",
	})
}

// isRateLimitError checks if an error is a rate limit error based on message content.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return len(msg) > 11 && msg[:12] == "rate limited"
}
