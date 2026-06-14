package mfa

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const totpDigits = 6
const totpPeriod = 30

// TOTPProvider implements Provider for RFC 6238 TOTP.
type TOTPProvider struct {
	issuer string
}

func NewTOTPProvider(issuer string) *TOTPProvider {
	return &TOTPProvider{issuer: issuer}
}

func (p *TOTPProvider) Type() string { return "totp" }

func (p *TOTPProvider) Enroll(ctx context.Context, userID, tenantID string) (*EnrollResult, error) {
	secret, err := generateTOTPSecret()
	if err != nil {
		return nil, err
	}

	label := url.QueryEscape(userID + "@" + tenantID)
	issuer := url.QueryEscape(p.issuer)
	otpauth := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&digits=%d&period=%d",
		label, secret, issuer, totpDigits, totpPeriod)

	return &EnrollResult{
		Secret:     secret,
		OtpauthURL: otpauth,
		QRCodeURL:  "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=" + url.QueryEscape(otpauth),
	}, nil
}

func (p *TOTPProvider) Challenge(_ context.Context, methodID, _ string) (*ChallengeResult, error) {
	return &ChallengeResult{
		ChallengeID: methodID,
		Message:     "Enter the 6-digit code from your authenticator app",
	}, nil
}

func (p *TOTPProvider) Verify(_ context.Context, _, code, secret string) (bool, error) {
	now := time.Now().Unix()
	for _, delta := range []int64{-1, 0, 1} {
		counter := (now/totpPeriod + delta)
		expected := computeTOTP(secret, counter)
		if code == expected {
			return true, nil
		}
	}
	return false, nil
}

func generateTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "="), nil
}

func computeTOTP(secret string, counter int64) string {
	key, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret) + strings.Repeat("=", (8-len(secret)%8)%8))
	if err != nil {
		return ""
	}

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	h := hmac.New(sha1.New, key)
	h.Write(buf)
	sum := h.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, code%uint32(math.Pow10(totpDigits)))
}
