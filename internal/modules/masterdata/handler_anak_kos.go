package masterdata

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

func validateAnakKos(req CreateAnakKosRequest) []string {
	errs := []string{}
	if !validNIK(req.NIK) {
		errs = append(errs, "NIK harus 16 digit angka")
	}
	if strings.TrimSpace(req.NamaLengkap) == "" {
		errs = append(errs, "Nama lengkap wajib diisi")
	}
	if req.JenisKelamin != "L" && req.JenisKelamin != "P" {
		errs = append(errs, "Jenis kelamin harus L atau P")
	}
	if _, err := parseDate(req.TanggalLahir); err != nil {
		errs = append(errs, err.Error())
	}
	if strings.TrimSpace(req.Agama) == "" {
		errs = append(errs, "Agama wajib diisi")
	}
	if strings.TrimSpace(req.AlamatAsal) == "" {
		errs = append(errs, "Alamat asal wajib diisi")
	}
	return errs
}

func (h *Handler) ListAnakKos(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	jk := r.URL.Query().Get("jenis_kelamin")
	agama := r.URL.Query().Get("agama")
	status := r.URL.Query().Get("status")

	query := h.DB.WithContext(r.Context()).Model(&database.AnakKos{}).
		Where("deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(nik ILIKE ? OR nama_lengkap ILIKE ? OR nomor_hp ILIKE ? OR pekerjaan ILIKE ? OR alamat_asal ILIKE ?)", like, like, like, like, like)
	}
	if jk == "L" || jk == "P" {
		query = query.Where("jenis_kelamin = ?", jk)
	}
	if agama != "" {
		query = query.Where("agama = ?", agama)
	}
	if status == "true" || status == "false" {
		query = query.Where("status = ?", status == "true")
	}

	var total int64
	query.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("created_at DESC")
	case "-created_at":
		query = query.Order("created_at ASC")
	case "nik":
		query = query.Order("nik DESC")
	default:
		query = query.Order("created_at DESC")
	}

	list := make([]AnakKosData, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Find(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Anak Kos", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetAnakKos(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var a AnakKosData
	err := h.DB.WithContext(r.Context()).
		Model(&database.AnakKos{}).
		Where("id = ? AND deleted_at IS NULL", id).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Data anak kos tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.Success(w, http.StatusOK, "Detail Anak Kos", a)
}

func (h *Handler) CreateAnakKos(w http.ResponseWriter, r *http.Request) {
	var req CreateAnakKosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	req.NIK = strings.TrimSpace(req.NIK)
	if errs := validateAnakKos(req); len(errs) > 0 {
		response.ValidationError(w, map[string][]string{"anak_kos": errs})
		return
	}

	status := true
	if req.Status != nil {
		status = *req.Status
	}

	ak := database.AnakKos{
		NIK:          req.NIK,
		NamaLengkap:  req.NamaLengkap,
		JenisKelamin: req.JenisKelamin,
		TanggalLahir: req.TanggalLahir,
		Agama:        req.Agama,
		Pekerjaan:    req.Pekerjaan,
		NomorHP:      req.NomorHP,
		AlamatAsal:   req.AlamatAsal,
		Status:       status,
	}

	if err := h.DB.WithContext(r.Context()).Create(&ak).Error; err != nil {
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"nik": {"NIK telah terdaftar"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Tambah Anak Kos",
		"Menambahkan anak kos "+req.NamaLengkap, nil, req)
	response.Created(w, "Anak kos berhasil ditambahkan", map[string]string{"id": ak.ID})
}

func (h *Handler) UpdateAnakKos(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateAnakKosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	if strings.TrimSpace(req.NamaLengkap) == "" {
		errs["nama_lengkap"] = []string{"Nama lengkap wajib diisi"}
	}
	if strings.TrimSpace(req.Agama) == "" {
		errs["agama"] = []string{"Agama wajib diisi"}
	}
	if strings.TrimSpace(req.AlamatAsal) == "" {
		errs["alamat_asal"] = []string{"Alamat asal wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var existing database.AnakKos
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Data anak kos tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	updates := map[string]interface{}{
		"nama_lengkap":  req.NamaLengkap,
		"jenis_kelamin": req.JenisKelamin,
		"agama":         req.Agama,
		"pekerjaan":     req.Pekerjaan,
		"nomor_hp":      req.NomorHP,
		"alamat_asal":   req.AlamatAsal,
		"status":        status,
	}

	result := h.DB.WithContext(r.Context()).Model(&database.AnakKos{}).
		Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		notFound(w, "Data anak kos tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Ubah Anak Kos",
		"Memperbarui data anak kos "+req.NamaLengkap, existing, req)
	response.Success(w, http.StatusOK, "Data anak kos berhasil diperbarui", nil)
}

func (h *Handler) DeleteAnakKos(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var existing database.AnakKos
	err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Data anak kos tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.AnakKos{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", "NOW()")
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		notFound(w, "Data anak kos tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Hapus Anak Kos",
		"Menghapus (soft) anak kos "+existing.NamaLengkap, nil, map[string]string{"id": id})
	response.Success(w, http.StatusOK, "Data anak kos berhasil dihapus", nil)
}
