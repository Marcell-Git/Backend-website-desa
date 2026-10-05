package backup

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/backup", func(r chi.Router) {
		r.With(middleware.RequirePermission("backup", "view")).Get("/", h.List)
		r.With(middleware.RequirePermission("backup", "create")).Post("/", h.Create)
		r.With(middleware.RequirePermission("backup", "view")).Get("/{id}/download", h.Download)
		r.With(middleware.RequirePermission("backup", "restore")).Post("/{id}/restore", h.Restore)
		r.With(middleware.RequirePermission("backup", "delete")).Delete("/{id}", h.Delete)
	})
}
