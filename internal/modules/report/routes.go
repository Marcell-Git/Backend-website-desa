package report

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login. Modul ini punya dua handler:
// h untuk laporan, ex untuk ekspor PDF/Excel.
func Register(r chi.Router, h *Handler, ex *ExportHandler) {
	r.Route("/reports", func(r chi.Router) {
		r.With(middleware.RequirePermission("laporan", "view")).Get("/kas", h.LaporanKas)
		r.With(middleware.RequirePermission("laporan", "view")).Get("/iuran", h.LaporanIuran)
		r.With(middleware.RequirePermission("laporan", "view")).Get("/pengeluaran", h.LaporanPengeluaran)
		r.With(middleware.RequirePermission("laporan", "view")).Get("/keuangan", h.LaporanKeuangan)
	})

	r.Route("/export", func(r chi.Router) {
		r.With(middleware.RequirePermission("laporan", "export")).Get("/kas/pdf", ex.ExportKasPDF)
		r.With(middleware.RequirePermission("laporan", "export")).Get("/kas/excel", ex.ExportKasExcel)
		r.With(middleware.RequirePermission("laporan", "export")).Get("/iuran/pdf", ex.ExportIuranPDF)
		r.With(middleware.RequirePermission("laporan", "export")).Get("/iuran/excel", ex.ExportIuranExcel)
		r.With(middleware.RequirePermission("laporan", "export")).Get("/pengeluaran/pdf", ex.ExportPengeluaranPDF)
		r.With(middleware.RequirePermission("laporan", "export")).Get("/pengeluaran/excel", ex.ExportPengeluaranExcel)
		r.With(middleware.RequirePermission("kartu-keluarga", "export")).Get("/kk/pdf", ex.ExportKKPDF)
		r.With(middleware.RequirePermission("anak-kos", "export")).Get("/anak-kos/pdf", ex.ExportAnakKosPDF)
	})
}

// RegisterPublic mendaftarkan route tanpa login.
func RegisterPublic(r chi.Router, h *Handler) {
	r.Get("/public/laporan", h.PublicLaporan)
}
