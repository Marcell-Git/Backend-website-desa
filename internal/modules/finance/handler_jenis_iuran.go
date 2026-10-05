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

func (h *Handler) ListJenisIuran(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))

	query := h.DB.WithContext(r.Context()).
		Table("jenis_iuran").
		Where("deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("nama ILIKE ?", like)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("jenis_iuran").
		Where("deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("nama ILIKE ?", like)
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("created_at DESC")
	case "-created_at":
		query = query.Order("created_at ASC")
	default:
		query = query.Order("created_at DESC")
	}

	list := make([]JenisIuran, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Jenis Iuran", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) CreateJenisIuran(w http.ResponseWriter, r *http.Request) {
	var req CreateJenisIuranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama jenis iuran wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	model := database.JenisIuran{Nama: req.Nama, Deskripsi: req.Deskripsi}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"nama": {"Nama jenis iuran sudah ada"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Jenis Iuran", "Tambah Jenis Iuran",
		"Menambahkan jenis iuran: "+req.Nama, nil, req)
	response.Created(w, "Jenis iuran berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) UpdateJenisIuran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateJenisIuranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama jenis iuran wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.JenisIuran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"nama":       req.Nama,
			"deskripsi":  req.Deskripsi,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if helper.IsDuplicateErrorGORM(result.Error) {
			response.ValidationError(w, map[string][]string{"nama": {"Nama jenis iuran sudah ada"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Jenis iuran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Jenis Iuran", "Ubah Jenis Iuran",
		"Memperbarui jenis iuran", nil, req)
	response.Success(w, http.StatusOK, "Jenis iuran berhasil diperbarui", nil)
}

func (h *Handler) DeleteJenisIuran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.RekapIuran{}).
		Where("jenis_iuran_id = ? AND deleted_at IS NULL", id).Count(&count)
	if count > 0 {
		response.Error(w, http.StatusConflict, "Jenis iuran tidak dapat dihapus karena masih digunakan dalam rekap iuran")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.JenisIuran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Jenis iuran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Jenis Iuran", "Hapus Jenis Iuran",
		"Menghapus (soft) jenis iuran", nil, map[string]string{"id": id})
	response.Success(w, http.StatusOK, "Jenis iuran berhasil dihapus", nil)
}
