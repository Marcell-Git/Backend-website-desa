package finance

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

func (h *Handler) ListIuran(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	bulan := r.URL.Query().Get("bulan")
	tahun := r.URL.Query().Get("tahun")
	jenisIuranID := r.URL.Query().Get("jenis_iuran_id")

	type iuranRow struct {
		RekapIuran
		NamaJenisIuran string `json:"nama_jenis_iuran"`
		NamaUser       string `json:"nama_user"`
	}

	query := h.DB.WithContext(r.Context()).
		Table("rekap_iuran i").
		Select("i.*, ji.nama as nama_jenis_iuran, u.nama as nama_user").
		Joins("LEFT JOIN jenis_iuran ji ON ji.id = i.jenis_iuran_id AND ji.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = i.dibuat_oleh AND u.deleted_at IS NULL").
		Where("i.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(ji.nama ILIKE ? OR i.keterangan ILIKE ?)", like, like)
	}
	if bulan != "" {
		query = query.Where("EXTRACT(MONTH FROM i.tanggal) = ?", bulan)
	}
	if tahun != "" {
		query = query.Where("EXTRACT(YEAR FROM i.tanggal) = ?", tahun)
	}
	if jenisIuranID != "" {
		query = query.Where("i.jenis_iuran_id = ?", jenisIuranID)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("rekap_iuran i").
		Joins("LEFT JOIN jenis_iuran ji ON ji.id = i.jenis_iuran_id AND ji.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = i.dibuat_oleh AND u.deleted_at IS NULL").
		Where("i.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(ji.nama ILIKE ? OR i.keterangan ILIKE ?)", like, like)
	}
	if bulan != "" {
		countQuery = countQuery.Where("EXTRACT(MONTH FROM i.tanggal) = ?", bulan)
	}
	if tahun != "" {
		countQuery = countQuery.Where("EXTRACT(YEAR FROM i.tanggal) = ?", tahun)
	}
	if jenisIuranID != "" {
		countQuery = countQuery.Where("i.jenis_iuran_id = ?", jenisIuranID)
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("i.created_at ASC")
	case "-created_at":
		query = query.Order("i.created_at DESC")
	case "nominal":
		query = query.Order("i.nominal ASC")
	case "-nominal":
		query = query.Order("i.nominal DESC")
	default:
		query = query.Order("i.tanggal DESC, i.created_at DESC")
	}

	list := make([]iuranRow, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Rekap Iuran", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetIuran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	type iuranDetail struct {
		RekapIuran
		NamaJenisIuran string `json:"nama_jenis_iuran"`
		NamaUser       string `json:"nama_user"`
	}

	var item iuranDetail
	err := h.DB.WithContext(r.Context()).
		Table("rekap_iuran i").
		Select("i.*, ji.nama as nama_jenis_iuran, u.nama as nama_user").
		Joins("LEFT JOIN jenis_iuran ji ON ji.id = i.jenis_iuran_id AND ji.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = i.dibuat_oleh AND u.deleted_at IS NULL").
		Where("i.id = ? AND i.deleted_at IS NULL", id).
		Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Rekap iuran tidak ditemukan")
		return
	}

	response.Success(w, http.StatusOK, "Detail Rekap Iuran", item)
}

func (h *Handler) CreateIuran(w http.ResponseWriter, r *http.Request) {
	var req CreateRekapIuranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	tanggal, err := parseDate(req.Tanggal)
	if err != nil {
		errs["tanggal"] = []string{err.Error()}
	}
	if strings.TrimSpace(req.JenisIuranID) == "" {
		errs["jenis_iuran_id"] = []string{"Jenis iuran wajib dipilih"}
	}
	if req.Nominal <= 0 {
		errs["nominal"] = []string{"Nominal harus lebih besar dari 0"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var jiCount int64
	h.DB.WithContext(r.Context()).Model(&database.JenisIuran{}).
		Where("id = ? AND deleted_at IS NULL", req.JenisIuranID).Count(&jiCount)
	if jiCount == 0 {
		response.ValidationError(w, map[string][]string{"jenis_iuran_id": {"Jenis iuran tidak ditemukan"}})
		return
	}

	model := database.RekapIuran{
		Tanggal:      tanggal,
		JenisIuranID: req.JenisIuranID,
		Nominal:      req.Nominal,
		Keterangan:   req.Keterangan,
		DibuatOleh:   *userIDPtr(r),
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Iuran", "Tambah Iuran",
		"Menambahkan rekap iuran", nil, req)
	response.Created(w, "Rekap iuran berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) UpdateIuran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRekapIuranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	tanggal, err := parseDate(req.Tanggal)
	if err != nil {
		errs["tanggal"] = []string{err.Error()}
	}
	if strings.TrimSpace(req.JenisIuranID) == "" {
		errs["jenis_iuran_id"] = []string{"Jenis iuran wajib dipilih"}
	}
	if req.Nominal <= 0 {
		errs["nominal"] = []string{"Nominal harus lebih besar dari 0"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var jiCount int64
	h.DB.WithContext(r.Context()).Model(&database.JenisIuran{}).
		Where("id = ? AND deleted_at IS NULL", req.JenisIuranID).Count(&jiCount)
	if jiCount == 0 {
		response.ValidationError(w, map[string][]string{"jenis_iuran_id": {"Jenis iuran tidak ditemukan"}})
		return
	}

	var before database.RekapIuran
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.RekapIuran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"tanggal":        tanggal,
			"jenis_iuran_id": req.JenisIuranID,
			"nominal":        req.Nominal,
			"keterangan":     req.Keterangan,
			"updated_at":     time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Rekap iuran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Iuran", "Ubah Iuran",
		"Memperbarui rekap iuran", before, req)
	response.Success(w, http.StatusOK, "Rekap iuran berhasil diperbarui", nil)
}

func (h *Handler) DeleteIuran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var before database.RekapIuran
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.RekapIuran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Rekap iuran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Iuran", "Hapus Iuran",
		"Menghapus (soft) rekap iuran", before, nil)
	response.Success(w, http.StatusOK, "Rekap iuran berhasil dihapus", nil)
}
