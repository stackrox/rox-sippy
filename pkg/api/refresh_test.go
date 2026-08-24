package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockSyncer implements DataSyncer for testing
type mockSyncer struct {
	syncFunc func(ctx context.Context) error
}

func (m *mockSyncer) Sync(ctx context.Context) error {
	if m.syncFunc != nil {
		return m.syncFunc(ctx)
	}
	return nil
}

func TestHandleRefreshRequest_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/refresh", nil)
	w := httptest.NewRecorder()

	syncer := &mockSyncer{
		syncFunc: func(ctx context.Context) error {
			return nil
		},
	}

	HandleRefreshRequest(w, req, syncer)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandleRefreshRequest_RateLimited(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/refresh", nil)
	w := httptest.NewRecorder()

	syncer := &mockSyncer{
		syncFunc: func(ctx context.Context) error {
			return errors.New("rate limited: next sync allowed at 2026-08-24T10:15:00Z")
		},
	}

	HandleRefreshRequest(w, req, syncer)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

func TestHandleRefreshRequest_NoSyncer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/refresh", nil)
	w := httptest.NewRecorder()

	HandleRefreshRequest(w, req, nil)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestHandleRefreshRequest_SyncError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/refresh", nil)
	w := httptest.NewRecorder()

	syncer := &mockSyncer{
		syncFunc: func(ctx context.Context) error {
			return errors.New("connection error")
		},
	}

	HandleRefreshRequest(w, req, syncer)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limited: next sync allowed at 2026-08-24T10:15:00Z"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("connection failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRateLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("isRateLimitError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
