package masterdata

import (
	"encoding/json"
	"errors"
	"fmt"
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

func validNIK(nik string) bool {
	if len(nik) != 16 {
		return false
	}
	for _, c := range nik {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("tanggal wajib diisi")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, errors.New("format tanggal harus YYYY-MM-DD")
	}
	if t.After(time.Now()) {
		return time.Time{}, errors.New("tanggal tidak boleh di masa depan")
	}
	return t, nil
}

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

// collectNIKErrors menandai NIK yang duplikat di dalam form atau sudah terdaftar di database.
func (h *Handler) collectNIKErrors(r *http.Request, req *CreateKKRequest, errs map[string][]string) {
	seen := map[string]bool{}
	check := func(nik, key string) {
		nik = strings.TrimSpace(nik)
		if nik == "" || !validNIK(nik) {
			return
		}
		if seen[nik] {
			errs[key] = append(errs[key], "NIK duplikat dengan entri lain pada form ini")
			return
		}
		seen[nik] = true
		var exists bool
		h.DB.WithContext(r.Context()).Raw(
			"SELECT EXISTS(SELECT 1 FROM anggota_keluarga WHERE nik = ? AND deleted_at IS NULL)", nik).Scan(&exists)
		if exists {
			errs[key] = append(errs[key], "NIK telah terdaftar")
		}
	}
	if req.Kepala != nil {
		check(req.Kepala.NIK, "nik")
	}
	for i := range req.Anggota {
		check(req.Anggota[i].NIK, fmt.Sprintf("anggota_%d_nik", i))
	}
}

func notFound(w http.ResponseWriter, msg string) {
	response.Error(w, http.StatusNotFound, msg)
}

func (h *Handler) ListKK(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	rtFilter := r.URL.Query().Get("rt")
	rwFilter := r.URL.Query().Get("rw")

	type kkRow struct {
		database.KartuKeluarga
		NamaKepala    *string `json:"nama_kepala_keluarga"`
		JumlahAnggota int64   `json:"jumlah_anggota"`
	}

	query := h.DB.WithContext(r.Context()).
		Table("kartu_keluarga kk").
		Select("kk.*, a.nama_lengkap as nama_kepala, (SELECT COUNT(*) FROM anggota_keluarga ag WHERE ag.kartu_keluarga_id = kk.id AND ag.deleted_at IS NULL) as jumlah_anggota").
		Joins("LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id AND a.deleted_at IS NULL").
		Where("kk.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(kk.nomor_kk ILIKE ? OR a.nama_lengkap ILIKE ? OR kk.alamat ILIKE ?)", like, like, like)
	}
	if rtFilter != "" {
		query = query.Where("kk.rt = ?", rtFilter)
	}
	if rwFilter != "" {
		query = query.Where("kk.rw = ?", rwFilter)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("kartu_keluarga kk").
		Joins("LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id AND a.deleted_at IS NULL").
		Where("kk.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(kk.nomor_kk ILIKE ? OR a.nama_lengkap ILIKE ? OR kk.alamat ILIKE ?)", like, like, like)
	}
	if rtFilter != "" {
		countQuery = countQuery.Where("kk.rt = ?", rtFilter)
	}
	if rwFilter != "" {
		countQuery = countQuery.Where("kk.rw = ?", rwFilter)
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("kk.created_at DESC")
	case "-created_at":
		query = query.Order("kk.created_at ASC")
	case "jumlah_anggota":
		query = query.Order("jumlah_anggota DESC")
	case "-jumlah_anggota":
		query = query.Order("jumlah_anggota ASC")
	default:
		query = query.Order("kk.created_at DESC")
	}

	list := make([]kkRow, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Kartu Keluarga", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetKK(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	type kkDetail struct {
		database.KartuKeluarga
		NamaKepala         *string `json:"nama_kepala_keluarga"`
		NIKKepala          *string `json:"nik_kepala_keluarga"`
		TempatLahirKepala  *string `json:"tempat_lahir_kepala"`
		TanggalLahirKepala *string `json:"tanggal_lahir_kepala"`
		JumlahAnggota      int64   `json:"jumlah_anggota"`
	}

	var kk kkDetail
	err := h.DB.WithContext(r.Context()).
		Table("kartu_keluarga kk").
		Select("kk.*, a.nama_lengkap AS nama_kepala, a.nik AS nik_kepala, a.tempat_lahir AS tempat_lahir_kepala, a.tanggal_lahir AS tanggal_lahir_kepala, (SELECT COUNT(*) FROM anggota_keluarga ag WHERE ag.kartu_keluarga_id = kk.id AND ag.deleted_at IS NULL) as jumlah_anggota").
		Joins("LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id AND a.deleted_at IS NULL").
		Where("kk.id = ? AND kk.deleted_at IS NULL", id).
		Scan(&kk).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(w, "Kartu Keluarga tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if kk.ID == "" {
		notFound(w, "Kartu Keluarga tidak ditemukan")
		return
	}

	var anggotaList []Anggota
	h.DB.WithContext(r.Context()).
		Table("anggota_keluarga").
		Where("kartu_keluarga_id = ? AND deleted_at IS NULL", id).
		Order("created_at ASC").
		Scan(&anggotaList)

	for i := range anggotaList {
		anggotaList[i].KartuKeluargaID = id
	}

	detail := map[string]interface{}{
		"kartu_keluarga": kk,
		"anggota":        anggotaList,
	}
	response.Success(w, http.StatusOK, "Detail Kartu Keluarga", detail)
}

func (h *Handler) CreateKK(w http.ResponseWriter, r *http.Request) {
	var req CreateKKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.NomorKK = strings.TrimSpace(req.NomorKK)
	if !validNIK(req.NomorKK) {
		errs["nomor_kk"] = []string{"Nomor KK harus 16 digit angka"}
	}
	if strings.TrimSpace(req.Alamat) == "" {
		errs["alamat"] = []string{"Alamat wajib diisi"}
	}
	if strings.TrimSpace(req.RT) == "" {
		errs["rt"] = []string{"RT wajib diisi"}
	}
	if strings.TrimSpace(req.RW) == "" {
		errs["rw"] = []string{"RW wajib diisi"}
	}
	if strings.TrimSpace(req.Desa) == "" {
		errs["desa"] = []string{"Desa wajib diisi"}
	}
	if strings.TrimSpace(req.Kecamatan) == "" {
		errs["kecamatan"] = []string{"Kecamatan wajib diisi"}
	}
	if strings.TrimSpace(req.Kabupaten) == "" {
		errs["kabupaten"] = []string{"Kabupaten wajib diisi"}
	}
	if strings.TrimSpace(req.Provinsi) == "" {
		errs["provinsi"] = []string{"Provinsi wajib diisi"}
	}
	if req.Kepala != nil {
		for k, v := range validateAnggota(*req.Kepala, true) {
			errs[k] = v
		}
	}
	for i, a := range req.Anggota {
		if strings.TrimSpace(a.HubunganKeluarga) == "Kepala Keluarga" {
			errs[fmt.Sprintf("anggota_%d_hubungan_keluarga", i)] = []string{"Kepala keluarga diisi pada bagian Kepala Keluarga"}
			continue
		}
		for k, v := range validateAnggota(a, false) {
			errs[fmt.Sprintf("anggota_%d_%s", i, k)] = v
		}
	}
	if len(errs) == 0 {
		h.collectNIKErrors(r, &req, errs)
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var kkID string
	err := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		kk := database.KartuKeluarga{
			NomorKK:   req.NomorKK,
			NomorHP:   req.NomorHP,
			Alamat:    req.Alamat,
			RT:        req.RT,
			RW:        req.RW,
			Desa:      req.Desa,
			Kecamatan: req.Kecamatan,
			Kabupaten: req.Kabupaten,
			Provinsi:  req.Provinsi,
		}
		if err := tx.Create(&kk).Error; err != nil {
			return err
		}
		kkID = kk.ID

		if req.Kepala != nil {
			memberID, err := insertAnggota(tx, kkID, *req.Kepala, "Kepala Keluarga")
			if err != nil {
				return err
			}
			tx.Model(&database.KartuKeluarga{}).Where("id = ?", kkID).Updates(map[string]interface{}{
				"kepala_keluarga_id": memberID,
				"updated_at":         time.Now(),
			})
		}

		for _, a := range req.Anggota {
			if _, err := insertAnggota(tx, kkID, a, strings.TrimSpace(a.HubunganKeluarga)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"nomor_kk": {"Nomor KK telah terdaftar"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Tambah KK",
		"Menambahkan KK nomor "+req.NomorKK, nil, req)
	response.Created(w, "Kartu Keluarga berhasil ditambahkan", map[string]string{"id": kkID})
}

func (h *Handler) UpdateKK(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateKKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	if strings.TrimSpace(req.Alamat) == "" {
		errs["alamat"] = []string{"Alamat wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.KartuKeluarga{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"nomor_hp":   req.NomorHP,
			"alamat":     req.Alamat,
			"rt":         req.RT,
			"rw":         req.RW,
			"desa":       req.Desa,
			"kecamatan":  req.Kecamatan,
			"kabupaten":  req.Kabupaten,
			"provinsi":   req.Provinsi,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		notFound(w, "Kartu Keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Ubah KK",
		"Memperbarui data KK", nil, req)
	response.Success(w, http.StatusOK, "Kartu Keluarga berhasil diperbarui", nil)
}

func (h *Handler) DeleteKK(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var jumlah int64
	h.DB.WithContext(r.Context()).Model(&database.AnggotaKeluarga{}).
		Where("kartu_keluarga_id = ? AND deleted_at IS NULL", id).Count(&jumlah)
	if jumlah > 0 {
		response.Error(w, http.StatusConflict, "KK tidak dapat dihapus karena masih memiliki anggota keluarga")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.KartuKeluarga{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		notFound(w, "Kartu Keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Master Data", "Hapus KK",
		"Menghapus (soft) Kartu Keluarga", nil, map[string]string{"id": id})
	response.Success(w, http.StatusOK, "Kartu Keluarga berhasil dihapus", nil)
}
