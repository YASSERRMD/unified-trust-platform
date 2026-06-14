package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter returns a middleware that enforces a sliding-window rate limit
// using Redis. limit requests are allowed per window duration per IP address.
func RateLimiter(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}
			ip := realIP(r)
			key := "rate:" + ip

			ctx := context.Background()
			now := time.Now().UnixMilli()
			windowMs := window.Milliseconds()

			pipe := rdb.Pipeline()
			// Remove entries older than the window
			pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(now-windowMs, 10))
			// Count remaining
			countCmd := pipe.ZCard(ctx, key)
			// Add current request
			pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
			pipe.Expire(ctx, key, window+time.Second)
			_, err := pipe.Exec(ctx)
			if err != nil {
				// Redis unavailable — fail open
				next.ServeHTTP(w, r)
				return
			}

			count := countCmd.Val()
			remaining := int64(limit) - count
			if remaining < 0 {
				remaining = 0
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))

			if count >= int64(limit) {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(window.Seconds()), 10))
				http.Error(w, `{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// realIP extracts the client IP respecting common proxy headers.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For may be a comma-separated list; first is the client
		if idx := indexOf(ip, ','); idx != -1 {
			return ip[:idx]
		}
		return ip
	}
	return r.RemoteAddr
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
