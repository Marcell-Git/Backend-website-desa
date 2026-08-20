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

func (h *Handler) ListPengeluaran(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	bulan := r.URL.Query().Get("bulan")
	tahun := r.URL.Query().Get("tahun")
	kategoriID := r.URL.Query().Get("kategori_pengeluaran_id")

	type pengeluaranRow struct {
		RekapPengeluaran
		NamaKategoriPengeluaran string `json:"nama_kategori_pengeluaran"`
		NamaUser                string `json:"nama_user"`
	}

	query := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran p").
		Select("p.*, kp.nama as nama_kategori_pengeluaran, u.nama as nama_user").
		Joins("LEFT JOIN kategori_pengeluaran kp ON kp.id = p.kategori_pengeluaran_id AND kp.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL").
		Where("p.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(kp.nama ILIKE ? OR p.keterangan ILIKE ?)", like, like)
	}
	if bulan != "" {
		query = query.Where("EXTRACT(MONTH FROM p.tanggal) = ?", bulan)
	}
	if tahun != "" {
		query = query.Where("EXTRACT(YEAR FROM p.tanggal) = ?", tahun)
	}
	if kategoriID != "" {
		query = query.Where("p.kategori_pengeluaran_id = ?", kategoriID)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran p").
		Joins("LEFT JOIN kategori_pengeluaran kp ON kp.id = p.kategori_pengeluaran_id AND kp.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL").
		Where("p.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(kp.nama ILIKE ? OR p.keterangan ILIKE ?)", like, like)
	}
	if bulan != "" {
		countQuery = countQuery.Where("EXTRACT(MONTH FROM p.tanggal) = ?", bulan)
	}
	if tahun != "" {
		countQuery = countQuery.Where("EXTRACT(YEAR FROM p.tanggal) = ?", tahun)
	}
	if kategoriID != "" {
		countQuery = countQuery.Where("p.kategori_pengeluaran_id = ?", kategoriID)
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("p.created_at ASC")
	case "-created_at":
		query = query.Order("p.created_at DESC")
	case "nominal":
		query = query.Order("p.nominal ASC")
	case "-nominal":
		query = query.Order("p.nominal DESC")
	default:
		query = query.Order("p.tanggal DESC, p.created_at DESC")
	}

	list := make([]pengeluaranRow, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Rekap Pengeluaran", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetPengeluaran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	type pengeluaranDetail struct {
		RekapPengeluaran
		NamaKategoriPengeluaran string `json:"nama_kategori_pengeluaran"`
		NamaUser                string `json:"nama_user"`
	}

	var item pengeluaranDetail
	err := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran p").
		Select("p.*, kp.nama as nama_kategori_pengeluaran, u.nama as nama_user").
		Joins("LEFT JOIN kategori_pengeluaran kp ON kp.id = p.kategori_pengeluaran_id AND kp.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL").
		Where("p.id = ? AND p.deleted_at IS NULL", id).
		Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Rekap pengeluaran tidak ditemukan")
		return
	}

	response.Success(w, http.StatusOK, "Detail Rekap Pengeluaran", item)
}

func (h *Handler) CreatePengeluaran(w http.ResponseWriter, r *http.Request) {
	var req CreateRekapPengeluaranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	tanggal, err := parseDate(req.Tanggal)
	if err != nil {
		errs["tanggal"] = []string{err.Error()}
	}
	if strings.TrimSpace(req.KategoriPengeluaranID) == "" {
		errs["kategori_pengeluaran_id"] = []string{"Kategori pengeluaran wajib dipilih"}
	}
	if req.Nominal <= 0 {
		errs["nominal"] = []string{"Nominal harus lebih besar dari 0"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var kpCount int64
	h.DB.WithContext(r.Context()).Model(&database.KategoriPengeluaran{}).
		Where("id = ? AND deleted_at IS NULL", req.KategoriPengeluaranID).Count(&kpCount)
	if kpCount == 0 {
		response.ValidationError(w, map[string][]string{"kategori_pengeluaran_id": {"Kategori pengeluaran tidak ditemukan"}})
		return
	}

	model := database.RekapPengeluaran{
		Tanggal:                tanggal,
		KategoriPengeluaranID: req.KategoriPengeluaranID,
		Nominal:               req.Nominal,
		Keterangan:            req.Keterangan,
		DibuatOleh:            *userIDPtr(r),
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Pengeluaran", "Tambah Pengeluaran",
		"Menambahkan rekap pengeluaran", nil, req)
	response.Created(w, "Rekap pengeluaran berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) UpdatePengeluaran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRekapPengeluaranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	tanggal, err := parseDate(req.Tanggal)
	if err != nil {
		errs["tanggal"] = []string{err.Error()}
	}
	if strings.TrimSpace(req.KategoriPengeluaranID) == "" {
		errs["kategori_pengeluaran_id"] = []string{"Kategori pengeluaran wajib dipilih"}
	}
	if req.Nominal <= 0 {
		errs["nominal"] = []string{"Nominal harus lebih besar dari 0"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var kpCount int64
	h.DB.WithContext(r.Context()).Model(&database.KategoriPengeluaran{}).
		Where("id = ? AND deleted_at IS NULL", req.KategoriPengeluaranID).Count(&kpCount)
	if kpCount == 0 {
		response.ValidationError(w, map[string][]string{"kategori_pengeluaran_id": {"Kategori pengeluaran tidak ditemukan"}})
		return
	}

	var before database.RekapPengeluaran
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.RekapPengeluaran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"tanggal":                  tanggal,
			"kategori_pengeluaran_id": req.KategoriPengeluaranID,
			"nominal":                 req.Nominal,
			"keterangan":              req.Keterangan,
			"updated_at":              time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Rekap pengeluaran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Pengeluaran", "Ubah Pengeluaran",
		"Memperbarui rekap pengeluaran", before, req)
	response.Success(w, http.StatusOK, "Rekap pengeluaran berhasil diperbarui", nil)
}

func (h *Handler) DeletePengeluaran(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var before database.RekapPengeluaran
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&before)

	result := h.DB.WithContext(r.Context()).Model(&database.RekapPengeluaran{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Rekap pengeluaran tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Pengeluaran", "Hapus Pengeluaran",
		"Menghapus (soft) rekap pengeluaran", before, nil)
	response.Success(w, http.StatusOK, "Rekap pengeluaran berhasil dihapus", nil)
}
