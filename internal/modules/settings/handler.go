package settings

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	DB      *gorm.DB
	BaseDir string
}

func NewHandler(db *gorm.DB, baseDir string) *Handler {
	return &Handler{DB: db, BaseDir: baseDir}
}

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	var s database.WebsiteSettings
	if err := h.DB.WithContext(r.Context()).First(&s).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat pengaturan")
		return
	}

	resp := WebsiteSettingsResponse{
		ID:             s.ID,
		NamaWebsite:    s.NamaWebsite,
		NamaRT:         s.NamaRT,
		Desa:           s.Desa,
		Kecamatan:      s.Kecamatan,
		Kabupaten:      s.Kabupaten,
		Provinsi:       s.Provinsi,
		Telepon:        s.Telepon,
		Email:          s.Email,
		Alamat:         s.Alamat,
		LogoPath:       s.LogoPath,
		Copyright:      s.Copyright,
		Visi:           s.Visi,
		Misi:           []string(s.Misi),
		ShowPengumuman: s.ShowPengumuman,
		ShowLaporan:    s.ShowLaporan,
		ShowPengaduan:  s.ShowPengaduan,
		ShowVisiMisi:   s.ShowVisiMisi,
	}
	response.Success(w, http.StatusOK, "Pengaturan website", resp)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	var s database.WebsiteSettings
	if err := h.DB.WithContext(r.Context()).First(&s).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat pengaturan")
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.NamaWebsite != nil {
		updates["nama_website"] = strings.TrimSpace(*req.NamaWebsite)
	}
	if req.NamaRT != nil {
		updates["nama_rt"] = strings.TrimSpace(*req.NamaRT)
	}
	if req.Desa != nil {
		updates["desa"] = strings.TrimSpace(*req.Desa)
	}
	if req.Kecamatan != nil {
		updates["kecamatan"] = strings.TrimSpace(*req.Kecamatan)
	}
	if req.Kabupaten != nil {
		updates["kabupaten"] = strings.TrimSpace(*req.Kabupaten)
	}
	if req.Provinsi != nil {
		updates["provinsi"] = strings.TrimSpace(*req.Provinsi)
	}
	if req.Telepon != nil {
		updates["telepon"] = strings.TrimSpace(*req.Telepon)
	}
	if req.Email != nil {
		updates["email"] = strings.TrimSpace(*req.Email)
	}
	if req.Alamat != nil {
		updates["alamat"] = strings.TrimSpace(*req.Alamat)
	}
	if req.Copyright != nil {
		updates["copyright"] = strings.TrimSpace(*req.Copyright)
	}
	if req.Visi != nil {
		updates["visi"] = strings.TrimSpace(*req.Visi)
	}
	if req.Misi != nil {
		misi := make(database.StringList, 0, len(*req.Misi))
		for _, item := range *req.Misi {
			if item = strings.TrimSpace(item); item != "" {
				misi = append(misi, item)
			}
		}
		updates["misi"] = misi
	}
	if req.ShowPengumuman != nil {
		updates["show_pengumuman"] = *req.ShowPengumuman
	}
	if req.ShowLaporan != nil {
		updates["show_laporan"] = *req.ShowLaporan
	}
	if req.ShowPengaduan != nil {
		updates["show_pengaduan"] = *req.ShowPengaduan
	}
	if req.ShowVisiMisi != nil {
		updates["show_visi_misi"] = *req.ShowVisiMisi
	}

	result := h.DB.WithContext(r.Context()).Model(&s).Updates(updates)
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memperbarui pengaturan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Website Settings", "Ubah Pengaturan",
		"Memperbarui pengaturan website", s, updates)
	response.Success(w, http.StatusOK, "Pengaturan berhasil diperbarui", nil)
}

func (h *Handler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	h.uploadFile(w, r, "logo", 5*1024*1024, []string{".png", ".jpg", ".jpeg", ".svg"})
}

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request, field string, maxSize int64, allowedExts []string) {
	if err := r.ParseMultipartForm(maxSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Ukuran file terlalu besar")
		return
	}

	file, header, err := r.FormFile(field)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "File tidak ditemukan")
		return
	}
	defer file.Close()

	if header.Size > maxSize {
		response.Error(w, http.StatusBadRequest, "Ukuran file terlalu besar")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := false
	for _, e := range allowedExts {
		if ext == e {
			allowed = true
			break
		}
	}
	if !allowed {
		response.Error(w, http.StatusBadRequest, "Format file tidak diizinkan")
		return
	}

	uploadDir := filepath.Join(h.BaseDir, "uploads", "settings")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal membuat direktori upload")
		return
	}

	ctx := context.Background()
	var s database.WebsiteSettings
	if err := h.DB.WithContext(ctx).First(&s).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat pengaturan")
		return
	}

	var oldPath string
	if field == "logo" && s.LogoPath != "" {
		oldPath = filepath.Join(h.BaseDir, s.LogoPath)
	}

	filename := field + "_" + time.Now().Format("20060102150405") + ext
	destPath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(destPath)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menyimpan file")
		return
	}

	relPath := "uploads/settings/" + filename
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if field == "logo" {
		updates["logo_path"] = relPath
	}

	result := h.DB.WithContext(r.Context()).Model(&s).Updates(updates)
	if result.Error != nil {
		os.Remove(destPath)
		response.Error(w, http.StatusInternalServerError, "Gagal memperbarui pengaturan")
		return
	}

	if oldPath != "" {
		os.Remove(oldPath)
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Website Settings", "Upload "+field,
		"Upload file "+field+": "+header.Filename, nil, map[string]string{"path": relPath})
	response.Success(w, http.StatusOK, field+" berhasil diupload", map[string]string{"path": relPath})
}
