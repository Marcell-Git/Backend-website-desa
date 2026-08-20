package announcement

import (
	"encoding/json"
	"errors"
	"net/http"
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

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("tanggal wajib diisi")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, errors.New("format tanggal harus YYYY-MM-DD")
	}
	return t, nil
}

func (h *Handler) ListPengumuman(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	status := r.URL.Query().Get("status")
	bulan := r.URL.Query().Get("bulan")
	tahun := r.URL.Query().Get("tahun")

	query := h.DB.WithContext(r.Context()).
		Table("pengumuman p").
		Select("p.*, u.nama as nama_user").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL").
		Where("p.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(p.judul ILIKE ? OR p.isi ILIKE ?)", like, like)
	}
	if status != "" {
		query = query.Where("p.status = ?", status)
	}
	if bulan != "" {
		query = query.Where("EXTRACT(MONTH FROM p.tanggal_publikasi) = ?", bulan)
	}
	if tahun != "" {
		query = query.Where("EXTRACT(YEAR FROM p.tanggal_publikasi) = ?", tahun)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("pengumuman p").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL").
		Where("p.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(p.judul ILIKE ? OR p.isi ILIKE ?)", like, like)
	}
	if status != "" {
		countQuery = countQuery.Where("p.status = ?", status)
	}
	if bulan != "" {
		countQuery = countQuery.Where("EXTRACT(MONTH FROM p.tanggal_publikasi) = ?", bulan)
	}
	if tahun != "" {
		countQuery = countQuery.Where("EXTRACT(YEAR FROM p.tanggal_publikasi) = ?", tahun)
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("p.created_at DESC")
	case "-created_at":
		query = query.Order("p.created_at ASC")
	case "judul":
		query = query.Order("p.judul ASC")
	case "-judul":
		query = query.Order("p.judul DESC")
	default:
		query = query.Order("p.tanggal_publikasi DESC, p.created_at DESC")
	}

	list := make([]Pengumuman, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Pengumuman", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetPengumuman(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	type pengumumanDetail struct {
		Pengumuman
		NamaUser string `json:"nama_user"`
	}

	var item pengumumanDetail
	err := h.DB.WithContext(r.Context()).
		Table("pengumuman p").
		Select("p.*, u.nama as nama_user").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL").
		Where("p.id = ? AND p.deleted_at IS NULL", id).
		Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Pengumuman tidak ditemukan")
		return
	}

	response.Success(w, http.StatusOK, "Detail Pengumuman", item)
}

func (h *Handler) CreatePengumuman(w http.ResponseWriter, r *http.Request) {
	var req CreatePengumumanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Judul = strings.TrimSpace(req.Judul)
	if req.Judul == "" {
		errs["judul"] = []string{"Judul wajib diisi"}
	}
	req.Isi = strings.TrimSpace(req.Isi)
	if req.Isi == "" {
		errs["isi"] = []string{"Isi pengumuman wajib diisi"}
	}
	tanggal, err := parseDate(req.TanggalPublikasi)
	if err != nil {
		errs["tanggal_publikasi"] = []string{err.Error()}
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.Status != "draft" && req.Status != "published" {
		errs["status"] = []string{"Status harus draft atau published"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	model := database.Pengumuman{
		Judul:            req.Judul,
		Isi:              req.Isi,
		TanggalPublikasi: tanggal,
		Status:           req.Status,
		DibuatOleh:       *userIDPtr(r),
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengumuman", "Tambah Pengumuman",
		"Menambahkan pengumuman: "+req.Judul, nil, req)
	response.Created(w, "Pengumuman berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) UpdatePengumuman(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdatePengumumanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Judul = strings.TrimSpace(req.Judul)
	if req.Judul == "" {
		errs["judul"] = []string{"Judul wajib diisi"}
	}
	req.Isi = strings.TrimSpace(req.Isi)
	if req.Isi == "" {
		errs["isi"] = []string{"Isi pengumuman wajib diisi"}
	}
	tanggal, err := parseDate(req.TanggalPublikasi)
	if err != nil {
		errs["tanggal_publikasi"] = []string{err.Error()}
	}
	if req.Status != "draft" && req.Status != "published" {
		errs["status"] = []string{"Status harus draft atau published"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var before database.Pengumuman
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.Pengumuman{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"judul":              req.Judul,
			"isi":                req.Isi,
			"tanggal_publikasi":  tanggal,
			"status":             req.Status,
			"updated_at":         time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Pengumuman tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengumuman", "Ubah Pengumuman",
		"Memperbarui pengumuman: "+req.Judul, before, req)
	response.Success(w, http.StatusOK, "Pengumuman berhasil diperbarui", nil)
}

func (h *Handler) DeletePengumuman(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var before database.Pengumuman
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.Pengumuman{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Pengumuman tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengumuman", "Hapus Pengumuman",
		"Menghapus (soft) pengumuman: "+before.Judul, before, nil)
	response.Success(w, http.StatusOK, "Pengumuman berhasil dihapus", nil)
}

func (h *Handler) ListPublished(w http.ResponseWriter, r *http.Request) {
	query := h.DB.WithContext(r.Context()).
		Table("pengumuman p").
		Select("p.id, p.judul, p.isi, p.tanggal_publikasi, p.status, u.nama as nama_user, p.created_at").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL").
		Where("p.deleted_at IS NULL AND p.status = 'published'").
		Order("p.tanggal_publikasi DESC")

	list := make([]Pengumuman, 0)
	if err := query.Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.Success(w, http.StatusOK, "Daftar Pengumuman", list)
}
