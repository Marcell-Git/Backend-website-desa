package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"rt26/backend/internal/config"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/modules/announcement"
	"rt26/backend/internal/modules/auditlog"
	"rt26/backend/internal/modules/auth"
	"rt26/backend/internal/modules/dashboard"
	"rt26/backend/internal/modules/finance"
	"rt26/backend/internal/modules/masterdata"
	"rt26/backend/internal/modules/pengaduan"
	"rt26/backend/internal/modules/report"
	"rt26/backend/internal/modules/user"
	"rt26/backend/internal/pkg/response"
)

func New(db *gorm.DB, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.CORS)
	r.Use(recoverer)

	authService := auth.NewService(db, cfg.JWTSecret, cfg.AccessMinutes, cfg.RefreshDays)
	authHandler := auth.NewHandler(authService)
	dashboardHandler := dashboard.NewHandler(db)
	masterdataHandler := masterdata.NewHandler(db)
	financeHandler := finance.NewHandler(db)
	announcementHandler := announcement.NewHandler(db)
	pengaduanHandler := pengaduan.NewHandler(db)
	auditLogHandler := auditlog.NewHandler(db)
	reportHandler := report.NewHandler(db)
	exportHandler := report.NewExportHandler(db)
	userHandler := user.NewHandler(db)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, http.StatusOK, "OK", map[string]string{"status": "healthy"})
		})

		// public
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Get("/public/pengumuman", announcementHandler.ListPublished)
		r.Post("/public/pengaduan", pengaduanHandler.SubmitPengaduan)

		// protected
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.JWTSecret))

			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/profile", authHandler.Profile)
			r.Put("/auth/password", authHandler.ChangePassword)

			r.With(middleware.RequirePermission("dashboard", "view")).
				Get("/dashboard", dashboardHandler.Summary)

			r.Route("/kartu-keluarga", func(r chi.Router) {
				r.With(middleware.RequirePermission("kartu-keluarga", "view")).Get("/", masterdataHandler.ListKK)
				r.With(middleware.RequirePermission("kartu-keluarga", "create")).Post("/", masterdataHandler.CreateKK)
				r.With(middleware.RequirePermission("kartu-keluarga", "view")).Get("/{id}", masterdataHandler.GetKK)
				r.With(middleware.RequirePermission("kartu-keluarga", "update")).Put("/{id}", masterdataHandler.UpdateKK)
				r.With(middleware.RequirePermission("kartu-keluarga", "delete")).Delete("/{id}", masterdataHandler.DeleteKK)
			})

			r.Route("/anggota", func(r chi.Router) {
				r.With(middleware.RequirePermission("anggota", "view")).Get("/", masterdataHandler.ListAnggota)
				r.With(middleware.RequirePermission("anggota", "create")).Post("/", masterdataHandler.CreateAnggota)
				r.With(middleware.RequirePermission("anggota", "view")).Get("/{id}", masterdataHandler.GetAnggota)
				r.With(middleware.RequirePermission("anggota", "update")).Put("/{id}", masterdataHandler.UpdateAnggota)
				r.With(middleware.RequirePermission("anggota", "delete")).Delete("/{id}", masterdataHandler.DeleteAnggota)
				r.With(middleware.RequirePermission("anggota", "update")).Patch("/{id}/kepala", masterdataHandler.SetKepala)
				r.With(middleware.RequirePermission("anggota", "update")).Patch("/{id}/pindah", masterdataHandler.MoveAnggota)
			})

			r.Route("/anak-kos", func(r chi.Router) {
				r.With(middleware.RequirePermission("anak-kos", "view")).Get("/", masterdataHandler.ListAnakKos)
				r.With(middleware.RequirePermission("anak-kos", "create")).Post("/", masterdataHandler.CreateAnakKos)
				r.With(middleware.RequirePermission("anak-kos", "view")).Get("/{id}", masterdataHandler.GetAnakKos)
				r.With(middleware.RequirePermission("anak-kos", "update")).Put("/{id}", masterdataHandler.UpdateAnakKos)
				r.With(middleware.RequirePermission("anak-kos", "delete")).Delete("/{id}", masterdataHandler.DeleteAnakKos)
			})

			r.Route("/kas", func(r chi.Router) {
				r.With(middleware.RequirePermission("kas", "view")).Get("/", financeHandler.ListKas)
				r.With(middleware.RequirePermission("kas", "create")).Post("/", financeHandler.CreateKas)
				r.With(middleware.RequirePermission("kas", "view")).Get("/{id}", financeHandler.GetKas)
				r.With(middleware.RequirePermission("kas", "update")).Put("/{id}", financeHandler.UpdateKas)
				r.With(middleware.RequirePermission("kas", "delete")).Delete("/{id}", financeHandler.DeleteKas)
			})

			r.Route("/jenis-iuran", func(r chi.Router) {
				r.With(middleware.RequirePermission("jenis-iuran", "view")).Get("/", financeHandler.ListJenisIuran)
				r.With(middleware.RequirePermission("jenis-iuran", "create")).Post("/", financeHandler.CreateJenisIuran)
				r.With(middleware.RequirePermission("jenis-iuran", "update")).Put("/{id}", financeHandler.UpdateJenisIuran)
				r.With(middleware.RequirePermission("jenis-iuran", "delete")).Delete("/{id}", financeHandler.DeleteJenisIuran)
			})

			r.Route("/iuran", func(r chi.Router) {
				r.With(middleware.RequirePermission("iuran", "view")).Get("/", financeHandler.ListIuran)
				r.With(middleware.RequirePermission("iuran", "create")).Post("/", financeHandler.CreateIuran)
				r.With(middleware.RequirePermission("iuran", "view")).Get("/{id}", financeHandler.GetIuran)
				r.With(middleware.RequirePermission("iuran", "update")).Put("/{id}", financeHandler.UpdateIuran)
				r.With(middleware.RequirePermission("iuran", "delete")).Delete("/{id}", financeHandler.DeleteIuran)
			})

			r.Route("/kategori-pengeluaran", func(r chi.Router) {
				r.With(middleware.RequirePermission("kategori-pengeluaran", "view")).Get("/", financeHandler.ListKategoriPengeluaran)
				r.With(middleware.RequirePermission("kategori-pengeluaran", "create")).Post("/", financeHandler.CreateKategoriPengeluaran)
				r.With(middleware.RequirePermission("kategori-pengeluaran", "update")).Put("/{id}", financeHandler.UpdateKategoriPengeluaran)
				r.With(middleware.RequirePermission("kategori-pengeluaran", "delete")).Delete("/{id}", financeHandler.DeleteKategoriPengeluaran)
			})

			r.Route("/pengeluaran", func(r chi.Router) {
				r.With(middleware.RequirePermission("pengeluaran", "view")).Get("/", financeHandler.ListPengeluaran)
				r.With(middleware.RequirePermission("pengeluaran", "create")).Post("/", financeHandler.CreatePengeluaran)
				r.With(middleware.RequirePermission("pengeluaran", "view")).Get("/{id}", financeHandler.GetPengeluaran)
				r.With(middleware.RequirePermission("pengeluaran", "update")).Put("/{id}", financeHandler.UpdatePengeluaran)
				r.With(middleware.RequirePermission("pengeluaran", "delete")).Delete("/{id}", financeHandler.DeletePengeluaran)
			})

			r.Route("/pengumuman", func(r chi.Router) {
				r.With(middleware.RequirePermission("pengumuman", "view")).Get("/", announcementHandler.ListPengumuman)
				r.With(middleware.RequirePermission("pengumuman", "create")).Post("/", announcementHandler.CreatePengumuman)
				r.With(middleware.RequirePermission("pengumuman", "view")).Get("/{id}", announcementHandler.GetPengumuman)
				r.With(middleware.RequirePermission("pengumuman", "update")).Put("/{id}", announcementHandler.UpdatePengumuman)
				r.With(middleware.RequirePermission("pengumuman", "delete")).Delete("/{id}", announcementHandler.DeletePengumuman)
			})

			r.Route("/pengaduan", func(r chi.Router) {
				r.With(middleware.RequirePermission("pengaduan", "view")).Get("/", pengaduanHandler.ListPengaduan)
				r.With(middleware.RequirePermission("pengaduan", "view")).Get("/{id}", pengaduanHandler.GetPengaduan)
				r.With(middleware.RequirePermission("pengaduan", "update")).Patch("/{id}/status", pengaduanHandler.UpdateStatus)
				r.With(middleware.RequirePermission("pengaduan", "update")).Post("/{id}/catatan", pengaduanHandler.AddCatatan)
				r.With(middleware.RequirePermission("pengaduan", "view")).Get("/{id}/bukti", pengaduanHandler.DownloadBukti)
				r.With(middleware.RequirePermission("pengaduan", "delete")).Delete("/{id}", pengaduanHandler.DeletePengaduan)
			})

			r.Route("/audit-logs", func(r chi.Router) {
				r.With(middleware.RequirePermission("audit-log", "view")).Get("/", auditLogHandler.List)
				r.With(middleware.RequirePermission("audit-log", "view")).Get("/{id}", auditLogHandler.Get)
			})

			r.Route("/users", func(r chi.Router) {
				r.With(middleware.RequirePermission("user", "view")).Get("/", userHandler.ListUsers)
				r.With(middleware.RequirePermission("user", "create")).Post("/", userHandler.CreateUser)
				r.With(middleware.RequirePermission("user", "view")).Get("/{id}", userHandler.GetUser)
				r.With(middleware.RequirePermission("user", "update")).Put("/{id}", userHandler.UpdateUser)
				r.With(middleware.RequirePermission("user", "update")).Patch("/{id}/status", userHandler.UpdateUserStatus)
				r.With(middleware.RequirePermission("user", "update")).Patch("/{id}/password", userHandler.ResetPassword)
			})

			r.With(middleware.RequirePermission("role", "view")).Get("/roles", userHandler.ListRoles)

			r.Route("/reports", func(r chi.Router) {
				r.With(middleware.RequirePermission("laporan", "view")).Get("/kas", reportHandler.LaporanKas)
				r.With(middleware.RequirePermission("laporan", "view")).Get("/iuran", reportHandler.LaporanIuran)
				r.With(middleware.RequirePermission("laporan", "view")).Get("/pengeluaran", reportHandler.LaporanPengeluaran)
				r.With(middleware.RequirePermission("laporan", "view")).Get("/keuangan", reportHandler.LaporanKeuangan)
			})

			r.Route("/export", func(r chi.Router) {
				r.With(middleware.RequirePermission("laporan", "export")).Get("/kas/pdf", exportHandler.ExportKasPDF)
				r.With(middleware.RequirePermission("laporan", "export")).Get("/kas/excel", exportHandler.ExportKasExcel)
				r.With(middleware.RequirePermission("laporan", "export")).Get("/iuran/pdf", exportHandler.ExportIuranPDF)
				r.With(middleware.RequirePermission("laporan", "export")).Get("/iuran/excel", exportHandler.ExportIuranExcel)
				r.With(middleware.RequirePermission("laporan", "export")).Get("/pengeluaran/pdf", exportHandler.ExportPengeluaranPDF)
				r.With(middleware.RequirePermission("laporan", "export")).Get("/pengeluaran/excel", exportHandler.ExportPengeluaranExcel)
			})
		})
	})

	return r
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, `{"success":false,"message":"Internal Server Error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
