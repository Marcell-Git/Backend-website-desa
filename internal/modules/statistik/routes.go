package statistik

import (
	"github.com/go-chi/chi/v5"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {}

// RegisterPublic mendaftarkan route tanpa login.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/public/statistik", h.PublicStatistik)
}
