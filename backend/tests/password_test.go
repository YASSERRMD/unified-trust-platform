package tests

import (
	"testing"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	plain := "MyS3cur3P@ssword!"
	hash, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if hash == plain {
		t.Error("hash should not equal plain text")
	}
	if !auth.CheckPassword(hash, plain) {
		t.Error("check should succeed for correct password")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Error("check should fail for wrong password")
	}
}

func TestPasswordPolicyValidation(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"too short", "Ab1!", true},
		{"no upper", "alllowercase1!", true},
		{"no lower", "ALLUPPERCASE1!", true},
		{"no digit", "NoDigitHere!!", true},
		{"no symbol", "NoSymbolHere1", true},
		{"valid", "Valid@Pass1word!", false},
		{"exactly 12 chars", "Abcdef1!ghij", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := auth.ValidatePasswordPolicy(tc.password, auth.DefaultPolicy)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %q but got none", tc.password)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tc.password, err)
			}
		})
	}
}
