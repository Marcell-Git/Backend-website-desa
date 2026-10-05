package settings

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.With(middleware.RequirePermission("website-settings", "update")).
		Put("/website-settings", h.UpdateSettings)
	r.With(middleware.RequirePermission("website-settings", "update")).
		Post("/website-settings/logo", h.UploadLogo)
}

// RegisterPublic mendaftarkan route tanpa login. Pengaturan website dibaca oleh
// halaman publik, jadi hanya GET yang terbuka.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/website-settings", h.GetSettings)
}
