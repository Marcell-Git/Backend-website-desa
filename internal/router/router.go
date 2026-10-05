package router

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"rt26/backend/internal/config"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/modules/announcement"
	"rt26/backend/internal/modules/auditlog"
	"rt26/backend/internal/modules/auth"
	"rt26/backend/internal/modules/backup"
	"rt26/backend/internal/modules/beritakegiatan"
	"rt26/backend/internal/modules/dashboard"
	"rt26/backend/internal/modules/finance"
	"rt26/backend/internal/modules/inventaris"
	"rt26/backend/internal/modules/masterdata"
	"rt26/backend/internal/modules/pengaduan"
	"rt26/backend/internal/modules/pengurus"
	"rt26/backend/internal/modules/peraturan"
	"rt26/backend/internal/modules/report"
	"rt26/backend/internal/modules/settings"
	"rt26/backend/internal/modules/statistik"
	"rt26/backend/internal/modules/user"
	"rt26/backend/internal/pkg/response"
)

func New(db *gorm.DB, cfg *config.Config, baseDir ...string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.CORS)
	r.Use(recoverer)

	dir := "."
	if len(baseDir) > 0 && baseDir[0] != "" {
		dir = baseDir[0]
	}

	authService := auth.NewService(db, cfg.JWTSecret, cfg.AccessMinutes, cfg.RefreshDays)
	authHandler := auth.NewHandler(authService)
	dashboardHandler := dashboard.NewHandler(db)
	masterdataHandler := masterdata.NewHandler(db)
	ocrHandler := masterdata.NewOCRHandler(cfg.GeminiAPIKey, cfg.GeminiModel)
	financeHandler := finance.NewHandler(db)
	announcementHandler := announcement.NewHandler(db)
	beritaKegiatanHandler := beritakegiatan.NewHandler(db)
	inventarisHandler := inventaris.NewHandler(db, dir)
	peraturanHandler := peraturan.NewHandler(db, dir)
	pengaduanHandler := pengaduan.NewHandler(db)
	pengurusHandler := pengurus.NewHandler(db, dir)
	auditLogHandler := auditlog.NewHandler(db)
	reportHandler := report.NewHandler(db)
	exportHandler := report.NewExportHandler(db)
	userHandler := user.NewHandler(db)
	settingsHandler := settings.NewHandler(db, dir)
	statistikHandler := statistik.NewHandler(db)
	backupHandler := backup.NewHandler(db, cfg, dir)
	loginLimiter := middleware.NewLoginRateLimiter().Middleware

	r.Handle("/uploads/settings/*", http.StripPrefix("/uploads/", http.FileServer(noDirectoryListing{http.Dir(filepath.Join(dir, "uploads"))})))
	r.Handle("/uploads/inventaris/*", http.StripPrefix("/uploads/", http.FileServer(noDirectoryListing{http.Dir(filepath.Join(dir, "uploads"))})))
	r.Handle("/uploads/pengurus/*", http.StripPrefix("/uploads/", http.FileServer(noDirectoryListing{http.Dir(filepath.Join(dir, "uploads"))})))
	r.Handle("/uploads/peraturan/*", http.StripPrefix("/uploads/", http.FileServer(noDirectoryListing{http.Dir(filepath.Join(dir, "uploads"))})))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, http.StatusOK, "OK", map[string]string{"status": "healthy"})
		})

		// public
		auth.RegisterPublic(r, authHandler, loginLimiter)
		announcement.RegisterPublic(r, announcementHandler)
		beritakegiatan.RegisterPublic(r, beritaKegiatanHandler)
		inventaris.RegisterPublic(r, inventarisHandler)
		peraturan.RegisterPublic(r, peraturanHandler)
		pengurus.RegisterPublic(r, pengurusHandler)
		report.RegisterPublic(r, reportHandler)
		statistik.RegisterPublic(r, statistikHandler)
		pengaduan.RegisterPublic(r, pengaduanHandler)
		masterdata.RegisterPublic(r, masterdataHandler)
		settings.RegisterPublic(r, settingsHandler)

		// protected
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.JWTSecret))

			auth.Register(r, authHandler)
			dashboard.Register(r, dashboardHandler)
			masterdata.Register(r, masterdataHandler, ocrHandler)
			finance.Register(r, financeHandler)
			announcement.Register(r, announcementHandler)
			beritakegiatan.Register(r, beritaKegiatanHandler)
			inventaris.Register(r, inventarisHandler)
			peraturan.Register(r, peraturanHandler)
			pengaduan.Register(r, pengaduanHandler)
			pengurus.Register(r, pengurusHandler)
			user.Register(r, userHandler)
			auditlog.Register(r, auditLogHandler)
			backup.Register(r, backupHandler)
			settings.Register(r, settingsHandler)
			report.Register(r, reportHandler, exportHandler)
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

type noDirectoryListing struct {
	fs http.FileSystem
}

func (n noDirectoryListing) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}
