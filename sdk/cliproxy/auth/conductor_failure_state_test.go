package auth

import (
	"net/http"
	"testing"
	"time"
)

func TestApplyAuthFailureStateSchedulesCooldownByStatus(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		status        int
		wantMessage   string
		wantCooldown  time.Duration
		wantQuota     bool
		wantBackoff   int
		wantRecoverAt time.Time
	}{
		{
			name:         "unauthorized",
			status:       http.StatusUnauthorized,
			wantMessage:  "unauthorized",
			wantCooldown: 30 * time.Minute,
		},
		{
			name:         "forbidden",
			status:       http.StatusForbidden,
			wantMessage:  "payment_required",
			wantCooldown: 30 * time.Minute,
		},
		{
			name:         "not found",
			status:       http.StatusNotFound,
			wantMessage:  "not_found",
			wantCooldown: 12 * time.Hour,
		},
		{
			name:         "transient upstream",
			status:       http.StatusBadGateway,
			wantMessage:  "transient upstream error",
			wantCooldown: time.Minute,
		},
		{
			name:          "rate limit backoff",
			status:        http.StatusTooManyRequests,
			wantMessage:   "quota exhausted",
			wantCooldown:  time.Second,
			wantQuota:     true,
			wantBackoff:   1,
			wantRecoverAt: now.Add(time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &Auth{ID: "auth-" + tt.name}

			applyAuthFailureState(auth, &Error{HTTPStatus: tt.status, Message: "upstream error"}, nil, now)

			if !auth.Unavailable {
				t.Fatal("auth.Unavailable = false, want true")
			}
			if auth.Status != StatusError {
				t.Fatalf("auth.Status = %q, want %q", auth.Status, StatusError)
			}
			if auth.StatusMessage != tt.wantMessage {
				t.Fatalf("auth.StatusMessage = %q, want %q", auth.StatusMessage, tt.wantMessage)
			}
			wantNext := now.Add(tt.wantCooldown)
			if !auth.NextRetryAfter.Equal(wantNext) {
				t.Fatalf("auth.NextRetryAfter = %v, want %v", auth.NextRetryAfter, wantNext)
			}
			if auth.Quota.Exceeded != tt.wantQuota {
				t.Fatalf("auth.Quota.Exceeded = %v, want %v", auth.Quota.Exceeded, tt.wantQuota)
			}
			if auth.Quota.BackoffLevel != tt.wantBackoff {
				t.Fatalf("auth.Quota.BackoffLevel = %d, want %d", auth.Quota.BackoffLevel, tt.wantBackoff)
			}
			if !auth.Quota.NextRecoverAt.Equal(tt.wantRecoverAt) {
				t.Fatalf("auth.Quota.NextRecoverAt = %v, want %v", auth.Quota.NextRecoverAt, tt.wantRecoverAt)
			}
		})
	}
}

func TestApplyAuthFailureStateRateLimitUsesRetryAfter(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	retryAfter := 2 * time.Minute
	auth := &Auth{ID: "auth-429"}

	applyAuthFailureState(auth, &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"}, &retryAfter, now)

	wantNext := now.Add(retryAfter)
	if !auth.NextRetryAfter.Equal(wantNext) {
		t.Fatalf("auth.NextRetryAfter = %v, want %v", auth.NextRetryAfter, wantNext)
	}
	if !auth.Quota.Exceeded {
		t.Fatal("auth.Quota.Exceeded = false, want true")
	}
	if auth.Quota.Reason != "quota" {
		t.Fatalf("auth.Quota.Reason = %q, want quota", auth.Quota.Reason)
	}
	if !auth.Quota.NextRecoverAt.Equal(wantNext) {
		t.Fatalf("auth.Quota.NextRecoverAt = %v, want %v", auth.Quota.NextRecoverAt, wantNext)
	}
	if auth.Quota.BackoffLevel != 0 {
		t.Fatalf("auth.Quota.BackoffLevel = %d, want 0 when Retry-After is supplied", auth.Quota.BackoffLevel)
	}
}

func TestApplyAuthFailureStateRateLimitBackoffIncreases(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	auth := &Auth{ID: "auth-429-backoff"}

	applyAuthFailureState(auth, &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"}, nil, now)
	if !auth.NextRetryAfter.Equal(now.Add(time.Second)) {
		t.Fatalf("first NextRetryAfter = %v, want %v", auth.NextRetryAfter, now.Add(time.Second))
	}
	if auth.Quota.BackoffLevel != 1 {
		t.Fatalf("first BackoffLevel = %d, want 1", auth.Quota.BackoffLevel)
	}

	nextNow := now.Add(10 * time.Second)
	applyAuthFailureState(auth, &Error{HTTPStatus: http.StatusTooManyRequests, Message: "quota"}, nil, nextNow)
	if !auth.NextRetryAfter.Equal(nextNow.Add(2 * time.Second)) {
		t.Fatalf("second NextRetryAfter = %v, want %v", auth.NextRetryAfter, nextNow.Add(2*time.Second))
	}
	if auth.Quota.BackoffLevel != 2 {
		t.Fatalf("second BackoffLevel = %d, want 2", auth.Quota.BackoffLevel)
	}
}
