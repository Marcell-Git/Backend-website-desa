package pengaduan

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/pengaduan", func(r chi.Router) {
		r.With(middleware.RequirePermission("pengaduan", "view")).Get("/", h.ListPengaduan)
		r.With(middleware.RequirePermission("pengaduan", "view")).Get("/{id}", h.GetPengaduan)
		r.With(middleware.RequirePermission("pengaduan", "update")).Patch("/{id}/status", h.UpdateStatus)
		r.With(middleware.RequirePermission("pengaduan", "update")).Post("/{id}/catatan", h.AddCatatan)
		r.With(middleware.RequirePermission("pengaduan", "view")).Get("/{id}/bukti", h.DownloadBukti)
		r.With(middleware.RequirePermission("pengaduan", "delete")).Delete("/{id}", h.DeletePengaduan)
	})
}

// RegisterPublic mendaftarkan route tanpa login untuk warga yang mengirim pengaduan.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Post("/public/pengaduan", h.SubmitPengaduan)
}
