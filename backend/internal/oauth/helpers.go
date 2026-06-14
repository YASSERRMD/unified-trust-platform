package oauth

import (
	"crypto/sha256"
	"fmt"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/auth"
)

func sha256Bytes(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

func checkSecret(hash, secret string) bool {
	return auth.CheckPassword(hash, secret)
}

func sha256Hex(s string) string {
	return fmt.Sprintf("%x", sha256Bytes([]byte(s)))
}
