package auditlog

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/audit-logs", func(r chi.Router) {
		r.With(middleware.RequirePermission("audit-log", "view")).Get("/", h.List)
		r.With(middleware.RequirePermission("audit-log", "view")).Get("/{id}", h.Get)
	})
}
