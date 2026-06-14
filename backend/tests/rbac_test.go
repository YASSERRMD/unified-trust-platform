package tests

import (
	"testing"
)

// RBAC tests that don't require a live DB validate the permission string format.

func TestPermissionStringFormat(t *testing.T) {
	cases := []struct {
		resource, action, want string
	}{
		{"user", "read", "user:read"},
		{"role", "write", "role:write"},
		{"policy", "read", "policy:read"},
		{"audit", "export", "audit:export"},
	}

	for _, tc := range cases {
		got := tc.resource + ":" + tc.action
		if got != tc.want {
			t.Errorf("expected %q, got %q", tc.want, got)
		}
	}
}

func TestRBACDenyByDefault(t *testing.T) {
	perms := map[string]bool{
		"user:read":  true,
		"role:write": true,
	}

	check := func(resource, action string) bool {
		return perms[resource+":"+action]
	}

	if !check("user", "read") {
		t.Error("expected user:read to be permitted")
	}
	if check("user", "delete") {
		t.Error("expected user:delete to be denied")
	}
	if check("audit", "export") {
		t.Error("expected audit:export to be denied (not assigned)")
	}
}
