package pengaduan

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

func userName(r *http.Request) string {
	name := middleware.GetUserName(r)
	if name == "" {
		return "Admin"
	}
	return name
}

var allowedExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".pdf": true, ".mp4": true,
}

const maxFileSize = 10 << 20 // 10 MB

func randomFilename(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}

func (h *Handler) SubmitPengaduan(w http.ResponseWriter, r *http.Request) {
	var showPengaduan bool
	h.DB.WithContext(r.Context()).
		Table("website_settings").
		Select("show_pengaduan").
		Scan(&showPengaduan)
	if !showPengaduan {
		response.Error(w, http.StatusForbidden, "Formulir pengaduan sedang ditutup")
		return
	}

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	namaPelapor := strings.TrimSpace(r.FormValue("nama_pelapor"))
	lokasi := strings.TrimSpace(r.FormValue("lokasi"))
	keterangan := strings.TrimSpace(r.FormValue("keterangan"))

	errs := map[string][]string{}
	if namaPelapor == "" {
		errs["nama_pelapor"] = []string{"Nama pelapor wajib diisi"}
	}
	if lokasi == "" {
		errs["lokasi"] = []string{"Lokasi wajib diisi"}
	}
	if keterangan == "" {
		errs["keterangan"] = []string{"Keterangan wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	model := database.Pengaduan{
		NamaPelapor: namaPelapor,
		Lokasi:      lokasi,
		Keterangan:  keterangan,
		Status:      "menunggu",
	}

	file, handler, err := r.FormFile("bukti")
	if err == nil {
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if !allowedExtensions[ext] {
			response.Error(w, http.StatusBadRequest, "Format file tidak didukung. Format yang diizinkan: JPG, JPEG, PNG, PDF, MP4")
			return
		}
		if handler.Size > maxFileSize {
			response.Error(w, http.StatusBadRequest, "Ukuran file maksimal 10 MB")
			return
		}

		uploadDir := "uploads/pengaduan"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		filename, err := randomFilename(ext)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		dstPath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(dstPath)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		model.BuktiFile = &dstPath
	}

	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengaduan", "Kirim Pengaduan",
		"Pengaduan baru dari: "+namaPelapor, nil, map[string]string{
			"nama_pelapor": namaPelapor,
			"lokasi":       lokasi,
		})

	response.Created(w, "Pengaduan berhasil dikirim", map[string]string{"id": model.ID})
}

func (h *Handler) ListPengaduan(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	status := r.URL.Query().Get("status")
	bulan := r.URL.Query().Get("bulan")
	tahun := r.URL.Query().Get("tahun")

	query := h.DB.WithContext(r.Context()).
		Table("pengaduan p").
		Select("p.*").
		Where("p.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(p.nama_pelapor ILIKE ? OR p.lokasi ILIKE ?)", like, like)
	}
	if status != "" {
		query = query.Where("p.status = ?", status)
	}
	if bulan != "" {
		query = query.Where("EXTRACT(MONTH FROM p.created_at) = ?", bulan)
	}
	if tahun != "" {
		query = query.Where("EXTRACT(YEAR FROM p.created_at) = ?", tahun)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("pengaduan p").
		Where("p.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(p.nama_pelapor ILIKE ? OR p.lokasi ILIKE ?)", like, like)
	}
	if status != "" {
		countQuery = countQuery.Where("p.status = ?", status)
	}
	if bulan != "" {
		countQuery = countQuery.Where("EXTRACT(MONTH FROM p.created_at) = ?", bulan)
	}
	if tahun != "" {
		countQuery = countQuery.Where("EXTRACT(YEAR FROM p.created_at) = ?", tahun)
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("p.created_at DESC")
	case "-created_at":
		query = query.Order("p.created_at ASC")
	case "nama_pelapor":
		query = query.Order("p.nama_pelapor ASC")
	case "-nama_pelapor":
		query = query.Order("p.nama_pelapor DESC")
	default:
		query = query.Order("p.created_at DESC")
	}

	list := make([]Pengaduan, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Pengaduan", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetPengaduan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var item Pengaduan
	err := h.DB.WithContext(r.Context()).
		Table("pengaduan p").
		Select("p.*").
		Where("p.id = ? AND p.deleted_at IS NULL", id).
		Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Pengaduan tidak ditemukan")
		return
	}

	var riwayat []PengaduanRiwayat
	h.DB.WithContext(r.Context()).
		Table("pengaduan_riwayat").
		Where("pengaduan_id = ?", id).
		Order("created_at ASC").
		Scan(&riwayat)

	if riwayat == nil {
		riwayat = []PengaduanRiwayat{}
	}

	detail := PengaduanDetail{
		Pengaduan: item,
		Riwayat:   riwayat,
	}

	response.Success(w, http.StatusOK, "Detail Pengaduan", detail)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	if req.Status == "" {
		errs["status"] = []string{"Status wajib diisi"}
	}
	if req.Status != "menunggu" && req.Status != "diproses" && req.Status != "selesai" {
		errs["status"] = []string{"Status harus menunggu, diproses, atau selesai"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var current database.Pengaduan
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&current).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pengaduan tidak ditemukan")
		return
	}

	validTransitions := map[string]string{
		"menunggu": "diproses",
		"diproses": "selesai",
	}
	expectedNext, ok := validTransitions[current.Status]
	if !ok || expectedNext != req.Status {
		errs["status"] = []string{fmt.Sprintf("Tidak dapat mengubah status dari %s ke %s", current.Status, req.Status)}
		response.ValidationError(w, errs)
		return
	}

	catatan := strings.TrimSpace(req.Catatan)
	if catatan == "" {
		catatan = fmt.Sprintf("Status diubah dari %s ke %s", current.Status, req.Status)
	}

	result := h.DB.WithContext(r.Context()).Model(&database.Pengaduan{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"status":     req.Status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	riwayat := database.PengaduanRiwayat{
		PengaduanID:   id,
		Status:        req.Status,
		Catatan:       catatan,
		DitanganiOleh: userName(r),
	}
	h.DB.WithContext(r.Context()).Create(&riwayat)

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengaduan", "Ubah Status Pengaduan",
		fmt.Sprintf("Status pengaduan diubah dari %s ke %s", current.Status, req.Status), current.Status, req.Status)

	response.Success(w, http.StatusOK, "Status pengaduan berhasil diperbarui", nil)
}

func (h *Handler) AddCatatan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req AddCatatanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Catatan = strings.TrimSpace(req.Catatan)
	if req.Catatan == "" {
		errs["catatan"] = []string{"Catatan wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var current database.Pengaduan
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&current).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pengaduan tidak ditemukan")
		return
	}

	riwayat := database.PengaduanRiwayat{
		PengaduanID:   id,
		Status:        current.Status,
		Catatan:       req.Catatan,
		DitanganiOleh: userName(r),
	}
	if err := h.DB.WithContext(r.Context()).Create(&riwayat).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengaduan", "Tambah Catatan",
		"Catatan penanganan ditambahkan untuk pengaduan: "+current.NamaPelapor, nil, req)

	response.Created(w, "Catatan berhasil ditambahkan", map[string]string{"id": riwayat.ID})
}

func (h *Handler) DeletePengaduan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var before database.Pengaduan
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.Pengaduan{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Pengaduan tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengaduan", "Hapus Pengaduan",
		"Menghapus (soft) pengaduan dari: "+before.NamaPelapor, before, nil)
	response.Success(w, http.StatusOK, "Pengaduan berhasil dihapus", nil)
}

func (h *Handler) DownloadBukti(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var item database.Pengaduan
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&item).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pengaduan tidak ditemukan")
		return
	}

	if item.BuktiFile == nil || *item.BuktiFile == "" {
		response.Error(w, http.StatusNotFound, "Tidak ada bukti file")
		return
	}

	filePath := *item.BuktiFile
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		response.Error(w, http.StatusNotFound, "File bukti tidak ditemukan di server")
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".pdf":
		contentType = "application/pdf"
	case ".mp4":
		contentType = "video/mp4"
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"bukti%s\"", ext))
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filePath)
}
