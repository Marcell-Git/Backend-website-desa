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

func (h *Handler) ListKategoriPengeluaran(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))

	query := h.DB.WithContext(r.Context()).
		Table("kategori_pengeluaran").
		Where("deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("nama ILIKE ?", like)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("kategori_pengeluaran").
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

	list := make([]KategoriPengeluaran, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Kategori Pengeluaran", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) CreateKategoriPengeluaran(w http.ResponseWriter, r *http.Request) {
	var req CreateKategoriPengeluaranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama kategori pengeluaran wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	model := database.KategoriPengeluaran{Nama: req.Nama, Deskripsi: req.Deskripsi}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"nama": {"Nama kategori pengeluaran sudah ada"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Kategori Pengeluaran", "Tambah Kategori",
		"Menambahkan kategori pengeluaran: "+req.Nama, nil, req)
	response.Created(w, "Kategori pengeluaran berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) UpdateKategoriPengeluaran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateKategoriPengeluaranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama kategori pengeluaran wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.KategoriPengeluaran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"nama":       req.Nama,
			"deskripsi":  req.Deskripsi,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if helper.IsDuplicateErrorGORM(result.Error) {
			response.ValidationError(w, map[string][]string{"nama": {"Nama kategori pengeluaran sudah ada"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Kategori pengeluaran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Kategori Pengeluaran", "Ubah Kategori",
		"Memperbarui kategori pengeluaran", nil, req)
	response.Success(w, http.StatusOK, "Kategori pengeluaran berhasil diperbarui", nil)
}

func (h *Handler) DeleteKategoriPengeluaran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.RekapPengeluaran{}).
		Where("kategori_pengeluaran_id = ? AND deleted_at IS NULL", id).Count(&count)
	if count > 0 {
		response.Error(w, http.StatusConflict, "Kategori pengeluaran tidak dapat dihapus karena masih digunakan dalam rekap pengeluaran")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.KategoriPengeluaran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Kategori pengeluaran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Kategori Pengeluaran", "Hapus Kategori",
		"Menghapus (soft) kategori pengeluaran", nil, map[string]string{"id": id})
	response.Success(w, http.StatusOK, "Kategori pengeluaran berhasil dihapus", nil)
}
