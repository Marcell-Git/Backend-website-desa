package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func normalizeSocialURL(label, raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("URL %s harus diawali http:// atau https://", label)
	}
	return value, nil
}

type mapsFieldError struct {
	field   string
	message string
}

func checkCoordinate(field, label string, v *float64, limit float64) *mapsFieldError {
	if v == nil {
		return nil
	}
	if *v == 0 {
		return &mapsFieldError{field, label + " tidak boleh 0 pada koordinat Indonesia"}
	}
	if *v < -limit || *v > limit {
		return &mapsFieldError{field, fmt.Sprintf("%s harus antara %g dan %g", label, -limit, limit)}
	}
	return nil
}

func applyMaps(req UpdateSettingsRequest, curLat, curLng *float64, updates map[string]interface{}) *mapsFieldError {
	lat, lng := curLat, curLng
	if req.MapsLat != nil {
		lat = req.MapsLat.Value
	}
	if req.MapsLng != nil {
		lng = req.MapsLng.Value
	}

	if err := checkCoordinate("maps_lat", "Latitude", lat, 90); err != nil {
		return err
	}
	if err := checkCoordinate("maps_lng", "Longitude", lng, 180); err != nil {
		return err
	}

	// Salah satu diisi tanpa yang lain akan membuat peta berada di titik yang salah.
	if (lat == nil) != (lng == nil) {
		return &mapsFieldError{"maps_lat", "Latitude dan Longitude harus diisi bersama atau dikosongkan bersama"}
	}

	if req.MapsLat != nil {
		updates["maps_lat"] = lat
	}
	if req.MapsLng != nil {
		updates["maps_lng"] = lng
	}
	return nil
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
		MapsLat:        s.MapsLat,
		MapsLng:        s.MapsLng,
		Facebook:       s.Facebook,
		Instagram:      s.Instagram,
		Whatsapp:       s.Whatsapp,
		Youtube:        s.Youtube,
		Tiktok:         s.Tiktok,
		LogoPath:       s.LogoPath,
		Copyright:      s.Copyright,
		Visi:           s.Visi,
		Misi:           []string(s.Misi),
		ShowPengumuman: s.ShowPengumuman,
		ShowLaporan:    s.ShowLaporan,
		ShowPengaduan:  s.ShowPengaduan,
		ShowVisiMisi:   s.ShowVisiMisi,
		ShowPeraturan:  s.ShowPeraturan,
		ShowStatistik:  s.ShowStatistik,
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
	if req.ShowPeraturan != nil {
		updates["show_peraturan"] = *req.ShowPeraturan
	}
	if req.ShowStatistik != nil {
		updates["show_statistik"] = *req.ShowStatistik
	}

	socialErrs := map[string][]string{}
	for _, field := range []struct {
		key   string
		label string
		value *string
	}{
		{"facebook", "Facebook", req.Facebook},
		{"instagram", "Instagram", req.Instagram},
		{"whatsapp", "WhatsApp", req.Whatsapp},
		{"youtube", "YouTube", req.Youtube},
		{"tiktok", "TikTok", req.Tiktok},
	} {
		if field.value == nil {
			continue
		}
		url, err := normalizeSocialURL(field.label, *field.value)
		if err != nil {
			socialErrs[field.key] = []string{err.Error()}
			continue
		}
		updates[field.key] = url
	}
	if len(socialErrs) > 0 {
		response.ValidationError(w, socialErrs)
		return
	}

	if err := applyMaps(req, s.MapsLat, s.MapsLng, updates); err != nil {
		errs := map[string][]string{err.field: {err.message}}
		response.ValidationError(w, errs)
		return
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
