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
	ipWindow        = time.Minute
	ipMaxRequests   = 10
	usernameWindow  = 15 * time.Minute
	usernameMaxFails = 5
	maxBodyInspect  = 64 << 10
)

type LoginRateLimiter struct {
	mu            sync.Mutex
	ipHits        map[string][]time.Time
	usernameFails map[string][]time.Time
	now           func() time.Time
}

func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{
		ipHits:        make(map[string][]time.Time),
		usernameFails: make(map[string][]time.Time),
		now:           time.Now,
	}
}

func (l *LoginRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if l.ipBlocked(ip) {
			response.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan login dari perangkat ini. Silakan coba lagi beberapa saat lagi.")
			return
		}

		username := extractLoginUsername(r)
		if username != "" && l.usernameBlocked(username) {
			response.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan login gagal untuk akun ini. Silakan coba lagi dalam 15 menit.")
			return
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		switch {
		case rec.status == http.StatusUnauthorized:
			l.recordFailure(username)
		case rec.status < http.StatusBadRequest:
			l.clearFailures(username)
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

func (l *LoginRateLimiter) usernameBlocked(username string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := l.now().Add(-usernameWindow)
	fails := filterRecent(l.usernameFails[username], cutoff)
	l.usernameFails[username] = fails
	return len(fails) >= usernameMaxFails
}

func (l *LoginRateLimiter) recordFailure(username string) {
	if username == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.usernameFails[username] = append(l.usernameFails[username], l.now())
}

func (l *LoginRateLimiter) clearFailures(username string) {
	if username == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.usernameFails, username)
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

func extractLoginUsername(r *http.Request) string {
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
		Username string `json:"username"`
	}
	if json.Unmarshal(body, &req) != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(req.Username))
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
