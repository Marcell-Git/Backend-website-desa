package masterdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

type execer interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

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

func insertAnggota(ctx context.Context, ex execer, kkID string, req CreateAnggotaRequest, hubungan string) (string, error) {
	status := true
	if req.Status != nil {
		status = *req.Status
	}
	var id string
	err := ex.QueryRow(ctx, `
		INSERT INTO anggota_keluarga (kartu_keluarga_id, nik, nama_lengkap, tempat_lahir, tanggal_lahir,
		                              jenis_kelamin, agama, pekerjaan, nomor_hp, hubungan_keluarga, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		kkID, req.NIK, req.NamaLengkap, req.TempatLahir, req.TanggalLahir,
		req.JenisKelamin, req.Agama, req.Pekerjaan, req.NomorHP, hubungan, status,
	).Scan(&id)
	return id, err
}

func (h *Handler) loadAnggota(ctx context.Context, id string) (*Anggota, error) {
	var a Anggota
	var pekerjaan, nomorHP string
	err := h.Pool.QueryRow(ctx, `
		SELECT id, kartu_keluarga_id, nik, nama_lengkap, tempat_lahir,
		       to_char(tanggal_lahir, 'YYYY-MM-DD'), jenis_kelamin, agama,
		       COALESCE(pekerjaan, ''), COALESCE(nomor_hp, ''), hubungan_keluarga, status,
		       created_at, updated_at
		FROM anggota_keluarga
		WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&a.ID, &a.KartuKeluargaID, &a.NIK, &a.NamaLengkap, &a.TempatLahir, &a.TanggalLahir,
		&a.JenisKelamin, &a.Agama, &pekerjaan, &nomorHP, &a.HubunganKeluarga,
		&a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if pekerjaan != "" {
		a.Pekerjaan = &pekerjaan
	}
	if nomorHP != "" {
		a.NomorHP = &nomorHP
	}
	return &a, nil
}

func (h *Handler) hasKepala(ctx context.Context, ex execer, kkID string) (bool, error) {
	var exists bool
	err := ex.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM anggota_keluarga
			WHERE kartu_keluarga_id = $1 AND hubungan_keluarga = 'Kepala Keluarga' AND deleted_at IS NULL
		)`, kkID).Scan(&exists)
	return exists, err
}

func (h *Handler) demoteKepala(ctx context.Context, ex execer, kkID string) error {
	_, err := ex.Exec(ctx, `
		UPDATE anggota_keluarga SET hubungan_keluarga = 'Anggota', updated_at = NOW()
		WHERE kartu_keluarga_id = $1 AND hubungan_keluarga = 'Kepala Keluarga' AND deleted_at IS NULL`, kkID)
	return err
}

// ---------------- Anggota ----------------

func (h *Handler) ListAnggota(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	kkID := r.URL.Query().Get("kartu_keluarga_id")
	jk := r.URL.Query().Get("jenis_kelamin")
	agama := r.URL.Query().Get("agama")
	status := r.URL.Query().Get("status")

	where := []string{"a.deleted_at IS NULL"}
	args := []interface{}{}

	if search != "" {
		args = append(args, "%"+search+"%")
		where = append(where, fmt.Sprintf(`(a.nik ILIKE $%d OR a.nama_lengkap ILIKE $%d OR a.nomor_hp ILIKE $%d OR a.pekerjaan ILIKE $%d)`, len(args), len(args), len(args), len(args)))
	}
	if kkID != "" {
		args = append(args, kkID)
		where = append(where, fmt.Sprintf("a.kartu_keluarga_id = $%d", len(args)))
	}
	if jk == "L" || jk == "P" {
		args = append(args, jk)
		where = append(where, fmt.Sprintf("a.jenis_kelamin = $%d", len(args)))
	}
	if agama != "" {
		args = append(args, agama)
		where = append(where, fmt.Sprintf("a.agama = $%d", len(args)))
	}
	if status == "true" || status == "false" {
		args = append(args, status == "true")
		where = append(where, fmt.Sprintf("a.status = $%d", len(args)))
	}

	whereSQL := strings.Join(where, " AND ")

	var total int64
	err := h.Pool.QueryRow(r.Context(), fmt.Sprintf(
		`SELECT COUNT(*) FROM anggota_keluarga a WHERE %s`, whereSQL), args...).Scan(&total)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	orderBy := "a.nama_lengkap ASC"
	switch pg.Sort {
	case "created_at":
		orderBy = "a.created_at ASC"
	case "-created_at":
		orderBy = "a.created_at DESC"
	case "nik":
		orderBy = "a.nik ASC"
	}

	args = append(args, pg.Limit, pg.Offset())
	rows, err := h.Pool.Query(r.Context(), fmt.Sprintf(`
		SELECT a.id, a.kartu_keluarga_id, a.nik, a.nama_lengkap, a.tempat_lahir,
		       to_char(a.tanggal_lahir, 'YYYY-MM-DD'), a.jenis_kelamin, a.agama,
		       COALESCE(a.pekerjaan, ''), COALESCE(a.nomor_hp, ''), a.hubungan_keluarga, a.status,
		       a.created_at, a.updated_at
		FROM anggota_keluarga a
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, whereSQL, orderBy, len(args)-1, len(args)), args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()

	list := []Anggota{}
	for rows.Next() {
		var a Anggota
		var pekerjaan, nomorHP string
		if err := rows.Scan(&a.ID, &a.KartuKeluargaID, &a.NIK, &a.NamaLengkap, &a.TempatLahir, &a.TanggalLahir,
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
		list = append(list, a)
	}
	if rows.Err() != nil {
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
	a, err := h.loadAnggota(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	err := h.Pool.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM kartu_keluarga WHERE id = $1 AND deleted_at IS NULL)`,
		req.KartuKeluargaID).Scan(&kkExists)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !kkExists {
		response.ValidationError(w, map[string][]string{"kartu_keluarga_id": {"KK tidak ditemukan"}})
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer tx.Rollback(r.Context())

	if isKepala {
		exists, err := h.hasKepala(r.Context(), tx, req.KartuKeluargaID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if exists {
			response.ValidationError(w, map[string][]string{"hubungan_keluarga": {"Kepala keluarga hanya boleh satu pada setiap KK"}})
			return
		}
	}

	memberID, err := insertAnggota(r.Context(), tx, req.KartuKeluargaID, req, req.HubunganKeluarga)
	if err != nil {
		if helper.IsDuplicateError(err) {
			response.ValidationError(w, map[string][]string{"nik": {"NIK telah terdaftar"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if isKepala {
		_, err = tx.Exec(r.Context(),
			`UPDATE kartu_keluarga SET kepala_keluarga_id = $1, updated_at = NOW() WHERE id = $2`,
			memberID, req.KartuKeluargaID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Tambah Anggota",
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

	a, err := h.loadAnggota(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer tx.Rollback(r.Context())

	if newIsKepala && !oldIsKepala {
		exists, err := h.hasKepala(r.Context(), tx, a.KartuKeluargaID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if exists {
			response.ValidationError(w, map[string][]string{"hubungan_keluarga": {"Kepala keluarga hanya boleh satu pada setiap KK"}})
			return
		}
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE anggota_keluarga SET nama_lengkap = $1, nomor_hp = $2, agama = $3,
		       pekerjaan = $4, hubungan_keluarga = $5, status = $6, updated_at = NOW()
		WHERE id = $7`,
		req.NamaLengkap, req.NomorHP, req.Agama, req.Pekerjaan, req.HubunganKeluarga, status, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if newIsKepala && !oldIsKepala {
		if err := h.demoteKepala(r.Context(), tx, a.KartuKeluargaID); err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		_, err = tx.Exec(r.Context(),
			`UPDATE kartu_keluarga SET kepala_keluarga_id = $1, updated_at = NOW() WHERE id = $2`,
			id, a.KartuKeluargaID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Ubah Anggota",
		"Memperbarui anggota "+req.NamaLengkap, a, req)
	response.Success(w, http.StatusOK, "Anggota keluarga berhasil diperbarui", nil)
}

func (h *Handler) DeleteAnggota(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.loadAnggota(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

	ct, err := h.Pool.Exec(r.Context(),
		`UPDATE anggota_keluarga SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if ct.RowsAffected() == 0 {
		notFound(w, "Anggota keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Hapus Anggota",
		"Menghapus (soft) anggota "+a.NamaLengkap, nil, map[string]string{"id": id})
	response.Success(w, http.StatusOK, "Anggota keluarga berhasil dihapus", nil)
}

func (h *Handler) SetKepala(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.loadAnggota(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer tx.Rollback(r.Context())

	if err := h.demoteKepala(r.Context(), tx, a.KartuKeluargaID); err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE anggota_keluarga SET hubungan_keluarga = 'Kepala Keluarga', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	_, err = tx.Exec(r.Context(),
		`UPDATE kartu_keluarga SET kepala_keluarga_id = $1, updated_at = NOW() WHERE id = $2`,
		id, a.KartuKeluargaID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Ubah Kepala Keluarga",
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

	a, err := h.loadAnggota(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	err = h.Pool.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM kartu_keluarga WHERE id = $1 AND deleted_at IS NULL)`,
		req.KartuKeluargaID).Scan(&targetExists)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !targetExists {
		response.ValidationError(w, map[string][]string{"kartu_keluarga_id": {"KK tujuan tidak ditemukan"}})
		return
	}

	ct, err := h.Pool.Exec(r.Context(),
		`UPDATE anggota_keluarga SET kartu_keluarga_id = $1, updated_at = NOW() WHERE id = $2`,
		req.KartuKeluargaID, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if ct.RowsAffected() == 0 {
		notFound(w, "Anggota keluarga tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.Pool, userIDPtr(r), "Master Data", "Pindah Anggota",
		"Memindahkan anggota "+a.NamaLengkap+" ke KK lain", a, req)
	response.Success(w, http.StatusOK, "Anggota berhasil dipindahkan ke KK lain", nil)
}
