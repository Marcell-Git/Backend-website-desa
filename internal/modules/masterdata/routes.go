package masterdata

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login. ocr ditulis terpisah karena
// panggilannya memakai GeminiHandler, bukan Handler biasa.
func Register(r chi.Router, h *Handler, ocr *OCRHandler) {
	r.Route("/kartu-keluarga", func(r chi.Router) {
		r.With(middleware.RequirePermission("kartu-keluarga", "view")).Get("/", h.ListKK)
		r.With(middleware.RequirePermission("kartu-keluarga", "create")).Post("/ocr", ocr.OCRKK)
		r.With(middleware.RequirePermission("kartu-keluarga", "create")).Post("/", h.CreateKK)
		r.With(middleware.RequirePermission("kartu-keluarga", "view")).Get("/{id}", h.GetKK)
		r.With(middleware.RequirePermission("kartu-keluarga", "update")).Put("/{id}", h.UpdateKK)
		r.With(middleware.RequirePermission("kartu-keluarga", "delete")).Delete("/{id}", h.DeleteKK)
	})

	r.Route("/anggota", func(r chi.Router) {
		r.With(middleware.RequirePermission("anggota", "view")).Get("/", h.ListAnggota)
		r.With(middleware.RequirePermission("anggota", "create")).Post("/", h.CreateAnggota)
		r.With(middleware.RequirePermission("anggota", "view")).Get("/{id}", h.GetAnggota)
		r.With(middleware.RequirePermission("anggota", "update")).Put("/{id}", h.UpdateAnggota)
		r.With(middleware.RequirePermission("anggota", "delete")).Delete("/{id}", h.DeleteAnggota)
		r.With(middleware.RequirePermission("anggota", "update")).Patch("/{id}/kepala", h.SetKepala)
		r.With(middleware.RequirePermission("anggota", "update")).Patch("/{id}/pindah", h.MoveAnggota)
	})

	r.Route("/anak-kos", func(r chi.Router) {
		r.With(middleware.RequirePermission("anak-kos", "view")).Get("/", h.ListAnakKos)
		r.With(middleware.RequirePermission("anak-kos", "create")).Post("/ocr", ocr.OCRKTP)
		r.With(middleware.RequirePermission("anak-kos", "create")).Post("/", h.CreateAnakKos)
		r.With(middleware.RequirePermission("anak-kos", "view")).Get("/{id}", h.GetAnakKos)
		r.With(middleware.RequirePermission("anak-kos", "update")).Put("/{id}", h.UpdateAnakKos)
		r.With(middleware.RequirePermission("anak-kos", "delete")).Delete("/{id}", h.DeleteAnakKos)
	})
}

// RegisterPublic mendaftarkan route tanpa login untuk pendataan mandiri.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Post("/public/anak-kos", h.CreateAnakKos)
}
