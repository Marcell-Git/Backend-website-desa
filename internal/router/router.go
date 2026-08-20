package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rt26/backend/internal/config"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/modules/auth"
	"rt26/backend/internal/modules/dashboard"
	"rt26/backend/internal/modules/masterdata"
	"rt26/backend/internal/pkg/response"
)

func New(pool *pgxpool.Pool, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.CORS)
	r.Use(recoverer)

	authService := auth.NewService(pool, cfg.JWTSecret, cfg.AccessMinutes, cfg.RefreshDays)
	authHandler := auth.NewHandler(authService)
	dashboardHandler := dashboard.NewHandler(pool)
	masterdataHandler := masterdata.NewHandler(pool)

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
