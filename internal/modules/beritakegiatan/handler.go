package beritakegiatan

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

type Handler struct{ DB *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{DB: db} }

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

func validate(req *UpsertRequest) (time.Time, map[string][]string) {
	errs := map[string][]string{}
	req.Kategori = strings.TrimSpace(req.Kategori)
	req.Judul = strings.TrimSpace(req.Judul)
	req.Lokasi = strings.TrimSpace(req.Lokasi)
	req.Keterangan = strings.TrimSpace(req.Keterangan)
	req.WarnaLabel = strings.ToUpper(strings.TrimSpace(req.WarnaLabel))
	if req.WarnaLabel == "" {
		req.WarnaLabel = "#047857"
	}
	if req.Kategori == "" {
		errs["kategori"] = []string{"Kategori wajib diisi"}
	}
	if !isHexColor(req.WarnaLabel) {
		errs["warna_label"] = []string{"Warna label harus berupa kode hex, misalnya #047857"}
	}
	if req.Judul == "" {
		errs["judul"] = []string{"Judul kegiatan wajib diisi"}
	}
	if req.Lokasi == "" {
		errs["lokasi"] = []string{"Lokasi wajib diisi"}
	}
	tanggal, err := time.Parse("2006-01-02", req.Tanggal)
	if req.Tanggal == "" {
		errs["tanggal"] = []string{"Tanggal wajib diisi"}
	} else if err != nil {
		errs["tanggal"] = []string{"Format tanggal harus YYYY-MM-DD"}
	}
	if _, err := time.Parse("15:04", req.Jam); req.Jam == "" {
		errs["jam"] = []string{"Jam wajib diisi"}
	} else if err != nil {
		errs["jam"] = []string{"Format jam harus HH:MM"}
	}
	return tanggal, errs
}

func isHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, char := range value[1:] {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	kategori := helper.CleanSearch(r.URL.Query().Get("kategori"))
	query := h.baseQuery(r).Where("b.deleted_at IS NULL")
	count := h.DB.WithContext(r.Context()).Model(&database.BeritaKegiatan{}).Where("deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(b.judul ILIKE ? OR b.lokasi ILIKE ? OR b.keterangan ILIKE ?)", like, like, like)
		count = count.Where("(judul ILIKE ? OR lokasi ILIKE ? OR keterangan ILIKE ?)", like, like, like)
	}
	if kategori != "" {
		query = query.Where("b.kategori ILIKE ?", kategori)
		count = count.Where("kategori ILIKE ?", kategori)
	}
	var total int64
	count.Count(&total)
	items := make([]Item, 0)
	if err := query.Order("b.tanggal DESC, b.jam DESC, b.created_at DESC").Offset(pg.Offset()).Limit(pg.Limit).Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat berita dan kegiatan")
		return
	}
	response.SuccessPaginated(w, "Daftar Berita dan Kegiatan", items, response.Pagination{Page: pg.Page, Limit: pg.Limit, TotalData: total, TotalPage: pg.TotalPage(total)})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var item Item
	err := h.baseQuery(r).Where("b.id = ? AND b.deleted_at IS NULL", r.PathValue("id")).Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Berita atau kegiatan tidak ditemukan")
		return
	}
	response.Success(w, http.StatusOK, "Detail Berita dan Kegiatan", item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req UpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	tanggal, errs := validate(&req)
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}
	model := database.BeritaKegiatan{Kategori: req.Kategori, WarnaLabel: req.WarnaLabel, Judul: req.Judul, Tanggal: tanggal, Jam: req.Jam, Lokasi: req.Lokasi, Keterangan: req.Keterangan, DibuatOleh: *userIDPtr(r)}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menyimpan berita dan kegiatan")
		return
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Berita dan Kegiatan", "Tambah", "Menambahkan: "+req.Judul, nil, model)
	response.Created(w, "Berita dan kegiatan berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	tanggal, errs := validate(&req)
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}
	var before database.BeritaKegiatan
	result := h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", r.PathValue("id")).First(&before)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		response.Error(w, http.StatusNotFound, "Berita atau kegiatan tidak ditemukan")
		return
	}
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat berita dan kegiatan")
		return
	}
	updates := map[string]interface{}{"kategori": req.Kategori, "warna_label": req.WarnaLabel, "judul": req.Judul, "tanggal": tanggal, "jam": req.Jam, "lokasi": req.Lokasi, "keterangan": req.Keterangan, "updated_at": time.Now()}
	if err := h.DB.WithContext(r.Context()).Model(&before).Updates(updates).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memperbarui berita dan kegiatan")
		return
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Berita dan Kegiatan", "Ubah", "Memperbarui: "+req.Judul, before, updates)
	response.Success(w, http.StatusOK, "Berita dan kegiatan berhasil diperbarui", nil)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var before database.BeritaKegiatan
	result := h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", r.PathValue("id")).First(&before)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		response.Error(w, http.StatusNotFound, "Berita atau kegiatan tidak ditemukan")
		return
	}
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat berita dan kegiatan")
		return
	}
	if err := h.DB.WithContext(r.Context()).Delete(&before).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menghapus berita dan kegiatan")
		return
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Berita dan Kegiatan", "Hapus", "Menghapus: "+before.Judul, before, nil)
	response.Success(w, http.StatusOK, "Berita dan kegiatan berhasil dihapus", nil)
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	items := make([]Item, 0)
	if err := h.baseQuery(r).Where("b.deleted_at IS NULL").Order("b.tanggal DESC, b.jam DESC, b.created_at DESC").Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat berita dan kegiatan")
		return
	}
	response.Success(w, http.StatusOK, "Daftar Berita dan Kegiatan", items)
}

func (h *Handler) baseQuery(r *http.Request) *gorm.DB {
	return h.DB.WithContext(r.Context()).Table("berita_kegiatan b").Select("b.*, u.username as nama_user").Joins("LEFT JOIN users u ON u.id = b.dibuat_oleh AND u.deleted_at IS NULL")
}
