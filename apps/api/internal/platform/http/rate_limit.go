package httpx

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateBucket struct {
	count     int
	resetAt   time.Time
	updatedAt time.Time
}

type RateLimiter struct {
	limit   int
	window  time.Duration
	mu      sync.Mutex
	buckets map[string]rateBucket
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]rateBucket),
	}
}

func (l *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(rateLimitKey(r), time.Now()) {
				WriteError(w, r, nilLogger(), NewError(http.StatusTooManyRequests, CodeRateLimited, "Слишком много запросов"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *RateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buckets) > 10000 {
		for bucketKey, bucket := range l.buckets {
			if now.Sub(bucket.updatedAt) > l.window*2 {
				delete(l.buckets, bucketKey)
			}
		}
	}

	bucket := l.buckets[key]
	if bucket.resetAt.IsZero() || now.After(bucket.resetAt) {
		l.buckets[key] = rateBucket{count: 1, resetAt: now.Add(l.window), updatedAt: now}
		return true
	}
	if bucket.count >= l.limit {
		bucket.updatedAt = now
		l.buckets[key] = bucket
		return false
	}
	bucket.count++
	bucket.updatedAt = now
	l.buckets[key] = bucket
	return true
}

func rateLimitKey(r *http.Request) string {
	return clientIP(r) + ":" + r.Method + ":" + r.URL.Path
}

func clientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
