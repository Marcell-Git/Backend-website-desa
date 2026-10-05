package announcement

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/pengumuman", func(r chi.Router) {
		r.With(middleware.RequirePermission("pengumuman", "view")).Get("/", h.ListPengumuman)
		r.With(middleware.RequirePermission("pengumuman", "create")).Post("/", h.CreatePengumuman)
		r.With(middleware.RequirePermission("pengumuman", "view")).Get("/{id}", h.GetPengumuman)
		r.With(middleware.RequirePermission("pengumuman", "update")).Put("/{id}", h.UpdatePengumuman)
		r.With(middleware.RequirePermission("pengumuman", "delete")).Delete("/{id}", h.DeletePengumuman)
	})
}

// RegisterPublic mendaftarkan route tanpa login, hanya pengumuman berstatus terbit.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/public/pengumuman", h.ListPublished)
	r.Get("/public/pengumuman/{id}", h.GetPublishedDetail)
}
