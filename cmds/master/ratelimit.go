package main

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	. "github.com/ayushmantripathy/url-shortner/internal"
)

const (
	defaultRateLimitMaxRequests = 60
	defaultRateLimitWindow      = time.Minute
)

type rateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	max     int
	clients map[string]*clientWindow
}

type clientWindow struct {
	count     int
	windowEnd time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		window:  window,
		max:     max,
		clients: make(map[string]*clientWindow),
	}
}

func newRateLimiterFromEnv() *rateLimiter {
	max := defaultRateLimitMaxRequests
	window := defaultRateLimitWindow

	if v := os.Getenv("RATE_LIMIT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			max = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_WINDOW"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			window = d
		}
	}

	return newRateLimiter(max, window)
}

func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	cw, ok := rl.clients[key]
	if !ok || !now.Before(cw.windowEnd) {
		rl.clients[key] = &clientWindow{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}

	if cw.count >= rl.max {
		return false
	}

	cw.count++
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			ReturnHTTPMessage(w, http.StatusTooManyRequests, "Too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}
