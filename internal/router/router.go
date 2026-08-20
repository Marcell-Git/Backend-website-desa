package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"rt26/backend/internal/config"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/modules/auth"
	"rt26/backend/internal/modules/dashboard"
	"rt26/backend/internal/modules/finance"
	"rt26/backend/internal/modules/masterdata"
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

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, http.StatusOK, "OK", map[string]string{"status": "healthy"})
		})

		// public
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)

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
