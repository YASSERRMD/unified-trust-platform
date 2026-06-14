package mfa

import "context"

// Provider is the interface all MFA methods must implement.
type Provider interface {
	// Type returns the method type identifier.
	Type() string
	// Enroll initialises a new enrollment and returns the secret/QR data.
	Enroll(ctx context.Context, userID, tenantID string) (*EnrollResult, error)
	// Challenge generates a new challenge (OTP send / TOTP instruction).
	Challenge(ctx context.Context, methodID, userID string) (*ChallengeResult, error)
	// Verify validates the user-supplied code against the challenge.
	Verify(ctx context.Context, challengeID, code, secret string) (bool, error)
}

type EnrollResult struct {
	Secret     string `json:"secret,omitempty"`
	QRCodeURL  string `json:"qrCodeUrl,omitempty"`
	OtpauthURL string `json:"otpauthUrl,omitempty"`
}

type ChallengeResult struct {
	ChallengeID string `json:"challengeId"`
	Message     string `json:"message"`
}
