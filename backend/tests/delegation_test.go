package tests

import (
	"testing"
	"time"
)

func TestDelegationExpiryLogic(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name      string
		startsAt  time.Time
		expiresAt *time.Time
		wantValid bool
	}{
		{
			name:      "no expiry — always valid after start",
			startsAt:  now.Add(-1 * time.Hour),
			expiresAt: nil,
			wantValid: true,
		},
		{
			name:      "future expiry — currently valid",
			startsAt:  now.Add(-1 * time.Hour),
			expiresAt: timePtr(now.Add(1 * time.Hour)),
			wantValid: true,
		},
		{
			name:      "expired — not valid",
			startsAt:  now.Add(-2 * time.Hour),
			expiresAt: timePtr(now.Add(-1 * time.Hour)),
			wantValid: false,
		},
		{
			name:      "not started yet — not valid",
			startsAt:  now.Add(1 * time.Hour),
			expiresAt: nil,
			wantValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isDelegationActive(tc.startsAt, tc.expiresAt, now)
			if got != tc.wantValid {
				t.Errorf("isDelegationActive() = %v, want %v", got, tc.wantValid)
			}
		})
	}
}

func TestDelegationRoleScope(t *testing.T) {
	grantedRoles := []string{"viewer", "analyst"}

	if !containsRole(grantedRoles, "viewer") {
		t.Error("expected viewer to be in granted roles")
	}
	if containsRole(grantedRoles, "admin") {
		t.Error("expected admin to NOT be in granted roles")
	}
}

func isDelegationActive(startsAt time.Time, expiresAt *time.Time, now time.Time) bool {
	if now.Before(startsAt) {
		return false
	}
	if expiresAt != nil && !now.Before(*expiresAt) {
		return false
	}
	return true
}

func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func timePtr(t time.Time) *time.Time { return &t }
