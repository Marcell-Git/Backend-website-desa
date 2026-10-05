package finance

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/kas", func(r chi.Router) {
		r.With(middleware.RequirePermission("kas", "view")).Get("/", h.ListKas)
		r.With(middleware.RequirePermission("kas", "create")).Post("/", h.CreateKas)
		r.With(middleware.RequirePermission("kas", "view")).Get("/{id}", h.GetKas)
		r.With(middleware.RequirePermission("kas", "update")).Put("/{id}", h.UpdateKas)
		r.With(middleware.RequirePermission("kas", "delete")).Delete("/{id}", h.DeleteKas)
	})

	r.Route("/jenis-iuran", func(r chi.Router) {
		r.With(middleware.RequirePermission("jenis-iuran", "view")).Get("/", h.ListJenisIuran)
		r.With(middleware.RequirePermission("jenis-iuran", "create")).Post("/", h.CreateJenisIuran)
		r.With(middleware.RequirePermission("jenis-iuran", "update")).Put("/{id}", h.UpdateJenisIuran)
		r.With(middleware.RequirePermission("jenis-iuran", "delete")).Delete("/{id}", h.DeleteJenisIuran)
	})

	r.Route("/iuran", func(r chi.Router) {
		r.With(middleware.RequirePermission("iuran", "view")).Get("/", h.ListIuran)
		r.With(middleware.RequirePermission("iuran", "create")).Post("/", h.CreateIuran)
		r.With(middleware.RequirePermission("iuran", "view")).Get("/{id}", h.GetIuran)
		r.With(middleware.RequirePermission("iuran", "update")).Put("/{id}", h.UpdateIuran)
		r.With(middleware.RequirePermission("iuran", "delete")).Delete("/{id}", h.DeleteIuran)
	})

	r.Route("/kategori-pengeluaran", func(r chi.Router) {
		r.With(middleware.RequirePermission("kategori-pengeluaran", "view")).Get("/", h.ListKategoriPengeluaran)
		r.With(middleware.RequirePermission("kategori-pengeluaran", "create")).Post("/", h.CreateKategoriPengeluaran)
		r.With(middleware.RequirePermission("kategori-pengeluaran", "update")).Put("/{id}", h.UpdateKategoriPengeluaran)
		r.With(middleware.RequirePermission("kategori-pengeluaran", "delete")).Delete("/{id}", h.DeleteKategoriPengeluaran)
	})

	r.Route("/pengeluaran", func(r chi.Router) {
		r.With(middleware.RequirePermission("pengeluaran", "view")).Get("/", h.ListPengeluaran)
		r.With(middleware.RequirePermission("pengeluaran", "create")).Post("/", h.CreatePengeluaran)
		r.With(middleware.RequirePermission("pengeluaran", "view")).Get("/{id}", h.GetPengeluaran)
		r.With(middleware.RequirePermission("pengeluaran", "update")).Put("/{id}", h.UpdatePengeluaran)
		r.With(middleware.RequirePermission("pengeluaran", "delete")).Delete("/{id}", h.DeletePengeluaran)
	})
}
