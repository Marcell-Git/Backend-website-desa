package inventaris

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	DB      *gorm.DB
	BaseDir string
}

func NewHandler(db *gorm.DB, baseDir string) *Handler {
	return &Handler{DB: db, BaseDir: baseDir}
}

var validKondisi = map[string]bool{
	"baik":            true,
	"rusak-ringan":    true,
	"rusak-berat":     true,
	"perlu-perbaikan": true,
}

var allowedFotoExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
}

const maxFotoSize = 5 << 20 // 5 MB

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	kategori := helper.CleanSearch(r.URL.Query().Get("kategori"))
	kondisi := helper.CleanSearch(r.URL.Query().Get("kondisi"))

	query := h.baseQuery(r).Where("i.deleted_at IS NULL")
	count := h.DB.WithContext(r.Context()).Model(&database.Inventaris{}).Where("deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(i.nama_barang ILIKE ? OR i.lokasi_penyimpanan ILIKE ? OR i.penanggung_jawab ILIKE ? OR i.keterangan ILIKE ?)", like, like, like, like)
		count = count.Where("(nama_barang ILIKE ? OR lokasi_penyimpanan ILIKE ? OR penanggung_jawab ILIKE ? OR keterangan ILIKE ?)", like, like, like, like)
	}
	if kategori != "" {
		query = query.Where("i.kategori ILIKE ?", kategori)
		count = count.Where("kategori ILIKE ?", kategori)
	}
	if kondisi != "" {
		query = query.Where("i.kondisi_barang = ?", kondisi)
		count = count.Where("kondisi_barang = ?", kondisi)
	}
	var total int64
	count.Count(&total)
	items := make([]Item, 0)
	if err := query.Order("i.created_at DESC").Offset(pg.Offset()).Limit(pg.Limit).Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data inventaris")
		return
	}
	response.SuccessPaginated(w, "Daftar Inventaris", items, response.Pagination{Page: pg.Page, Limit: pg.Limit, TotalData: total, TotalPage: pg.TotalPage(total)})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var item Item
	err := h.baseQuery(r).Where("i.id = ? AND i.deleted_at IS NULL", r.PathValue("id")).Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Barang tidak ditemukan")
		return
	}
	response.Success(w, http.StatusOK, "Detail Inventaris", item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxFotoSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	req, errs := h.readForm(r)
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}
	model := database.Inventaris{
		NamaBarang:        req.NamaBarang,
		Kategori:          req.Kategori,
		WarnaLabel:        req.WarnaLabel,
		JumlahBarang:      req.JumlahBarang,
		KondisiBarang:     req.KondisiBarang,
		LokasiPenyimpanan: req.LokasiPenyimpanan,
		PenanggungJawab:   req.PenanggungJawab,
		PeriodePerolehan:  req.PeriodePerolehan,
		Keterangan:        req.Keterangan,
		DibuatOleh:        *userIDPtr(r),
	}
	if err := h.saveFoto(r, &model.Foto); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		h.removeFoto(model.Foto)
		response.Error(w, http.StatusInternalServerError, "Gagal menyimpan data inventaris")
		return
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Inventaris", "Tambah", "Menambahkan: "+req.NamaBarang, nil, model)
	response.Created(w, "Barang berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxFotoSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	req, errs := h.readForm(r)
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}
	var before database.Inventaris
	result := h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", r.PathValue("id")).First(&before)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		response.Error(w, http.StatusNotFound, "Barang tidak ditemukan")
		return
	}
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data inventaris")
		return
	}

	updates := map[string]interface{}{
		"nama_barang":        req.NamaBarang,
		"kategori":           req.Kategori,
		"warna_label":        req.WarnaLabel,
		"jumlah_barang":      req.JumlahBarang,
		"kondisi_barang":     req.KondisiBarang,
		"lokasi_penyimpanan": req.LokasiPenyimpanan,
		"penanggung_jawab":   req.PenanggungJawab,
		"periode_perolehan":  req.PeriodePerolehan,
		"keterangan":         req.Keterangan,
		"updated_at":         time.Now(),
	}

	hapusFoto := r.FormValue("hapus_foto") == "1"
	if hapusFoto {
		updates["foto"] = nil
	} else {
		var newFoto *string
		if err := h.saveFoto(r, &newFoto); err != nil {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if newFoto != nil {
			updates["foto"] = *newFoto
		}
	}

	if err := h.DB.WithContext(r.Context()).Model(&before).Updates(updates).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memperbarui data inventaris")
		return
	}

	if hapusFoto && before.Foto != nil {
		h.removeFoto(before.Foto)
	} else if newFoto := updates["foto"]; newFoto != nil && before.Foto != nil && newFoto != *before.Foto {
		h.removeFoto(before.Foto)
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Inventaris", "Ubah", "Memperbarui: "+req.NamaBarang, before, updates)
	response.Success(w, http.StatusOK, "Barang berhasil diperbarui", nil)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var before database.Inventaris
	result := h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", r.PathValue("id")).First(&before)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		response.Error(w, http.StatusNotFound, "Barang tidak ditemukan")
		return
	}
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data inventaris")
		return
	}
	if err := h.DB.WithContext(r.Context()).Delete(&before).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menghapus data inventaris")
		return
	}
	if before.Foto != nil {
		h.removeFoto(before.Foto)
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Inventaris", "Hapus", "Menghapus: "+before.NamaBarang, before, nil)
	response.Success(w, http.StatusOK, "Barang berhasil dihapus", nil)
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	items := make([]Item, 0)
	if err := h.baseQuery(r).Where("i.deleted_at IS NULL").Order("i.nama_barang ASC").Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data inventaris")
		return
	}
	response.Success(w, http.StatusOK, "Daftar Inventaris", items)
}

func (h *Handler) baseQuery(r *http.Request) *gorm.DB {
	return h.DB.WithContext(r.Context()).Table("inventaris i").Select("i.*, u.username as nama_user").Joins("LEFT JOIN users u ON u.id = i.dibuat_oleh AND u.deleted_at IS NULL")
}

type itemRequest struct {
	NamaBarang        string
	Kategori          string
	WarnaLabel        string
	JumlahBarang      int
	KondisiBarang     string
	LokasiPenyimpanan string
	PenanggungJawab   string
	PeriodePerolehan  string
	Keterangan        string
}

func (h *Handler) readForm(r *http.Request) (itemRequest, map[string][]string) {
	errs := map[string][]string{}
	req := itemRequest{
		NamaBarang:        strings.TrimSpace(r.FormValue("nama_barang")),
		Kategori:          strings.TrimSpace(r.FormValue("kategori")),
		WarnaLabel:        strings.ToUpper(strings.TrimSpace(r.FormValue("warna_label"))),
		KondisiBarang:     strings.ToLower(strings.TrimSpace(r.FormValue("kondisi_barang"))),
		LokasiPenyimpanan: strings.TrimSpace(r.FormValue("lokasi_penyimpanan")),
		PenanggungJawab:   strings.TrimSpace(r.FormValue("penanggung_jawab")),
		PeriodePerolehan:  strings.TrimSpace(r.FormValue("periode_perolehan")),
		Keterangan:        strings.TrimSpace(r.FormValue("keterangan")),
	}
	if req.WarnaLabel == "" {
		req.WarnaLabel = "#047857"
	}
	if req.KondisiBarang == "" {
		req.KondisiBarang = "baik"
	}
	if req.NamaBarang == "" {
		errs["nama_barang"] = []string{"Nama barang wajib diisi"}
	}
	if req.Kategori == "" {
		errs["kategori"] = []string{"Kategori wajib diisi"}
	}
	if !isHexColor(req.WarnaLabel) {
		errs["warna_label"] = []string{"Warna label harus berupa kode hex, misalnya #047857"}
	}
	jumlahStr := r.FormValue("jumlah_barang")
	jumlah, err := strconv.Atoi(jumlahStr)
	if jumlahStr == "" || err != nil {
		errs["jumlah_barang"] = []string{"Jumlah barang wajib diisi angka"}
	} else if jumlah < 1 {
		errs["jumlah_barang"] = []string{"Jumlah barang minimal 1"}
	} else {
		req.JumlahBarang = jumlah
	}
	if !validKondisi[req.KondisiBarang] {
		errs["kondisi_barang"] = []string{"Kondisi barang tidak valid"}
	}
	if req.PeriodePerolehan != "" && !isYearMonth(req.PeriodePerolehan) {
		errs["periode_perolehan"] = []string{"Format periode harus YYYY-MM"}
	}
	return req, errs
}

func isYearMonth(value string) bool {
	if len(value) != 7 || value[4] != '-' {
		return false
	}
	if _, err := time.Parse("2006-01", value); err != nil {
		return false
	}
	return true
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

func (h *Handler) saveFoto(r *http.Request, dst **string) error {
	file, handler, err := r.FormFile("foto")
	if err != nil {
		return nil
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if !allowedFotoExts[ext] {
		return fmt.Errorf("format foto tidak didukung. Format yang diizinkan: JPG, JPEG, PNG, WEBP")
	}
	if handler.Size > maxFotoSize {
		return fmt.Errorf("ukuran foto maksimal 5 MB")
	}

	uploadDir := filepath.Join(h.BaseDir, "uploads", "inventaris")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("gagal menyimpan foto")
	}

	filename, err := randomFotoFilename(ext)
	if err != nil {
		return fmt.Errorf("gagal menyimpan foto")
	}
	dstPath := filepath.Join(uploadDir, filename)

	dest, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("gagal menyimpan foto")
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(dstPath)
		return fmt.Errorf("gagal menyimpan foto")
	}

	relPath := "uploads/inventaris/" + filename
	*dst = &relPath
	return nil
}

func (h *Handler) removeFoto(relPath *string) {
	if relPath == nil || *relPath == "" {
		return
	}
	abs := filepath.Join(h.BaseDir, (*relPath))
	os.Remove(abs)
}

func randomFotoFilename(ext string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return time.Now().Format("20060102150405") + "_" + hex.EncodeToString(buf) + ext, nil
}
