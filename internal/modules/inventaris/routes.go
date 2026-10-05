package inventaris

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/inventaris", func(r chi.Router) {
		r.With(middleware.RequirePermission("inventaris", "view")).Get("/", h.List)
		r.With(middleware.RequirePermission("inventaris", "create")).Post("/", h.Create)
		r.With(middleware.RequirePermission("inventaris", "view")).Get("/{id}", h.Get)
		r.With(middleware.RequirePermission("inventaris", "update")).Put("/{id}", h.Update)
		r.With(middleware.RequirePermission("inventaris", "delete")).Delete("/{id}", h.Delete)
	})
}

// RegisterPublic mendaftarkan route tanpa login.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/public/inventaris", h.ListPublic)
}
