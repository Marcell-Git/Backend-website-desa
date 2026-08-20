package masterdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	Pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{Pool: pool}
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

func notFound(w http.ResponseWriter, msg string) {
	response.Error(w, http.StatusNotFound, msg)
}

// ---------------- Kartu Keluarga ----------------

func (h *Handler) ListKK(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	rt := r.URL.Query().Get("rt")
	rw := r.URL.Query().Get("rw")

	where := []string{"kk.deleted_at IS NULL"}
	args := []interface{}{}

	if search != "" {
		args = append(args, "%"+search+"%")
		where = append(where, fmt.Sprintf(`(kk.nomor_kk ILIKE $%d OR a.nama_lengkap ILIKE $%d OR kk.alamat ILIKE $%d)`, len(args), len(args), len(args)))
	}
	if rt != "" {
		args = append(args, rt)
		where = append(where, fmt.Sprintf("kk.rt = $%d", len(args)))
	}
	if rw != "" {
		args = append(args, rw)
		where = append(where, fmt.Sprintf("kk.rw = $%d", len(args)))
	}

	whereSQL := strings.Join(where, " AND ")

	var total int64
	err := h.Pool.QueryRow(r.Context(), fmt.Sprintf(`
		SELECT COUNT(*)
		FROM kartu_keluarga kk
		LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id
		WHERE %s`, whereSQL), args...).Scan(&total)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	orderBy := "a.nama_lengkap ASC"
	switch pg.Sort {
	case "created_at":
		orderBy = "kk.created_at ASC"
	case "-created_at":
		orderBy = "kk.created_at DESC"
	case "jumlah_anggota":
		orderBy = "jumlah_anggota ASC"
	case "-jumlah_anggota":
		orderBy = "jumlah_anggota DESC"
	}

	args = append(args, pg.Limit, pg.Offset())
	rows, err := h.Pool.Query(r.Context(), fmt.Sprintf(`
		SELECT kk.id, kk.nomor_kk, kk.nomor_hp, kk.alamat, kk.rt, kk.rw, kk.desa,
		       kk.kecamatan, kk.kabupaten, kk.provinsi, kk.kepala_keluarga_id,
		       a.nama_lengkap,
		       (SELECT COUNT(*) FROM anggota_keluarga ag
		         WHERE ag.kartu_keluarga_id = kk.id AND ag.deleted_at IS NULL) AS jumlah_anggota,
		       kk.created_at, kk.updated_at
		FROM kartu_keluarga kk
		LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, whereSQL, orderBy, len(args)-1, len(args)), args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()

	list := []KartuKeluarga{}
	for rows.Next() {
		var kk KartuKeluarga
		if err := rows.Scan(&kk.ID, &kk.NomorKK, &kk.NomorHP, &kk.Alamat, &kk.RT, &kk.RW, &kk.Desa,
			&kk.Kecamatan, &kk.Kabupaten, &kk.Provinsi, &kk.KepalaKeluargaID,
			&kk.NamaKepala, &kk.JumlahAnggota, &kk.CreatedAt, &kk.UpdatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		list = append(list, kk)
	}
	if rows.Err() != nil {
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
	var kk KartuKeluarga
	var tanggalLahir *time.Time
	err := h.Pool.QueryRow(r.Context(), `
		SELECT kk.id, kk.nomor_kk, kk.nomor_hp, kk.alamat, kk.rt, kk.rw, kk.desa,
		       kk.kecamatan, kk.kabupaten, kk.provinsi, kk.kepala_keluarga_id,
		       a.nama_lengkap, a.nik, a.tempat_lahir, a.tanggal_lahir,
		       (SELECT COUNT(*) FROM anggota_keluarga ag
		         WHERE ag.kartu_keluarga_id = kk.id AND ag.deleted_at IS NULL) AS jumlah_anggota,
		       kk.created_at, kk.updated_at
		FROM kartu_keluarga kk
		LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id
		WHERE kk.id = $1 AND kk.deleted_at IS NULL`, id,
	).Scan(&kk.ID, &kk.NomorKK, &kk.NomorHP, &kk.Alamat, &kk.RT, &kk.RW, &kk.Desa,
		&kk.Kecamatan, &kk.Kabupaten, &kk.Provinsi, &kk.KepalaKeluargaID,
		&kk.NamaKepala, &kk.NIKKepala, &kk.TempatLahirKepala, &tanggalLahir,
		&kk.JumlahAnggota, &kk.CreatedAt, &kk.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notFound(w, "Kartu Keluarga tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if tanggalLahir != nil {
		kk.TanggalLahirKepala = tanggalLahir.Format("2006-01-02")
	}

	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, nik, nama_lengkap, tempat_lahir, to_char(tanggal_lahir, 'YYYY-MM-DD'),
		       jenis_kelamin, agama, COALESCE(pekerjaan, ''), COALESCE(nomor_hp, ''),
		       hubungan_keluarga, status, created_at, updated_at
		FROM anggota_keluarga
		WHERE kartu_keluarga_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC`, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()

	anggota := []Anggota{}
	for rows.Next() {
		var a Anggota
		var pekerjaan, nomorHP string
		if err := rows.Scan(&a.ID, &a.NIK, &a.NamaLengkap, &a.TempatLahir, &a.TanggalLahir,
			&a.JenisKelamin, &a.Agama, &pekerjaan, &nomorHP, &a.HubunganKeluarga,
			&a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if pekerjaan != "" {
			a.Pekerjaan = &pekerjaan
		}
		if nomorHP != "" {
			a.NomorHP = &nomorHP
		}
		a.KartuKeluargaID = id
		anggota = append(anggota, a)
	}

	detail := map[string]interface{}{
		"kartu_keluarga": kk,
		"anggota":        anggota,
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
		if e := validateAnggota(*req.Kepala, true); len(e) > 0 {
			errs["kepala_keluarga"] = e
		}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer tx.Rollback(r.Context())

	var kkID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO kartu_keluarga (nomor_kk, nomor_hp, alamat, rt, rw, desa, kecamatan, kabupaten, provinsi)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		req.NomorKK, req.NomorHP, req.Alamat, req.RT, req.RW, req.Desa,
		req.Kecamatan, req.Kabupaten, req.Provinsi,
	).Scan(&kkID)
	if err != nil {
		if helper.IsDuplicateError(err) {
			response.ValidationError(w, map[string][]string{"nomor_kk": {"Nomor KK telah terdaftar"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if req.Kepala != nil {
		memberID, err := insertAnggota(r.Context(), tx, kkID, *req.Kepala, "Kepala Keluarga")
		if err != nil {
			if helper.IsDuplicateError(err) {
				response.ValidationError(w, map[string][]string{"nik": {"NIK telah terdaftar"}})
				return
			}
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		_, err = tx.Exec(r.Context(),
			`UPDATE kartu_keluarga SET kepala_keluarga_id = $1, updated_at = NOW() WHERE id = $2`,
			memberID, kkID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Tambah KK",
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

	ct, err := h.Pool.Exec(r.Context(), `
		UPDATE kartu_keluarga SET nomor_hp = $1, alamat = $2, rt = $3, rw = $4,
		       desa = $5, kecamatan = $6, kabupaten = $7, provinsi = $8, updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL`,
		req.NomorHP, req.Alamat, req.RT, req.RW, req.Desa, req.Kecamatan, req.Kabupaten, req.Provinsi, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if ct.RowsAffected() == 0 {
		notFound(w, "Kartu Keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Ubah KK",
		"Memperbarui data KK", nil, req)
	response.Success(w, http.StatusOK, "Kartu Keluarga berhasil diperbarui", nil)
}

func (h *Handler) DeleteKK(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var jumlah int
	err := h.Pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM anggota_keluarga WHERE kartu_keluarga_id = $1 AND deleted_at IS NULL`, id).Scan(&jumlah)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if jumlah > 0 {
		response.Error(w, http.StatusConflict, "KK tidak dapat dihapus karena masih memiliki anggota keluarga")
		return
	}

	ct, err := h.Pool.Exec(r.Context(),
		`UPDATE kartu_keluarga SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if ct.RowsAffected() == 0 {
		notFound(w, "Kartu Keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Hapus KK",
		"Menghapus (soft) Kartu Keluarga", nil, map[string]string{"id": id})
	response.Success(w, http.StatusOK, "Kartu Keluarga berhasil dihapus", nil)
}
