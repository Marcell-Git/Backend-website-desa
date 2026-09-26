package peraturan

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

const maxPDFSize = 10 << 20 // 10 MB

// pdfMagic adalah penanda awal berkas PDF. Ekstensi .pdf saja tidak cukup
// karena berkas berbahaya bisa disamarkan menjadi .pdf lalu disajikan inline
// dari path publik.
var pdfMagic = []byte("%PDF-")

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))

	query := h.baseQuery(r).Where("p.deleted_at IS NULL")
	count := h.DB.WithContext(r.Context()).Model(&database.Peraturan{}).Where("deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("p.judul ILIKE ?", like)
		count = count.Where("judul ILIKE ?", like)
	}

	var total int64
	count.Count(&total)

	items := make([]Item, 0)
	if err := query.Order("p.created_at DESC").
		Offset(pg.Offset()).Limit(pg.Limit).Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data peraturan")
		return
	}
	response.SuccessPaginated(w, "Daftar Peraturan", items, response.Pagination{
		Page: pg.Page, Limit: pg.Limit, TotalData: total, TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var item Item
	err := h.baseQuery(r).Where("p.id = ? AND p.deleted_at IS NULL", r.PathValue("id")).Scan(&item).Error
	if err != nil || item.ID == "" {
		response.Error(w, http.StatusNotFound, "Peraturan tidak ditemukan")
		return
	}
	response.Success(w, http.StatusOK, "Detail Peraturan", item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxPDFSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	judul := strings.TrimSpace(r.FormValue("judul"))
	if errs := validate(judul); len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var relPath string
	if err := h.savePDF(r, &relPath); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if relPath == "" {
		response.Error(w, http.StatusBadRequest, "File PDF wajib diunggah")
		return
	}

	model := database.Peraturan{
		Judul:      judul,
		FilePDF:    relPath,
		DibuatOleh: *userIDPtr(r),
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		h.removeFile(relPath)
		response.Error(w, http.StatusInternalServerError, "Gagal menyimpan peraturan")
		return
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Peraturan", "Tambah", "Menambahkan: "+judul, nil, model)
	response.Created(w, "Peraturan berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxPDFSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	judul := strings.TrimSpace(r.FormValue("judul"))
	if errs := validate(judul); len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var before database.Peraturan
	result := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", r.PathValue("id")).First(&before)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		response.Error(w, http.StatusNotFound, "Peraturan tidak ditemukan")
		return
	}
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data peraturan")
		return
	}

	oldPath := before.FilePDF
	var newPath string
	if err := h.savePDF(r, &newPath); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	updates := map[string]interface{}{
		"judul":      judul,
		"updated_at": time.Now(),
	}
	if newPath != "" {
		updates["file_pdf"] = newPath
	}

	if err := h.DB.WithContext(r.Context()).Model(&before).Updates(updates).Error; err != nil {
		if newPath != "" {
			h.removeFile(newPath)
		}
		response.Error(w, http.StatusInternalServerError, "Gagal memperbarui peraturan")
		return
	}

	// before sudah merefleksikan nilai baru setelah Updates di bawah, jadi
	// path lama harus diambil sebelum update dilakukan.
	if newPath != "" && oldPath != newPath {
		h.removeFile(oldPath)
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Peraturan", "Ubah", "Memperbarui: "+judul, before, updates)
	response.Success(w, http.StatusOK, "Peraturan berhasil diperbarui", nil)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var before database.Peraturan
	result := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", r.PathValue("id")).First(&before)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		response.Error(w, http.StatusNotFound, "Peraturan tidak ditemukan")
		return
	}
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data peraturan")
		return
	}
	if err := h.DB.WithContext(r.Context()).Delete(&before).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal menghapus peraturan")
		return
	}
	h.removeFile(before.FilePDF)
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Peraturan", "Hapus", "Menghapus: "+before.Judul, before, nil)
	response.Success(w, http.StatusOK, "Peraturan berhasil dihapus", nil)
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	items := make([]PublicItem, 0)
	err := h.DB.WithContext(r.Context()).Model(&database.Peraturan{}).
		Select("id, judul, file_pdf, created_at").
		Where("deleted_at IS NULL").
		Order("created_at DESC").Scan(&items).Error
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data peraturan")
		return
	}
	response.Success(w, http.StatusOK, "Daftar Peraturan", items)
}

func (h *Handler) baseQuery(r *http.Request) *gorm.DB {
	return h.DB.WithContext(r.Context()).Table("peraturan p").
		Select("p.*, u.username as nama_user").
		Joins("LEFT JOIN users u ON u.id = p.dibuat_oleh AND u.deleted_at IS NULL")
}

func validate(judul string) map[string][]string {
	errs := map[string][]string{}
	if judul == "" {
		errs["judul"] = []string{"Judul wajib diisi"}
	} else if len([]rune(judul)) > 200 {
		errs["judul"] = []string{"Judul maksimal 200 karakter"}
	}
	return errs
}

// savePDF menyimpan berkas PDF dan mengisi path relatif pada dst.
// Path relatif tidak diubah bila tidak ada file yang diunggah, sehingga Update
// bisa membedakan "tidak ganti file" dari "ganti file".
func (h *Handler) savePDF(r *http.Request, dst *string) error {
	file, header, err := r.FormFile("file_pdf")
	if err != nil {
		return nil
	}
	defer file.Close()

	if !strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
		return fmt.Errorf("format file harus PDF")
	}
	if header.Size > maxPDFSize {
		return fmt.Errorf("ukuran file maksimal 10 MB")
	}

	magic := make([]byte, len(pdfMagic))
	if _, err := io.ReadFull(file, magic); err != nil {
		return fmt.Errorf("file PDF tidak dapat dibaca")
	}
	if !strings.HasPrefix(string(magic), string(pdfMagic)) {
		return fmt.Errorf("file yang diunggah bukan PDF yang valid")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("file PDF tidak dapat dibaca")
	}

	uploadDir := filepath.Join(h.BaseDir, "uploads", "peraturan")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("gagal menyimpan file")
	}

	filename, err := randomFilename(".pdf")
	if err != nil {
		return fmt.Errorf("gagal menyimpan file")
	}
	dstPath := filepath.Join(uploadDir, filename)

	dest, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("gagal menyimpan file")
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(dstPath)
		return fmt.Errorf("gagal menyimpan file")
	}

	*dst = "uploads/peraturan/" + filename
	return nil
}

func (h *Handler) removeFile(relPath string) {
	if relPath == "" {
		return
	}
	os.Remove(filepath.Join(h.BaseDir, filepath.Clean("/"+relPath)))
}

func randomFilename(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return time.Now().Format("20060102150405") + "_" + hex.EncodeToString(buf) + ext, nil
}
