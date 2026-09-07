package main

import (
	"math"
	"net"
	"net/http"
	"sync"
	"time"
)

// defaultRateRefill is the time it takes to refill a single token at the
// default rate of roughly 100 requests per minute (one token every 600ms).
const defaultRateRefill = time.Minute / 100

// defaultRateCapacity is the burst size: how many requests a client may fire
// before the bucket must refill.
const defaultRateCapacity = 100

// tokenBucket is a simple token bucket. It is not safe for concurrent use;
// callers hold the rateLimiter mutex around it.
type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	refill     time.Duration // time required to refill a single token
	capacity   float64
}

func newTokenBucket(refill time.Duration, capacity int) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(capacity),
		lastRefill: time.Now(),
		refill:     refill,
		capacity:   float64(capacity),
	}
}

// allow consumes one token if available, refilling based on elapsed time first.
func (b *tokenBucket) allow() bool {
	now := time.Now()
	if b.refill > 0 {
		elapsed := now.Sub(b.lastRefill)
		if elapsed >= b.refill {
			b.tokens = math.Min(b.capacity, b.tokens+float64(elapsed)/float64(b.refill))
			b.lastRefill = now
		}
	}
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// rateLimiter tracks one token bucket per client key. All access is guarded by
// a single mutex, so it is safe for concurrent use.
type rateLimiter struct {
	mu       sync.Mutex
	refill   time.Duration
	capacity int
	buckets  map[string]*tokenBucket
}

func newRateLimiter(refill time.Duration, capacity int) *rateLimiter {
	return &rateLimiter{
		refill:   refill,
		capacity: capacity,
		buckets:  make(map[string]*tokenBucket),
	}
}

// allow reports whether the key is allowed one more request, creating a bucket
// on first use.
func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		b = newTokenBucket(l.refill, l.capacity)
		l.buckets[key] = b
	}
	return b.allow()
}

// clientIP extracts the host portion of the request's remote address, dropping
// the port so all connections from one IP share a single bucket.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimitMiddleware limits each client IP to roughly 100 requests per minute
// with a burst of 100. When the limit is exceeded it answers 429 with a JSON
// error object; otherwise it passes the request through.
func rateLimitMiddleware(next http.Handler) http.Handler {
	return rateLimitMiddlewareWithLimiter(next, newRateLimiter(defaultRateRefill, defaultRateCapacity))
}

// rateLimitMiddlewareWithLimiter is the same as rateLimitMiddleware but uses an
// explicitly supplied limiter, so tests can inject a faster refill interval.
func rateLimitMiddlewareWithLimiter(next http.Handler, limiter *rateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}
