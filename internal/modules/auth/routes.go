package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Post("/auth/logout", h.Logout)
	r.Get("/auth/profile", h.Profile)
	r.Put("/auth/password", h.ChangePassword)
}

// RegisterPublic mendaftarkan route tanpa login. limiter hanya dipasang pada
// /auth/login agar percobaan brute force dibatasi.
func RegisterPublic(r chi.Router, h *Handler, limiter func(http.Handler) http.Handler) {
	r.With(limiter).Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.Refresh)
}
