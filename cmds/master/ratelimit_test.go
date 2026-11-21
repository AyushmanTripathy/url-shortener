package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterMax(t *testing.T) {
	rl := newRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !rl.allow("192.0.2.1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	if rl.allow("192.0.2.1") {
		t.Fatal("4th request should be blocked")
	}
}

func TestRateLimiterIsolatesClients(t *testing.T) {
	rl := newRateLimiter(1, time.Minute)

	if !rl.allow("192.0.2.1") {
		t.Fatal("first client's first request should be allowed")
	}
	if !rl.allow("198.51.100.7") {
		t.Fatal("second client's first request should be allowed")
	}
	if rl.allow("192.0.2.1") {
		t.Fatal("first client's second request should be blocked")
	}
}

func TestRateLimiterWindowResets(t *testing.T) {
	rl := newRateLimiter(1, 10*time.Millisecond)

	if !rl.allow("192.0.2.1") {
		t.Fatal("first request should be allowed")
	}
	if rl.allow("192.0.2.1") {
		t.Fatal("second request should be blocked")
	}

	time.Sleep(20 * time.Millisecond)
	if !rl.allow("192.0.2.1") {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestRateLimiterMiddlewareReturns429(t *testing.T) {
	rl := newRateLimiter(1, time.Minute)

	handler := rl.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("first request: got %d, want %d", rr.Code, http.StatusCreated)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: got %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
}

func TestClientIPUsesRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "203.0.113.9:4567"

	if got := clientIP(req); got != "203.0.113.9" {
		t.Fatalf("clientIP() = %q, want %q", got, "203.0.113.9")
	}
}
