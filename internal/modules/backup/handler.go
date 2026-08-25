package backup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"rt26/backend/internal/config"
	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

const commandTimeout = 10 * time.Minute

type Handler struct {
	DB      *gorm.DB
	Cfg     *config.Config
	BaseDir string
}

func NewHandler(db *gorm.DB, cfg *config.Config, baseDir string) *Handler {
	return &Handler{DB: db, Cfg: cfg, BaseDir: baseDir}
}

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

func (h *Handler) backupPath(filename string) string {
	return filepath.Join(h.BaseDir, h.Cfg.BackupDir, filepath.Base(filename))
}

type BackupItem struct {
	ID            string    `json:"id"`
	NamaFile      string    `json:"nama_file"`
	UkuranFile    int64     `json:"ukuran_file"`
	DibuatOleh    string    `json:"dibuat_oleh"`
	TanggalBackup time.Time `json:"tanggal_backup"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 15)
	search := helper.CleanSearch(r.URL.Query().Get("search"))

	type backupRow struct {
		ID        string
		Filename  string
		FileSize  int64
		Nama      *string
		CreatedAt time.Time
	}

	query := h.DB.WithContext(r.Context()).
		Table("backups b").
		Select("b.id, b.filename, b.file_size, u.username as nama, b.created_at").
		Joins("LEFT JOIN users u ON u.id = b.dibuat_oleh")

	if search != "" {
		query = query.Where("b.filename ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data backup")
		return
	}

	var rows []backupRow
	if err := query.Order("b.created_at DESC").
		Limit(pg.Limit).Offset(pg.Offset()).
		Scan(&rows).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data backup")
		return
	}

	items := make([]BackupItem, 0, len(rows))
	for _, row := range rows {
		nama := "Sistem"
		if row.Nama != nil && *row.Nama != "" {
			nama = *row.Nama
		}
		items = append(items, BackupItem{
			ID:            row.ID,
			NamaFile:      row.Filename,
			UkuranFile:    row.FileSize,
			DibuatOleh:    nama,
			TanggalBackup: row.CreatedAt,
		})
	}

	response.SuccessPaginated(w, "Daftar backup database", items, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	backupDir := filepath.Join(h.BaseDir, h.Cfg.BackupDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menyiapkan direktori backup")
		return
	}

	filename := fmt.Sprintf("backup-rt26-%s.sql", time.Now().Format("20060102-150405"))
	destPath := filepath.Join(backupDir, filename)

	ctx, cancel := context.WithTimeout(r.Context(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.Cfg.PgDumpPath,
		"--host", h.Cfg.DBHost,
		"--port", h.Cfg.DBPort,
		"--username", h.Cfg.DBUser,
		"--no-password",
		"--format=plain",
		"--clean",
		"--if-exists",
		"--exclude-table=backups",
		"--exclude-table=audit_logs",
		"--file", destPath,
		h.Cfg.DBName,
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+h.Cfg.DBPassword)

	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(destPath)
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			response.Error(w, http.StatusInternalServerError, "Gagal membuat backup: "+detail)
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal membuat backup database")
		return
	}

	info, err := os.Stat(destPath)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "File backup tidak ditemukan setelah dibuat")
		return
	}

	record := database.Backup{
		Filename:   filename,
		FileSize:   info.Size(),
		DibuatOleh: userIDPtr(r),
	}
	if err := h.DB.WithContext(r.Context()).Create(&record).Error; err != nil {
		os.Remove(destPath)
		response.Error(w, http.StatusInternalServerError, "Gagal menyimpan riwayat backup")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Backup Database", "Buat Backup",
		"Membuat backup database: "+filename, nil,
		map[string]interface{}{"nama_file": filename, "ukuran_file": info.Size()})

	item := BackupItem{
		ID:            record.ID,
		NamaFile:      record.Filename,
		UkuranFile:    record.FileSize,
		DibuatOleh:    middleware.GetUserName(r),
		TanggalBackup: record.CreatedAt,
	}
	response.Created(w, "Backup database berhasil dibuat", item)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var record database.Backup
	if err := h.DB.WithContext(r.Context()).First(&record, "id = ?", id).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Backup tidak ditemukan")
		return
	}

	path := h.backupPath(record.Filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		response.Error(w, http.StatusNotFound, "File backup tidak tersedia")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Backup Database", "Unduh Backup",
		"Mengunduh file backup: "+record.Filename, nil,
		map[string]string{"nama_file": record.Filename})

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", record.Filename))
	http.ServeFile(w, r, path)
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var record database.Backup
	if err := h.DB.WithContext(r.Context()).First(&record, "id = ?", id).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Backup tidak ditemukan")
		return
	}

	path := h.backupPath(record.Filename)
	if _, err := os.Stat(path); err != nil {
		response.Error(w, http.StatusNotFound, "File backup tidak tersedia")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.Cfg.PsqlPath,
		"--host", h.Cfg.DBHost,
		"--port", h.Cfg.DBPort,
		"--username", h.Cfg.DBUser,
		"--dbname", h.Cfg.DBName,
		"--no-password",
		"--set=ON_ERROR_STOP=1",
		"--file", path,
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+h.Cfg.DBPassword)

	if out, err := cmd.CombinedOutput(); err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			response.Error(w, http.StatusInternalServerError, "Gagal melakukan restore: "+detail)
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal melakukan restore database")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Backup Database", "Restore Database",
		"Restore database dari file: "+record.Filename, nil,
		map[string]string{"nama_file": record.Filename})

	response.Success(w, http.StatusOK, "Database berhasil dipulihkan dari "+record.Filename, nil)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var record database.Backup
	if err := h.DB.WithContext(r.Context()).First(&record, "id = ?", id).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Backup tidak ditemukan")
		return
	}

	path := h.backupPath(record.Filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		response.Error(w, http.StatusInternalServerError, "Gagal menghapus file backup")
		return
	}

	if err := h.DB.WithContext(r.Context()).Unscoped().Delete(&record).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menghapus riwayat backup")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Backup Database", "Hapus Backup",
		"Menghapus file backup: "+record.Filename,
		map[string]string{"nama_file": record.Filename}, nil)

	response.Success(w, http.StatusOK, "Backup berhasil dihapus", nil)
}
