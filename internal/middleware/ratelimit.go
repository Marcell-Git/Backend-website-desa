package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"rt26/backend/internal/pkg/response"
)

const (
	ipWindow       = time.Minute
	ipMaxRequests  = 10
	emailWindow    = 15 * time.Minute
	emailMaxFails  = 5
	maxBodyInspect = 64 << 10
)

type LoginRateLimiter struct {
	mu         sync.Mutex
	ipHits     map[string][]time.Time
	emailFails map[string][]time.Time
	now        func() time.Time
}

func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{
		ipHits:     make(map[string][]time.Time),
		emailFails: make(map[string][]time.Time),
		now:        time.Now,
	}
}

func (l *LoginRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if l.ipBlocked(ip) {
			response.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan login dari perangkat ini. Silakan coba lagi beberapa saat lagi.")
			return
		}

		email := extractLoginEmail(r)
		if email != "" && l.emailBlocked(email) {
			response.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan login gagal untuk akun ini. Silakan coba lagi dalam 15 menit.")
			return
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		switch {
		case rec.status == http.StatusUnauthorized:
			l.recordFailure(email)
		case rec.status < http.StatusBadRequest:
			l.clearFailures(email)
		}
	})
}

func (l *LoginRateLimiter) ipBlocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-ipWindow)
	hits := filterRecent(l.ipHits[ip], cutoff)

	if len(hits) >= ipMaxRequests {
		l.ipHits[ip] = hits
		return true
	}
	l.ipHits[ip] = append(hits, now)
	return false
}

func (l *LoginRateLimiter) emailBlocked(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := l.now().Add(-emailWindow)
	fails := filterRecent(l.emailFails[email], cutoff)
	l.emailFails[email] = fails
	return len(fails) >= emailMaxFails
}

func (l *LoginRateLimiter) recordFailure(email string) {
	if email == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.emailFails[email] = append(l.emailFails[email], l.now())
}

func (l *LoginRateLimiter) clearFailures(email string) {
	if email == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.emailFails, email)
}

func filterRecent(times []time.Time, cutoff time.Time) []time.Time {
	out := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	return out
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		if first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func extractLoginEmail(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyInspect))
	r.Body.Close()
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(nil))
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var req struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(body, &req) != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(req.Email))
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}
