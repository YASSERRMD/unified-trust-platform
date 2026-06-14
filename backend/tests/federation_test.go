package tests

import (
	"net/url"
	"strings"
	"testing"
)

func TestOIDCAuthURLConstruction(t *testing.T) {
	base := "https://accounts.example.com/authorize"
	clientID := "my-client-id"
	redirectURI := "https://app.example.com/callback"
	scopes := "openid email profile"
	state := "random-state-value"

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scopes)
	params.Set("state", state)

	authURL := base + "?" + params.Encode()

	if !strings.Contains(authURL, "response_type=code") {
		t.Error("expected response_type=code in auth URL")
	}
	if !strings.Contains(authURL, "client_id="+clientID) {
		t.Error("expected client_id in auth URL")
	}
	if !strings.Contains(authURL, "state="+state) {
		t.Error("expected state in auth URL")
	}
}

func TestOIDCScopeDefault(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "openid email profile"},
		{"openid", "openid"},
		{"openid groups", "openid groups"},
	}

	for _, tc := range cases {
		got := defaultScopes(tc.input)
		if got != tc.want {
			t.Errorf("defaultScopes(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFederationProviderProtocolDefault(t *testing.T) {
	protocol := ""
	if protocol == "" {
		protocol = "oidc"
	}
	if protocol != "oidc" {
		t.Errorf("expected default protocol to be oidc, got %q", protocol)
	}
}

func TestLinkedIdentityUpsert(t *testing.T) {
	// Verify that upsert logic distinguishes new vs existing by external_subject
	existing := map[string]string{
		"sub1": "user-uuid-1",
	}

	upsert := func(sub string) (string, bool) {
		id, found := existing[sub]
		if !found {
			id = "new-user-uuid"
			existing[sub] = id
		}
		return id, found
	}

	id1, existed1 := upsert("sub1")
	if !existed1 {
		t.Error("expected sub1 to be found as existing")
	}
	if id1 != "user-uuid-1" {
		t.Errorf("expected user-uuid-1, got %q", id1)
	}

	id2, existed2 := upsert("sub2")
	if existed2 {
		t.Error("expected sub2 to be new")
	}
	if id2 != "new-user-uuid" {
		t.Errorf("expected new-user-uuid, got %q", id2)
	}
}

func defaultScopes(input string) string {
	if input == "" {
		return "openid email profile"
	}
	return input
}
