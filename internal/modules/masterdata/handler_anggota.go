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

func validateAnggota(req CreateAnggotaRequest, isKepala bool) []string {
	errs := []string{}
	if !validNIK(req.NIK) {
		errs = append(errs, "NIK harus 16 digit angka")
	}
	if strings.TrimSpace(req.NamaLengkap) == "" {
		errs = append(errs, "Nama lengkap wajib diisi")
	}
	if strings.TrimSpace(req.TempatLahir) == "" {
		errs = append(errs, "Tempat lahir wajib diisi")
	}
	if _, err := parseDate(req.TanggalLahir); err != nil {
		errs = append(errs, err.Error())
	}
	if req.JenisKelamin != "L" && req.JenisKelamin != "P" {
		errs = append(errs, "Jenis kelamin harus L atau P")
	}
	if strings.TrimSpace(req.Agama) == "" {
		errs = append(errs, "Agama wajib diisi")
	}
	if !isKepala && strings.TrimSpace(req.HubunganKeluarga) == "" {
		errs = append(errs, "Hubungan keluarga wajib diisi")
	}
	return errs
}

func insertAnggota(tx *gorm.DB, kkID string, req CreateAnggotaRequest, hubungan string) (string, error) {
	status := true
	if req.Status != nil {
		status = *req.Status
	}
	a := database.AnggotaKeluarga{
		KartuKeluargaID:  kkID,
		NIK:              req.NIK,
		NamaLengkap:      req.NamaLengkap,
		TempatLahir:      req.TempatLahir,
		TanggalLahir:     req.TanggalLahir,
		JenisKelamin:     req.JenisKelamin,
		Agama:            req.Agama,
		Pekerjaan:        req.Pekerjaan,
		NomorHP:          req.NomorHP,
		HubunganKeluarga: hubungan,
		Status:           status,
	}
	if err := tx.Create(&a).Error; err != nil {
		return "", err
	}
	return a.ID, nil
}

func (h *Handler) loadAnggota(id string) (*Anggota, error) {
	var a Anggota
	err := h.DB.Where("id = ? AND deleted_at IS NULL", id).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (h *Handler) hasKepala(tx *gorm.DB, kkID string) (bool, error) {
	var count int64
	err := tx.Model(&database.AnggotaKeluarga{}).
		Where("kartu_keluarga_id = ? AND hubungan_keluarga = ? AND deleted_at IS NULL", kkID, "Kepala Keluarga").
		Count(&count).Error
	return count > 0, err
}

func (h *Handler) demoteKepala(tx *gorm.DB, kkID string) error {
	return tx.Model(&database.AnggotaKeluarga{}).
		Where("kartu_keluarga_id = ? AND hubungan_keluarga = ? AND deleted_at IS NULL", kkID, "Kepala Keluarga").
		Update("hubungan_keluarga", "Anggota").Error
}

func (h *Handler) ListAnggota(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	kkID := r.URL.Query().Get("kartu_keluarga_id")
	jk := r.URL.Query().Get("jenis_kelamin")
	agama := r.URL.Query().Get("agama")
	status := r.URL.Query().Get("status")

	query := h.DB.WithContext(r.Context()).Model(&database.AnggotaKeluarga{}).
		Where("deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(nik ILIKE ? OR nama_lengkap ILIKE ? OR nomor_hp ILIKE ? OR pekerjaan ILIKE ?)", like, like, like, like)
	}
	if kkID != "" {
		query = query.Where("kartu_keluarga_id = ?", kkID)
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
		query = query.Order("created_at ASC")
	case "-created_at":
		query = query.Order("created_at DESC")
	case "nik":
		query = query.Order("nik ASC")
	default:
		query = query.Order("nama_lengkap ASC")
	}

	var list []Anggota
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Find(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Anggota Keluarga", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetAnggota(w http.ResponseWriter, r *http.Request) {
	a, err := h.loadAnggota(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Anggota keluarga tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	response.Success(w, http.StatusOK, "Detail Anggota Keluarga", a)
}

func (h *Handler) CreateAnggota(w http.ResponseWriter, r *http.Request) {
	var req CreateAnggotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	req.KartuKeluargaID = strings.TrimSpace(req.KartuKeluargaID)
	if req.KartuKeluargaID == "" {
		response.ValidationError(w, map[string][]string{"kartu_keluarga_id": {"KK wajib dipilih"}})
		return
	}

	isKepala := strings.TrimSpace(req.HubunganKeluarga) == "Kepala Keluarga"
	if errs := validateAnggota(req, isKepala); len(errs) > 0 {
		response.ValidationError(w, map[string][]string{"anggota": errs})
		return
	}

	var kkExists bool
	h.DB.WithContext(r.Context()).Raw(
		"SELECT EXISTS(SELECT 1 FROM kartu_keluarga WHERE id = ? AND deleted_at IS NULL)",
		req.KartuKeluargaID).Scan(&kkExists)
	if !kkExists {
		response.ValidationError(w, map[string][]string{"kartu_keluarga_id": {"KK tidak ditemukan"}})
		return
	}

	var memberID string
	err := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if isKepala {
			exists, err := h.hasKepala(tx, req.KartuKeluargaID)
			if err != nil {
				return err
			}
			if exists {
				return errors.New("KEPALA_EXISTS")
			}
		}

		id, err := insertAnggota(tx, req.KartuKeluargaID, req, req.HubunganKeluarga)
		if err != nil {
			return err
		}
		memberID = id

		if isKepala {
			tx.Model(&database.KartuKeluarga{}).Where("id = ?", req.KartuKeluargaID).Updates(map[string]interface{}{
				"kepala_keluarga_id": memberID,
				"updated_at":         "NOW()",
			})
		}
		return nil
	})
	if err != nil {
		if err.Error() == "KEPALA_EXISTS" {
			response.ValidationError(w, map[string][]string{"hubungan_keluarga": {"Kepala keluarga hanya boleh satu pada setiap KK"}})
			return
		}
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"nik": {"NIK telah terdaftar"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Tambah Anggota",
		"Menambahkan anggota "+req.NamaLengkap, nil, req)
	response.Created(w, "Anggota keluarga berhasil ditambahkan", map[string]string{"id": memberID})
}

func (h *Handler) UpdateAnggota(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateAnggotaRequest
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
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	a, err := h.loadAnggota(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Anggota keluarga tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	newIsKepala := strings.TrimSpace(req.HubunganKeluarga) == "Kepala Keluarga"
	oldIsKepala := a.HubunganKeluarga == "Kepala Keluarga"

	if oldIsKepala && !newIsKepala && req.HubunganKeluarga != "" {
		response.Error(w, http.StatusConflict, "Kepala keluarga harus diganti terlebih dahulu sebelum diubah menjadi anggota biasa")
		return
	}
	if req.HubunganKeluarga == "" {
		req.HubunganKeluarga = a.HubunganKeluarga
	}

	status := a.Status
	if req.Status != nil {
		status = *req.Status
	}

	err = h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if newIsKepala && !oldIsKepala {
			exists, err := h.hasKepala(tx, a.KartuKeluargaID)
			if err != nil {
				return err
			}
			if exists {
				return errors.New("KEPALA_EXISTS")
			}
		}

		tx.Model(&database.AnggotaKeluarga{}).Where("id = ?", id).Updates(map[string]interface{}{
			"nama_lengkap":      req.NamaLengkap,
			"nomor_hp":          req.NomorHP,
			"agama":             req.Agama,
			"pekerjaan":         req.Pekerjaan,
			"hubungan_keluarga": req.HubunganKeluarga,
			"status":            status,
		})

		if newIsKepala && !oldIsKepala {
			h.demoteKepala(tx, a.KartuKeluargaID)
			tx.Model(&database.KartuKeluarga{}).Where("id = ?", a.KartuKeluargaID).Updates(map[string]interface{}{
				"kepala_keluarga_id": id,
				"updated_at":        "NOW()",
			})
		}
		return nil
	})
	if err != nil {
		if err.Error() == "KEPALA_EXISTS" {
			response.ValidationError(w, map[string][]string{"hubungan_keluarga": {"Kepala keluarga hanya boleh satu pada setiap KK"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Ubah Anggota",
		"Memperbarui anggota "+req.NamaLengkap, a, req)
	response.Success(w, http.StatusOK, "Anggota keluarga berhasil diperbarui", nil)
}

func (h *Handler) DeleteAnggota(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.loadAnggota(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Anggota keluarga tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if a.HubunganKeluarga == "Kepala Keluarga" {
		response.Error(w, http.StatusConflict, "Kepala keluarga tidak dapat dihapus. Tentukan pengganti kepala keluarga terlebih dahulu")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.AnggotaKeluarga{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", "NOW()")
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		notFound(w, "Anggota keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Hapus Anggota",
		"Menghapus (soft) anggota "+a.NamaLengkap, nil, map[string]string{"id": id})
	response.Success(w, http.StatusOK, "Anggota keluarga berhasil dihapus", nil)
}

func (h *Handler) SetKepala(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.loadAnggota(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Anggota keluarga tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if a.HubunganKeluarga == "Kepala Keluarga" {
		response.Success(w, http.StatusOK, "Anggota sudah menjadi kepala keluarga", nil)
		return
	}

	err = h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		h.demoteKepala(tx, a.KartuKeluargaID)
		tx.Model(&database.AnggotaKeluarga{}).Where("id = ?", id).Updates(map[string]interface{}{
			"hubungan_keluarga": "Kepala Keluarga",
		})
		tx.Model(&database.KartuKeluarga{}).Where("id = ?", a.KartuKeluargaID).Updates(map[string]interface{}{
			"kepala_keluarga_id": id,
		})
		return nil
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Ubah Kepala Keluarga",
		"Mengganti kepala keluarga menjadi "+a.NamaLengkap, nil, map[string]string{"kepala_keluarga_id": id})
	response.Success(w, http.StatusOK, "Kepala keluarga berhasil diperbarui", nil)
}

func (h *Handler) MoveAnggota(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req MoveAnggotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	a, err := h.loadAnggota(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Anggota keluarga tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if a.HubunganKeluarga == "Kepala Keluarga" {
		response.Error(w, http.StatusConflict, "Kepala keluarga tidak dapat dipindahkan. Tentukan pengganti kepala keluarga terlebih dahulu")
		return
	}

	var targetExists bool
	h.DB.WithContext(r.Context()).Raw(
		"SELECT EXISTS(SELECT 1 FROM kartu_keluarga WHERE id = ? AND deleted_at IS NULL)",
		req.KartuKeluargaID).Scan(&targetExists)
	if !targetExists {
		response.ValidationError(w, map[string][]string{"kartu_keluarga_id": {"KK tujuan tidak ditemukan"}})
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.AnggotaKeluarga{}).
		Where("id = ?", id).
		Update("kartu_keluarga_id", req.KartuKeluargaID)
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		notFound(w, "Anggota keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Pindah Anggota",
		"Memindahkan anggota "+a.NamaLengkap+" ke KK lain", a, req)
	response.Success(w, http.StatusOK, "Anggota berhasil dipindahkan ke KK lain", nil)
}
