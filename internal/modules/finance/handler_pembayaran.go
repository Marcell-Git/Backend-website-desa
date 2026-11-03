package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

const (
	jenisKas   = "kas"
	jenisIuran = "iuran"
)

var namaBulan = [12]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

var errSudahBayar = errors.New("sudah bayar")

func labelPeriode(t time.Time) string {
	return fmt.Sprintf("%s %d", namaBulan[t.Month()-1], t.Year())
}

// parsePeriode menerima format YYYY-MM dan mengembalikan tanggal 1 bulan tersebut.
func parsePeriode(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("periode wajib diisi")
	}
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return time.Time{}, errors.New("format periode harus YYYY-MM")
	}
	return t, nil
}

// iuranBerlaku memeriksa apakah jenis iuran ditagihkan pada periode tertentu.
func iuranBerlaku(ji database.JenisIuran, periode time.Time) bool {
	if ji.Nominal <= 0 || ji.BulanMulai == nil || periode.Before(*ji.BulanMulai) {
		return false
	}
	if ji.TipePeriode == tipePeriode && ji.BulanSelesai != nil && periode.After(*ji.BulanSelesai) {
		return false
	}
	return true
}

func (h *Handler) nominalKas(ctx context.Context) (database.PengaturanKas, error) {
	var p database.PengaturanKas
	err := h.DB.WithContext(ctx).Order("created_at ASC").FirstOrCreate(&p).Error
	return p, err
}

type statusPembayaranRow struct {
	KartuKeluargaID    string     `json:"kartu_keluarga_id"`
	NomorKK            string     `json:"nomor_kk"`
	NamaKepalaKeluarga *string    `json:"nama_kepala_keluarga"`
	Alamat             string     `json:"alamat"`
	PembayaranID       *string    `json:"pembayaran_id"`
	TanggalBayar       *time.Time `json:"tanggal_bayar"`
	Nominal            *float64   `json:"nominal"`
	NominalTagihan     *float64   `json:"nominal_tagihan"`
	Lunas              bool       `json:"lunas"`
}

type ringkasanPembayaran struct {
	TotalKK        int64   `json:"total_kk"`
	SudahBayar     int64   `json:"sudah_bayar"`
	BelumBayar     int64   `json:"belum_bayar"`
	TotalTerkumpul float64 `json:"total_terkumpul"`
	TotalKelebihan float64 `json:"total_kelebihan"`
	NominalTagihan float64 `json:"nominal_tagihan"`
}

// listStatusPembayaran menampilkan seluruh KK aktif beserta status pembayarannya
// untuk satu jenis tagihan pada satu bulan.
func (h *Handler) listStatusPembayaran(w http.ResponseWriter, r *http.Request, jenis string, jenisIuranID *string, periode time.Time, nominal float64, title string) {
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	status := r.URL.Query().Get("status")

	joinCond := "LEFT JOIN pembayaran_bulanan p ON p.kartu_keluarga_id = kk.id AND p.jenis = ? AND p.periode = ? AND p.deleted_at IS NULL"
	joinArgs := []interface{}{jenis, periode}
	if jenisIuranID != nil {
		joinCond += " AND p.jenis_iuran_id = ?"
		joinArgs = append(joinArgs, *jenisIuranID)
	}

	base := func() *gorm.DB {
		return h.DB.WithContext(r.Context()).
			Table("kartu_keluarga kk").
			Joins("LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id AND a.deleted_at IS NULL").
			Joins(joinCond, joinArgs...).
			Where("kk.deleted_at IS NULL")
	}

	ringkasan := ringkasanPembayaran{NominalTagihan: nominal}
	var agg struct {
		TotalKK        int64
		SudahBayar     int64
		TotalTerkumpul float64
		TotalKelebihan float64
	}
	if err := base().
		Select("COUNT(*) AS total_kk, COUNT(p.id) AS sudah_bayar, COALESCE(SUM(p.nominal), 0) AS total_terkumpul, COALESCE(SUM(GREATEST(p.nominal - p.nominal_tagihan, 0)), 0) AS total_kelebihan").
		Scan(&agg).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	ringkasan.TotalKK = agg.TotalKK
	ringkasan.SudahBayar = agg.SudahBayar
	ringkasan.BelumBayar = agg.TotalKK - agg.SudahBayar
	ringkasan.TotalTerkumpul = agg.TotalTerkumpul
	ringkasan.TotalKelebihan = agg.TotalKelebihan

	filter := func(q *gorm.DB) *gorm.DB {
		if search != "" {
			like := "%" + search + "%"
			q = q.Where("(a.nama_lengkap ILIKE ? OR kk.nomor_kk ILIKE ? OR kk.alamat ILIKE ?)", like, like, like)
		}
		switch status {
		case "lunas":
			q = q.Where("p.id IS NOT NULL")
		case "belum":
			q = q.Where("p.id IS NULL")
		}
		return q
	}

	list := make([]statusPembayaranRow, 0)
	if err := filter(base()).
		Select(`kk.id AS kartu_keluarga_id, kk.nomor_kk, a.nama_lengkap AS nama_kepala_keluarga, kk.alamat,
			p.id AS pembayaran_id, p.tanggal_bayar, p.nominal, p.nominal_tagihan, (p.id IS NOT NULL) AS lunas`).
		Order("(p.id IS NOT NULL) ASC, a.nama_lengkap ASC NULLS LAST, kk.nomor_kk ASC").
		Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.Success(w, http.StatusOK, title, map[string]interface{}{
		"data":      list,
		"ringkasan": ringkasan,
	})
}

type konfirmasiPembayaranRequest struct {
	KartuKeluargaID string `json:"kartu_keluarga_id"`
	JenisIuranID    string `json:"jenis_iuran_id"`
	Periode         string `json:"periode"`
	TanggalBayar    string `json:"tanggal_bayar"`
	// Nominal yang dibayar; kosong berarti sama dengan tagihan. Boleh lebih dari tagihan.
	Nominal *float64 `json:"nominal"`
}

func decodeKonfirmasi(w http.ResponseWriter, r *http.Request, butuhJenisIuran bool) (konfirmasiPembayaranRequest, time.Time, time.Time, bool) {
	var req konfirmasiPembayaranRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return req, time.Time{}, time.Time{}, false
	}

	errs := map[string][]string{}
	if strings.TrimSpace(req.KartuKeluargaID) == "" {
		errs["kartu_keluarga_id"] = []string{"Kartu keluarga wajib dipilih"}
	}
	if butuhJenisIuran && strings.TrimSpace(req.JenisIuranID) == "" {
		errs["jenis_iuran_id"] = []string{"Jenis iuran wajib dipilih"}
	}
	periode, err := parsePeriode(req.Periode)
	if err != nil {
		errs["periode"] = []string{err.Error()}
	}
	tanggal, err := parseDate(req.TanggalBayar)
	if err != nil {
		errs["tanggal_bayar"] = []string{err.Error()}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return req, time.Time{}, time.Time{}, false
	}
	return req, periode, tanggal, true
}

// namaKepalaKK mengembalikan nama kepala keluarga, atau nomor KK bila kepala belum diatur.
// nominalDibayar mengembalikan nominal yang dibayar warga. Pembayaran tidak boleh
// kurang dari tagihan, tetapi boleh lebih (kelebihan tetap dicatat sebagai pemasukan).
func nominalDibayar(w http.ResponseWriter, req konfirmasiPembayaranRequest, tagihan float64) (float64, bool) {
	if req.Nominal == nil {
		return tagihan, true
	}
	if *req.Nominal < tagihan {
		response.ValidationError(w, map[string][]string{"nominal": {"Nominal dibayar tidak boleh kurang dari tagihan " + formatRupiah(tagihan)}})
		return 0, false
	}
	return *req.Nominal, true
}

// keteranganBayar menambahkan catatan kelebihan bayar bila ada.
func keteranganBayar(dasar string, dibayar, tagihan float64) string {
	if dibayar > tagihan {
		return fmt.Sprintf("%s (tagihan %s, lebih %s)", dasar, formatRupiah(tagihan), formatRupiah(dibayar-tagihan))
	}
	return dasar
}

func formatRupiah(n float64) string {
	s := fmt.Sprintf("%.0f", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "Rp" + b.String()
}

func (h *Handler) namaKepalaKK(ctx context.Context, kkID string) (string, bool) {
	var row struct {
		NomorKK string
		Nama    *string
	}
	err := h.DB.WithContext(ctx).
		Table("kartu_keluarga kk").
		Select("kk.nomor_kk, a.nama_lengkap AS nama").
		Joins("LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id AND a.deleted_at IS NULL").
		Where("kk.id = ? AND kk.deleted_at IS NULL", kkID).
		Scan(&row).Error
	if err != nil || row.NomorKK == "" {
		return "", false
	}
	if row.Nama != nil && *row.Nama != "" {
		return *row.Nama, true
	}
	return "KK " + row.NomorKK, true
}

func (h *Handler) ListPembayaranKas(w http.ResponseWriter, r *http.Request) {
	periode, err := parsePeriode(r.URL.Query().Get("periode"))
	if err != nil {
		response.ValidationError(w, map[string][]string{"periode": {err.Error()}})
		return
	}
	pengaturan, err := h.nominalKas(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.listStatusPembayaran(w, r, jenisKas, nil, periode, pengaturan.NominalBulanan, "Status Pembayaran Kas")
}

func (h *Handler) KonfirmasiPembayaranKas(w http.ResponseWriter, r *http.Request) {
	req, periode, tanggal, ok := decodeKonfirmasi(w, r, false)
	if !ok {
		return
	}

	pengaturan, err := h.nominalKas(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if pengaturan.NominalBulanan <= 0 {
		response.Error(w, http.StatusUnprocessableEntity, "Atur nominal kas bulanan terlebih dahulu")
		return
	}
	dibayar, ok := nominalDibayar(w, req, pengaturan.NominalBulanan)
	if !ok {
		return
	}
	nama, ok := h.namaKepalaKK(r.Context(), req.KartuKeluargaID)
	if !ok {
		response.ValidationError(w, map[string][]string{"kartu_keluarga_id": {"Kartu keluarga tidak ditemukan"}})
		return
	}

	uid := *userIDPtr(r)
	var pembayaran database.PembayaranBulanan
	err = h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		keterangan := keteranganBayar("Konfirmasi pembayaran kas bulanan", dibayar, pengaturan.NominalBulanan)
		rekap := database.RekapKas{
			Tanggal:    tanggal,
			SumberKas:  fmt.Sprintf("Kas Bulanan %s - %s", labelPeriode(periode), nama),
			Nominal:    dibayar,
			Keterangan: &keterangan,
			DibuatOleh: uid,
		}
		if err := tx.Create(&rekap).Error; err != nil {
			return err
		}
		pembayaran = database.PembayaranBulanan{
			KartuKeluargaID: req.KartuKeluargaID,
			Jenis:           jenisKas,
			Periode:         periode,
			Nominal:         dibayar,
			NominalTagihan:  pengaturan.NominalBulanan,
			TanggalBayar:    tanggal,
			RekapKasID:      &rekap.ID,
			DibuatOleh:      uid,
		}
		if err := tx.Create(&pembayaran).Error; err != nil {
			if helper.IsDuplicateErrorGORM(err) {
				return errSudahBayar
			}
			return err
		}
		return nil
	})
	if errors.Is(err, errSudahBayar) {
		response.Error(w, http.StatusConflict, "KK ini sudah membayar kas untuk "+labelPeriode(periode))
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Kas", "Konfirmasi Pembayaran Kas",
		fmt.Sprintf("Konfirmasi kas %s: %s", labelPeriode(periode), nama), nil, pembayaran)
	response.Created(w, "Pembayaran kas berhasil dikonfirmasi", map[string]string{"id": pembayaran.ID})
}

func (h *Handler) ListPembayaranIuran(w http.ResponseWriter, r *http.Request) {
	errs := map[string][]string{}
	periode, err := parsePeriode(r.URL.Query().Get("periode"))
	if err != nil {
		errs["periode"] = []string{err.Error()}
	}
	jenisIuranID := r.URL.Query().Get("jenis_iuran_id")
	if jenisIuranID == "" {
		errs["jenis_iuran_id"] = []string{"Jenis iuran wajib dipilih"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var ji database.JenisIuran
	if err := h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", jenisIuranID).First(&ji).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Jenis iuran tidak ditemukan")
		return
	}
	if !iuranBerlaku(ji, periode) {
		response.Error(w, http.StatusUnprocessableEntity, fmt.Sprintf("%s tidak ditagihkan pada %s", ji.Nama, labelPeriode(periode)))
		return
	}
	h.listStatusPembayaran(w, r, jenisIuran, &ji.ID, periode, ji.Nominal, "Status Pembayaran Iuran")
}

func (h *Handler) KonfirmasiPembayaranIuran(w http.ResponseWriter, r *http.Request) {
	req, periode, tanggal, ok := decodeKonfirmasi(w, r, true)
	if !ok {
		return
	}

	var ji database.JenisIuran
	if err := h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", req.JenisIuranID).First(&ji).Error; err != nil {
		response.ValidationError(w, map[string][]string{"jenis_iuran_id": {"Jenis iuran tidak ditemukan"}})
		return
	}
	if !iuranBerlaku(ji, periode) {
		response.Error(w, http.StatusUnprocessableEntity, fmt.Sprintf("%s tidak ditagihkan pada %s", ji.Nama, labelPeriode(periode)))
		return
	}
	dibayar, ok := nominalDibayar(w, req, ji.Nominal)
	if !ok {
		return
	}
	nama, ok := h.namaKepalaKK(r.Context(), req.KartuKeluargaID)
	if !ok {
		response.ValidationError(w, map[string][]string{"kartu_keluarga_id": {"Kartu keluarga tidak ditemukan"}})
		return
	}

	uid := *userIDPtr(r)
	var pembayaran database.PembayaranBulanan
	err := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		keterangan := keteranganBayar(fmt.Sprintf("Pembayaran %s - %s", labelPeriode(periode), nama), dibayar, ji.Nominal)
		rekap := database.RekapIuran{
			Tanggal:      tanggal,
			JenisIuranID: ji.ID,
			Nominal:      dibayar,
			Keterangan:   &keterangan,
			DibuatOleh:   uid,
		}
		if err := tx.Create(&rekap).Error; err != nil {
			return err
		}
		pembayaran = database.PembayaranBulanan{
			KartuKeluargaID: req.KartuKeluargaID,
			Jenis:           jenisIuran,
			JenisIuranID:    &ji.ID,
			Periode:         periode,
			Nominal:         dibayar,
			NominalTagihan:  ji.Nominal,
			TanggalBayar:    tanggal,
			RekapIuranID:    &rekap.ID,
			DibuatOleh:      uid,
		}
		if err := tx.Create(&pembayaran).Error; err != nil {
			if helper.IsDuplicateErrorGORM(err) {
				return errSudahBayar
			}
			return err
		}
		return nil
	})
	if errors.Is(err, errSudahBayar) {
		response.Error(w, http.StatusConflict, fmt.Sprintf("KK ini sudah membayar %s untuk %s", ji.Nama, labelPeriode(periode)))
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Iuran", "Konfirmasi Pembayaran Iuran",
		fmt.Sprintf("Konfirmasi %s %s: %s", ji.Nama, labelPeriode(periode), nama), nil, pembayaran)
	response.Created(w, "Pembayaran iuran berhasil dikonfirmasi", map[string]string{"id": pembayaran.ID})
}

// batalPembayaran menghapus (soft) konfirmasi pembayaran beserta entri rekap yang dibuatnya.
func (h *Handler) batalPembayaran(w http.ResponseWriter, r *http.Request, jenis, modul string) {
	id := r.PathValue("id")

	var p database.PembayaranBulanan
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND jenis = ? AND deleted_at IS NULL", id, jenis).
		First(&p).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pembayaran tidak ditemukan")
		return
	}

	now := time.Now()
	err := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.PembayaranBulanan{}).Where("id = ?", p.ID).Update("deleted_at", now).Error; err != nil {
			return err
		}
		if p.RekapKasID != nil {
			if err := tx.Model(&database.RekapKas{}).Where("id = ?", *p.RekapKasID).Update("deleted_at", now).Error; err != nil {
				return err
			}
		}
		if p.RekapIuranID != nil {
			if err := tx.Model(&database.RekapIuran{}).Where("id = ?", *p.RekapIuranID).Update("deleted_at", now).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), modul, "Batal Konfirmasi Pembayaran",
		"Membatalkan konfirmasi pembayaran "+labelPeriode(p.Periode), p, nil)
	response.Success(w, http.StatusOK, "Konfirmasi pembayaran berhasil dibatalkan", nil)
}

func (h *Handler) BatalPembayaranKas(w http.ResponseWriter, r *http.Request) {
	h.batalPembayaran(w, r, jenisKas, "Rekap Kas")
}

func (h *Handler) BatalPembayaranIuran(w http.ResponseWriter, r *http.Request) {
	h.batalPembayaran(w, r, jenisIuran, "Rekap Iuran")
}

func (h *Handler) GetPengaturanKas(w http.ResponseWriter, r *http.Request) {
	p, err := h.nominalKas(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	response.Success(w, http.StatusOK, "Pengaturan Kas", p)
}

func (h *Handler) UpdatePengaturanKas(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NominalBulanan float64 `json:"nominal_bulanan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	if req.NominalBulanan <= 0 {
		response.ValidationError(w, map[string][]string{"nominal_bulanan": {"Nominal harus lebih besar dari 0"}})
		return
	}

	before, err := h.nominalKas(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if err := h.DB.WithContext(r.Context()).Model(&database.PengaturanKas{}).
		Where("id = ?", before.ID).
		Updates(map[string]interface{}{"nominal_bulanan": req.NominalBulanan, "updated_at": time.Now()}).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Rekap Kas", "Ubah Nominal Kas",
		"Memperbarui nominal kas bulanan", before, req)
	response.Success(w, http.StatusOK, "Nominal kas bulanan berhasil diperbarui", nil)
}

const pesanDariPembayaran = "Transaksi ini berasal dari konfirmasi pembayaran bulanan; batalkan melalui tab Pembayaran Bulanan"

// dariPembayaran memeriksa apakah entri rekap dibuat dari konfirmasi pembayaran bulanan.
func (h *Handler) dariPembayaran(r *http.Request, kolom, rekapID string) bool {
	var count int64
	h.DB.WithContext(r.Context()).Model(&database.PembayaranBulanan{}).
		Where(kolom+" = ? AND deleted_at IS NULL", rekapID).Count(&count)
	return count > 0
}
