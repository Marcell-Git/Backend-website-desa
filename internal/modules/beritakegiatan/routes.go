package beritakegiatan

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/berita-kegiatan", func(r chi.Router) {
		r.With(middleware.RequirePermission("berita-kegiatan", "view")).Get("/", h.List)
		r.With(middleware.RequirePermission("berita-kegiatan", "create")).Post("/", h.Create)
		r.With(middleware.RequirePermission("berita-kegiatan", "view")).Get("/{id}", h.Get)
		r.With(middleware.RequirePermission("berita-kegiatan", "update")).Put("/{id}", h.Update)
		r.With(middleware.RequirePermission("berita-kegiatan", "delete")).Delete("/{id}", h.Delete)
	})
}

// RegisterPublic mendaftarkan route tanpa login.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/public/berita-kegiatan", h.ListPublic)
}
