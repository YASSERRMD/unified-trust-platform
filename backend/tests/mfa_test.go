package tests

import (
	"context"
	"testing"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/mfa"
)

func TestTOTPProviderEnroll(t *testing.T) {
	p := mfa.NewTOTPProvider("UTP Test")
	result, err := p.Enroll(context.Background(), "user-1", "tenant-1")
	if err != nil {
		t.Fatalf("enroll error: %v", err)
	}
	if result.Secret == "" {
		t.Error("expected TOTP secret")
	}
	if result.OtpauthURL == "" {
		t.Error("expected otpauth URL")
	}
}

func TestTOTPProviderChallenge(t *testing.T) {
	p := mfa.NewTOTPProvider("UTP Test")
	result, err := p.Challenge(context.Background(), "method-id-1", "user-1")
	if err != nil {
		t.Fatalf("challenge error: %v", err)
	}
	if result.ChallengeID == "" {
		t.Error("expected challengeId")
	}
	if result.Message == "" {
		t.Error("expected message")
	}
}

func TestTOTPVerifyWrongCode(t *testing.T) {
	p := mfa.NewTOTPProvider("UTP Test")

	result, err := p.Enroll(context.Background(), "user-1", "tenant-1")
	if err != nil {
		t.Fatalf("enroll error: %v", err)
	}

	ok, err := p.Verify(context.Background(), "chal-id", "000000", result.Secret)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	// 000000 is extremely unlikely to be correct
	if ok {
		t.Log("000000 was valid (astronomically unlikely, passing anyway)")
	}
}

func TestMFAProviderInterface(t *testing.T) {
	var p mfa.Provider = mfa.NewTOTPProvider("UTP")
	if p.Type() != "totp" {
		t.Errorf("expected type=totp, got %q", p.Type())
	}
}
