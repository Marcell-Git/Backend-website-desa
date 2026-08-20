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

func (h *Handler) ListKas(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	bulan := r.URL.Query().Get("bulan")
	tahun := r.URL.Query().Get("tahun")

	type kasRow struct {
		RekapKas
		NamaUser string `json:"nama_user"`
	}

	query := h.DB.WithContext(r.Context()).
		Table("rekap_kas k").
		Select("k.*, u.nama as nama_user").
		Joins("LEFT JOIN users u ON u.id = k.dibuat_oleh AND u.deleted_at IS NULL").
		Where("k.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(k.sumber_kas ILIKE ? OR k.keterangan ILIKE ?)", like, like)
	}
	if bulan != "" {
		query = query.Where("EXTRACT(MONTH FROM k.tanggal) = ?", bulan)
	}
	if tahun != "" {
		query = query.Where("EXTRACT(YEAR FROM k.tanggal) = ?", tahun)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("rekap_kas k").
		Joins("LEFT JOIN users u ON u.id = k.dibuat_oleh AND u.deleted_at IS NULL").
		Where("k.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(k.sumber_kas ILIKE ? OR k.keterangan ILIKE ?)", like, like)
	}
	if bulan != "" {
		countQuery = countQuery.Where("EXTRACT(MONTH FROM k.tanggal) = ?", bulan)
	}
	if tahun != "" {
		countQuery = countQuery.Where("EXTRACT(YEAR FROM k.tanggal) = ?", tahun)
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("k.created_at DESC")
	case "-created_at":
		query = query.Order("k.created_at ASC")
	case "nominal":
		query = query.Order("k.nominal DESC")
	case "-nominal":
		query = query.Order("k.nominal ASC")
	default:
		query = query.Order("k.tanggal DESC, k.created_at DESC")
	}

	list := make([]kasRow, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Rekap Kas", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetKas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	type kasDetail struct {
		RekapKas
		NamaUser string `json:"nama_user"`
	}

	var item kasDetail
	err := h.DB.WithContext(r.Context()).
		Table("rekap_kas k").
		Select("k.*, u.nama as nama_user").
		Joins("LEFT JOIN users u ON u.id = k.dibuat_oleh AND u.deleted_at IS NULL").
		Where("k.id = ? AND k.deleted_at IS NULL", id).
		Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Rekap kas tidak ditemukan")
		return
	}

	response.Success(w, http.StatusOK, "Detail Rekap Kas", item)
}

func (h *Handler) CreateKas(w http.ResponseWriter, r *http.Request) {
	var req CreateRekapKasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	tanggal, err := parseDate(req.Tanggal)
	if err != nil {
		errs["tanggal"] = []string{err.Error()}
	}
	req.SumberKas = strings.TrimSpace(req.SumberKas)
	if req.SumberKas == "" {
		errs["sumber_kas"] = []string{"Sumber kas wajib diisi"}
	}
	if req.Nominal <= 0 {
		errs["nominal"] = []string{"Nominal harus lebih besar dari 0"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	model := database.RekapKas{
		Tanggal:    tanggal,
		SumberKas:  req.SumberKas,
		Nominal:    req.Nominal,
		Keterangan: req.Keterangan,
		DibuatOleh: *userIDPtr(r),
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Kas", "Tambah Kas",
		"Menambahkan rekap kas: "+req.SumberKas, nil, req)
	response.Created(w, "Rekap kas berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) UpdateKas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRekapKasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	tanggal, err := parseDate(req.Tanggal)
	if err != nil {
		errs["tanggal"] = []string{err.Error()}
	}
	req.SumberKas = strings.TrimSpace(req.SumberKas)
	if req.SumberKas == "" {
		errs["sumber_kas"] = []string{"Sumber kas wajib diisi"}
	}
	if req.Nominal <= 0 {
		errs["nominal"] = []string{"Nominal harus lebih besar dari 0"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var before database.RekapKas
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.RekapKas{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"tanggal":     tanggal,
			"sumber_kas":  req.SumberKas,
			"nominal":     req.Nominal,
			"keterangan":  req.Keterangan,
			"updated_at":  time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Rekap kas tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Kas", "Ubah Kas",
		"Memperbarui rekap kas", before, req)
	response.Success(w, http.StatusOK, "Rekap kas berhasil diperbarui", nil)
}

func (h *Handler) DeleteKas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var before database.RekapKas
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.RekapKas{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Rekap kas tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Kas", "Hapus Kas",
		"Menghapus (soft) rekap kas", before, nil)
	response.Success(w, http.StatusOK, "Rekap kas berhasil dihapus", nil)
}
