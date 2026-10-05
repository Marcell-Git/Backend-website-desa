package pengurus

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/pengurus", func(r chi.Router) {
		r.With(middleware.RequirePermission("pengurus", "view")).Get("/", h.List)
		r.With(middleware.RequirePermission("pengurus", "view")).Get("/all", h.ListAll)
		// Penyimpanan massalimpa seluruh data pengurus, jadi butuh izin
		// create, update, dan delete sekaligus.
		r.With(
			middleware.RequirePermission("pengurus", "create"),
			middleware.RequirePermission("pengurus", "update"),
			middleware.RequirePermission("pengurus", "delete"),
		).Put("/all", h.SaveBulk)
	})
}

// RegisterPublic mendaftarkan route tanpa login.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/public/pengurus", h.ListPublic)
}
