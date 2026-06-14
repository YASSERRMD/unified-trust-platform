package tests

import (
	"testing"
	"time"
)

func TestJITDurationValidation(t *testing.T) {
	cases := []struct {
		minutes int
		wantErr bool
	}{
		{0, true},
		{-1, true},
		{481, true},
		{1, false},
		{60, false},
		{480, false},
	}

	for _, tc := range cases {
		err := validateDuration(tc.minutes)
		if (err != nil) != tc.wantErr {
			t.Errorf("validateDuration(%d) error = %v, wantErr %v", tc.minutes, err, tc.wantErr)
		}
	}
}

func TestJITGrantExpiry(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name      string
		startsAt  time.Time
		expiresAt time.Time
		revokedAt *time.Time
		wantValid bool
	}{
		{
			name:      "active grant",
			startsAt:  now.Add(-30 * time.Minute),
			expiresAt: now.Add(30 * time.Minute),
			revokedAt: nil,
			wantValid: true,
		},
		{
			name:      "expired grant",
			startsAt:  now.Add(-2 * time.Hour),
			expiresAt: now.Add(-1 * time.Hour),
			revokedAt: nil,
			wantValid: false,
		},
		{
			name:      "revoked grant",
			startsAt:  now.Add(-30 * time.Minute),
			expiresAt: now.Add(30 * time.Minute),
			revokedAt: &now,
			wantValid: false,
		},
		{
			name:      "not started grant",
			startsAt:  now.Add(10 * time.Minute),
			expiresAt: now.Add(70 * time.Minute),
			revokedAt: nil,
			wantValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isGrantActive(tc.startsAt, tc.expiresAt, tc.revokedAt, now)
			if got != tc.wantValid {
				t.Errorf("isGrantActive() = %v, want %v", got, tc.wantValid)
			}
		})
	}
}

func TestJITStatusTransitions(t *testing.T) {
	// Only pending requests can be approved or denied
	validTransitions := map[string][]string{
		"pending":  {"approved", "denied"},
		"approved": {},
		"denied":   {},
		"expired":  {},
		"revoked":  {},
	}

	check := func(from, to string) bool {
		allowed := validTransitions[from]
		for _, a := range allowed {
			if a == to {
				return true
			}
		}
		return false
	}

	if !check("pending", "approved") {
		t.Error("expected pending -> approved to be valid")
	}
	if !check("pending", "denied") {
		t.Error("expected pending -> denied to be valid")
	}
	if check("approved", "denied") {
		t.Error("expected approved -> denied to be invalid")
	}
	if check("denied", "approved") {
		t.Error("expected denied -> approved to be invalid")
	}
}

func validateDuration(minutes int) error {
	if minutes <= 0 || minutes > 480 {
		return errInvalidDuration
	}
	return nil
}

type durationError struct{}

func (e durationError) Error() string { return "duration must be between 1 and 480 minutes" }

var errInvalidDuration = durationError{}

func isGrantActive(startsAt, expiresAt time.Time, revokedAt *time.Time, now time.Time) bool {
	if revokedAt != nil {
		return false
	}
	return now.After(startsAt) && now.Before(expiresAt)
}
