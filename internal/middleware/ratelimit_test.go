package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func loginRequest() *http.Request {
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"Admin@Example.com","password":"salah"}`))
	req.RemoteAddr = "10.0.0.1:54321"
	return req.Clone(context.Background())
}

func TestLoginRateLimiterEmailLockout(t *testing.T) {
	l := NewLoginRateLimiter()
	current := time.Now()
	l.now = func() time.Time { return current }

	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	var res *httptest.ResponseRecorder
	for i := 0; i < 5; i++ {
		res = httptest.NewRecorder()
		handler.ServeHTTP(res, loginRequest())
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("percobaan ke-%d: ingin 401, dapat %d", i+1, res.Code)
		}
	}

	res = httptest.NewRecorder()
	handler.ServeHTTP(res, loginRequest())
	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("setelah 5 kali gagal: ingin 429, dapat %d", res.Code)
	}

	current = current.Add(16 * time.Minute)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, loginRequest())
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("setelah lockout lewat: ingin 401, dapat %d", res.Code)
	}
}

func TestLoginRateLimiterIPThrottle(t *testing.T) {
	l := NewLoginRateLimiter()
	current := time.Now()
	l.now = func() time.Time { return current }

	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	for i := 0; i < ipMaxRequests; i++ {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, loginRequest())
		if res.Code == http.StatusTooManyRequests {
			t.Fatalf("request ke-%d tidak seharusnya diblokir", i+1)
		}
	}

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, loginRequest())
	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("melebihi batas IP: ingin 429, dapat %d", res.Code)
	}

	current = current.Add(ipWindow + time.Second)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, loginRequest())
	if res.Code != http.StatusBadRequest {
		t.Fatalf("window IP baru: ingin 400, dapat %d", res.Code)
	}
}

func TestLoginRateLimiterSuccessClearsFailures(t *testing.T) {
	l := NewLoginRateLimiter()
	current := time.Now()
	l.now = func() time.Time { return current }

	failHandler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	okHandler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < emailMaxFails-1; i++ {
		failHandler.ServeHTTP(httptest.NewRecorder(), loginRequest())
	}
	okHandler.ServeHTTP(httptest.NewRecorder(), loginRequest())

	for i := 0; i < emailMaxFails; i++ {
		res := httptest.NewRecorder()
		failHandler.ServeHTTP(res, loginRequest())
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("counter harus reset setelah sukses: request gagal ke-%d dapat %d", i+1, res.Code)
		}
	}
}

func TestLoginRateLimiterDifferentEmailsIndependent(t *testing.T) {
	l := NewLoginRateLimiter()

	failHandler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	for i := 0; i < emailMaxFails; i++ {
		failHandler.ServeHTTP(httptest.NewRecorder(), loginRequest())
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"lain@example.com","password":"x"}`))
	req.RemoteAddr = "10.0.0.2:1234"
	res := httptest.NewRecorder()
	failHandler.ServeHTTP(res, req.Clone(context.Background()))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("email lain tidak boleh terdampak lockout: ingin 401, dapat %d", res.Code)
	}
}
