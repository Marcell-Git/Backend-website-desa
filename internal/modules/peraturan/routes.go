package peraturan

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/peraturan", func(r chi.Router) {
		r.With(middleware.RequirePermission("peraturan", "view")).Get("/", h.List)
		r.With(middleware.RequirePermission("peraturan", "create")).Post("/", h.Create)
		r.With(middleware.RequirePermission("peraturan", "view")).Get("/{id}", h.Get)
		r.With(middleware.RequirePermission("peraturan", "update")).Put("/{id}", h.Update)
		r.With(middleware.RequirePermission("peraturan", "delete")).Delete("/{id}", h.Delete)
	})
}

// RegisterPublic mendaftarkan route tanpa login.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/public/peraturan", h.ListPublic)
}
