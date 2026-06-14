package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	bruteForceLimit  = 10
	bruteForceWindow = 15 * time.Minute
)

// BruteForceProtect wraps auth endpoints. It tracks failed attempts per IP
// and blocks the IP after bruteForceLimit failures within bruteForceWindow.
func BruteForceProtect(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}
			ip := realIP(r)
			key := "brute:" + ip

			ctx := context.Background()
			count, _ := rdb.Get(ctx, key).Int()
			if count >= bruteForceLimit {
				ttl, _ := rdb.TTL(ctx, key).Result()
				retryAfter := int(ttl.Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				http.Error(w, `{"error":{"code":"ACCOUNT_LOCKED","message":"too many failed attempts, try again later"}}`, http.StatusTooManyRequests)
				return
			}

			// Wrap ResponseWriter to detect failure status codes
			rw := &statusCapture{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			// 401 Unauthorized on auth endpoints = failed attempt
			if rw.status == http.StatusUnauthorized {
				pipe := rdb.Pipeline()
				pipe.Incr(ctx, key)
				pipe.Expire(ctx, key, bruteForceWindow)
				pipe.Exec(ctx)
			}
		})
	}
}

// RecordBruteForceSuccess clears the brute-force counter for an IP on success.
func RecordBruteForceSuccess(rdb *redis.Client, ip string) {
	rdb.Del(context.Background(), "brute:"+ip)
}

type statusCapture struct {
	http.ResponseWriter
	status int
}

func (s *statusCapture) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusCapture) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}
