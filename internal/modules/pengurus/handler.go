package pengurus

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

var allowedFotoExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
}

const maxFotoSize = 5 << 20 // 5 MB per file
const maxBulkSize = 60 << 20
const fotoFieldPrefix = "foto_"

type bulkRow struct {
	Key         string `json:"key"`
	ID          string `json:"id"`
	NamaLengkap string `json:"nama_lengkap"`
	Jabatan     string `json:"jabatan"`
	Urutan      int    `json:"urutan"`
	HapusFoto   bool   `json:"hapus_foto"`
}

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))

	query := h.DB.WithContext(r.Context()).
		Table("pengurus p").
		Where("p.deleted_at IS NULL")
	count := h.DB.WithContext(r.Context()).Model(&database.Pengurus{}).Where("deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(p.nama_lengkap ILIKE ? OR p.jabatan ILIKE ?)", like, like)
		count = count.Where("(nama_lengkap ILIKE ? OR jabatan ILIKE ?)", like, like)
	}
	var total int64
	count.Count(&total)
	items := make([]Item, 0)
	if err := query.
		Order("p.urutan ASC, p.created_at ASC").
		Offset(pg.Offset()).
		Limit(pg.Limit).
		Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data pengurus")
		return
	}
	response.SuccessPaginated(w, "Daftar Pengurus", items, response.Pagination{Page: pg.Page, Limit: pg.Limit, TotalData: total, TotalPage: pg.TotalPage(total)})
}

func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.all(r)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data pengurus")
		return
	}
	response.Success(w, http.StatusOK, "Daftar Pengurus", items)
}

func (h *Handler) all(r *http.Request) ([]Item, error) {
	items := make([]Item, 0)
	err := h.DB.WithContext(r.Context()).
		Table("pengurus p").
		Where("p.deleted_at IS NULL").
		Order("p.urutan ASC, p.created_at ASC").
		Scan(&items).Error
	return items, err
}

// SaveBulk replaces the entire board in one transaction: every row in the
// payload is created or updated, and any record left out is soft deleted.
// Photos arrive as multipart fields named foto_<row key>.
func (h *Handler) SaveBulk(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxBulkSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	var rows []bulkRow
	if err := json.Unmarshal([]byte(r.FormValue("rows")), &rows); err != nil {
		response.Error(w, http.StatusBadRequest, "Data pengurus tidak valid")
		return
	}
	if errs := validateBulkRows(rows); len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	files := map[string]*multipart.FileHeader{}
	for name, headers := range r.MultipartForm.File {
		if strings.HasPrefix(name, fotoFieldPrefix) && len(headers) > 0 {
			files[strings.TrimPrefix(name, fotoFieldPrefix)] = headers[0]
		}
	}

	var written []string
	saved, err := h.persistBulk(r, rows, files, &written)
	if err != nil {
		for _, path := range written {
			h.removeFoto(&path)
		}
		if ve, ok := err.(*bulkValidationError); ok {
			response.ValidationError(w, ve.errs)
			return
		}
		response.Error(w, http.StatusInternalServerError, "Gagal menyimpan data pengurus")
		return
	}

	response.Success(w, http.StatusOK, "Susunan pengurus berhasil disimpan", saved)
}

type bulkValidationError struct{ errs map[string][]string }

func (e *bulkValidationError) Error() string { return "validasi gagal" }

func validateBulkRows(rows []bulkRow) map[string][]string {
	errs := map[string][]string{}
	seenKey := map[string]bool{}
	seenID := map[string]bool{}

	for i := range rows {
		rows[i].Jabatan = strings.TrimSpace(rows[i].Jabatan)
		rows[i].NamaLengkap = strings.TrimSpace(rows[i].NamaLengkap)
		prefix := fmt.Sprintf("rows.%d", i)

		switch {
		case rows[i].Key == "":
			errs[prefix+".key"] = append(errs[prefix+".key"], "Baris tidak memiliki kunci")
		case seenKey[rows[i].Key]:
			errs[prefix+".key"] = append(errs[prefix+".key"], "Kunci baris harus unik")
		default:
			seenKey[rows[i].Key] = true
		}

		if rows[i].Jabatan == "" {
			errs[prefix+".jabatan"] = append(errs[prefix+".jabatan"], "Jabatan wajib diisi")
		}
		if rows[i].NamaLengkap == "" {
			errs[prefix+".nama_lengkap"] = append(errs[prefix+".nama_lengkap"], "Nama lengkap wajib diisi")
		}
		if rows[i].Urutan < 0 {
			errs[prefix+".urutan"] = append(errs[prefix+".urutan"], "Urutan tidak boleh negatif")
		}
		if rows[i].ID != "" {
			if seenID[rows[i].ID] {
				errs[prefix+".id"] = append(errs[prefix+".id"], "Seseorang hanya bisa punya satu jabatan")
			} else {
				seenID[rows[i].ID] = true
			}
		}
	}
	return errs
}

func (h *Handler) persistBulk(r *http.Request, rows []bulkRow, files map[string]*multipart.FileHeader, written *[]string) ([]Item, error) {
	tx := h.DB.WithContext(r.Context()).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()

	var existing []database.Pengurus
	if err := tx.Where("deleted_at IS NULL").Find(&existing).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]*database.Pengurus, len(existing))
	for i := range existing {
		byID[existing[i].ID] = &existing[i]
	}

	now := time.Now()
	kept := map[string]bool{}

	for i := range rows {
		row := rows[i]
		var target *database.Pengurus
		if row.ID != "" {
			target = byID[row.ID]
			if target == nil {
				return nil, &bulkValidationError{errs: map[string][]string{
					fmt.Sprintf("rows.%d.id", i): {"Jabatan ini sudah tidak ada, muat ulang halaman"},
				}}
			}
			kept[target.ID] = true
		}

		foto := ""
		if target != nil && target.Foto != nil {
			foto = *target.Foto
		}
		if header, ok := files[row.Key]; ok {
			path, err := h.saveFoto(header)
			if err != nil {
				return nil, &bulkValidationError{errs: map[string][]string{
					fmt.Sprintf("rows.%d.foto", i): {err.Error()},
				}}
			}
			*written = append(*written, path)
			foto = path
		} else if row.HapusFoto {
			foto = ""
		}

		updates := map[string]interface{}{
			"nama_lengkap": row.NamaLengkap,
			"jabatan":      row.Jabatan,
			"urutan":       row.Urutan,
			"updated_at":   now,
			"foto":         nil,
		}
		if foto != "" {
			updates["foto"] = foto
		}

		if target == nil {
			record := database.Pengurus{
				NamaLengkap: row.NamaLengkap,
				Jabatan:     row.Jabatan,
				Urutan:      row.Urutan,
			}
			if foto != "" {
				record.Foto = &foto
			}
			if err := tx.Create(&record).Error; err != nil {
				return nil, err
			}
			continue
		}

		if err := tx.Model(&database.Pengurus{}).Where("id = ?", target.ID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	for id := range byID {
		if kept[id] {
			continue
		}
		if err := tx.Delete(&database.Pengurus{}, "id = ? AND deleted_at IS NULL", id).Error; err != nil {
			return nil, err
		}
	}

	before, err := json.Marshal(existing)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	saved, err := h.all(r)
	if err != nil {
		return nil, err
	}

	// Any photo that existed before this request and is no longer referenced
	// belongs to a removed row or to a replaced photo, so unlink it now that
	// the transaction has committed.
	referenced := make(map[string]bool, len(saved))
	for _, item := range saved {
		if item.Foto != nil {
			referenced[*item.Foto] = true
		}
	}
	for i := range existing {
		if existing[i].Foto != nil && !referenced[*existing[i].Foto] {
			h.removeFoto(existing[i].Foto)
		}
	}

	after, err := json.Marshal(saved)
	if err != nil {
		return nil, err
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Pengurus", "Ubah",
		fmt.Sprintf("Memperbarui susunan pengurus (%d jabatan)", len(saved)),
		json.RawMessage(before), json.RawMessage(after))
	return saved, nil
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	items, err := h.all(r)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data pengurus")
		return
	}
	response.Success(w, http.StatusOK, "Daftar Pengurus", items)
}

func (h *Handler) saveFoto(header *multipart.FileHeader) (string, error) {
	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan foto")
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedFotoExts[ext] {
		return "", fmt.Errorf("format foto tidak didukung. Format yang diizinkan: JPG, JPEG, PNG, WEBP")
	}
	if header.Size > maxFotoSize {
		return "", fmt.Errorf("ukuran foto maksimal 5 MB")
	}

	uploadDir := filepath.Join(h.BaseDir, "uploads", "pengurus")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("gagal menyimpan foto")
	}

	filename, err := randomFotoFilename(ext)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan foto")
	}
	dstPath := filepath.Join(uploadDir, filename)

	dest, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan foto")
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(dstPath)
		return "", fmt.Errorf("gagal menyimpan foto")
	}

	return "uploads/pengurus/" + filename, nil
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
