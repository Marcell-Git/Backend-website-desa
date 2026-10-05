package dashboard

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.With(middleware.RequirePermission("dashboard", "view")).
		Get("/dashboard", h.Summary)
	r.With(middleware.RequirePermission("dashboard", "view")).
		Get("/dashboard/grafik", h.GrafikBulanan)
}
