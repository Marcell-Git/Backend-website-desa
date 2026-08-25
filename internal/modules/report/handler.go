package report

import (
	"net/http"

	"gorm.io/gorm"

	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) applyDateFilter(query *gorm.DB, f ReportFilter) *gorm.DB {
	if f.Bulan != "" {
		query = query.Where("EXTRACT(MONTH FROM tanggal) = ?", f.Bulan)
	}
	if f.Tahun != "" {
		query = query.Where("EXTRACT(YEAR FROM tanggal) = ?", f.Tahun)
	}
	if f.TanggalMulai != "" {
		query = query.Where("tanggal >= ?", f.TanggalMulai)
	}
	if f.TanggalAkhir != "" {
		query = query.Where("tanggal <= ?", f.TanggalAkhir)
	}
	return query
}

func (h *Handler) LaporanKas(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []KasRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_kas rk").
		Select("rk.tanggal, rk.sumber_kas, rk.nominal, rk.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN users u ON u.id = rk.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rk.deleted_at IS NULL")
	query = h.applyDateFilter(query, f)
	query = query.Order("rk.tanggal ASC")

	if err := query.Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat laporan kas")
		return
	}

	response.Success(w, http.StatusOK, "Laporan kas berhasil dimuat", items)
}

func (h *Handler) LaporanIuran(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []IuranRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_iuran ri").
		Select("ri.tanggal, ji.nama as jenis_iuran, ri.nominal, ri.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN jenis_iuran ji ON ji.id = ri.jenis_iuran_id AND ji.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = ri.dibuat_oleh AND u.deleted_at IS NULL").
		Where("ri.deleted_at IS NULL")
	query = h.applyDateFilter(query, f)
	if f.JenisIuranID != "" {
		query = query.Where("ri.jenis_iuran_id = ?", f.JenisIuranID)
	}
	query = query.Order("ri.tanggal ASC")

	if err := query.Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat laporan iuran")
		return
	}

	response.Success(w, http.StatusOK, "Laporan iuran berhasil dimuat", items)
}

func (h *Handler) LaporanPengeluaran(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []PengeluaranRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran rp").
		Select("rp.tanggal, kp.nama as kategori_pengeluaran, rp.nominal, rp.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN kategori_pengeluaran kp ON kp.id = rp.kategori_pengeluaran_id AND kp.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = rp.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rp.deleted_at IS NULL")
	query = h.applyDateFilter(query, f)
	if f.KategoriPengeluaranID != "" {
		query = query.Where("rp.kategori_pengeluaran_id = ?", f.KategoriPengeluaranID)
	}
	query = query.Order("rp.tanggal ASC")

	if err := query.Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat laporan pengeluaran")
		return
	}

	response.Success(w, http.StatusOK, "Laporan pengeluaran berhasil dimuat", items)
}

func (h *Handler) LaporanKeuangan(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var totalKas, totalIuran, totalPengeluaran float64

	kasQuery := h.DB.WithContext(r.Context()).
		Table("rekap_kas").Where("deleted_at IS NULL")
	kasQuery = h.applyDateFilter(kasQuery, f)
	kasQuery.Select("COALESCE(SUM(nominal), 0)").Scan(&totalKas)

	iuranQuery := h.DB.WithContext(r.Context()).
		Table("rekap_iuran").Where("deleted_at IS NULL")
	iuranQuery = h.applyDateFilter(iuranQuery, f)
	iuranQuery.Select("COALESCE(SUM(nominal), 0)").Scan(&totalIuran)

	pengeluaranQuery := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran").Where("deleted_at IS NULL")
	pengeluaranQuery = h.applyDateFilter(pengeluaranQuery, f)
	pengeluaranQuery.Select("COALESCE(SUM(nominal), 0)").Scan(&totalPengeluaran)

	summary := KeuanganSummary{
		TotalKas:         totalKas,
		TotalIuran:       totalIuran,
		TotalPengeluaran: totalPengeluaran,
		SaldoAkhir:       (totalKas + totalIuran) - totalPengeluaran,
		Bulan:            f.Bulan,
		Tahun:            f.Tahun,
	}

	var kasRows []KasRow
	kasDetail := h.DB.WithContext(r.Context()).
		Table("rekap_kas rk").
		Select("rk.tanggal, rk.sumber_kas, rk.nominal, rk.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN users u ON u.id = rk.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rk.deleted_at IS NULL")
	kasDetail = h.applyDateFilter(kasDetail, f)
	kasDetail.Order("rk.tanggal ASC").Scan(&kasRows)

	var iuranRows []IuranRow
	iuranDetail := h.DB.WithContext(r.Context()).
		Table("rekap_iuran ri").
		Select("ri.tanggal, ji.nama as jenis_iuran, ri.nominal, ri.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN jenis_iuran ji ON ji.id = ri.jenis_iuran_id AND ji.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = ri.dibuat_oleh AND u.deleted_at IS NULL").
		Where("ri.deleted_at IS NULL")
	iuranDetail = h.applyDateFilter(iuranDetail, f)
	iuranDetail.Order("ri.tanggal ASC").Scan(&iuranRows)

	var pengeluaranRows []PengeluaranRow
	pengeluaranDetail := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran rp").
		Select("rp.tanggal, kp.nama as kategori_pengeluaran, rp.nominal, rp.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN kategori_pengeluaran kp ON kp.id = rp.kategori_pengeluaran_id AND kp.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = rp.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rp.deleted_at IS NULL")
	pengeluaranDetail = h.applyDateFilter(pengeluaranDetail, f)
	pengeluaranDetail.Order("rp.tanggal ASC").Scan(&pengeluaranRows)

	report := KeuanganReport{
		Summary:     summary,
		Kas:         kasRows,
		Iuran:       iuranRows,
		Pengeluaran: pengeluaranRows,
	}

	response.Success(w, http.StatusOK, "Laporan keuangan berhasil dimuat", report)
}

func parseFilter(r *http.Request) ReportFilter {
	return ReportFilter{
		Bulan:                  r.URL.Query().Get("bulan"),
		Tahun:                  r.URL.Query().Get("tahun"),
		TanggalMulai:           r.URL.Query().Get("tanggal_mulai"),
		TanggalAkhir:           r.URL.Query().Get("tanggal_akhir"),
		JenisIuranID:           r.URL.Query().Get("jenis_iuran_id"),
		KategoriPengeluaranID:  r.URL.Query().Get("kategori_pengeluaran_id"),
	}
}
