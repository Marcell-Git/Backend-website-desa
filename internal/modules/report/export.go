package report

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/pkg/response"
)

type ExportHandler struct {
	DB *gorm.DB
}

func NewExportHandler(db *gorm.DB) *ExportHandler {
	return &ExportHandler{DB: db}
}

func (h *ExportHandler) ExportKasPDF(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []KasRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_kas rk").
		Select("rk.tanggal, rk.sumber_kas, rk.nominal, rk.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN users u ON u.id = rk.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rk.deleted_at IS NULL")
	applyDateFilterToQuery(query, f).Order("rk.tanggal ASC").Scan(&items)

	headers := []string{"Tanggal", "Sumber Kas", "Nominal", "Keterangan", "Dibuat Oleh"}
	rows := make([][]string, 0, len(items))
	var total float64
	for _, it := range items {
		total += it.nominal()
		rows = append(rows, []string{
			it.Tanggal.Format("02/01/2006"),
			it.SumberKas,
			formatRupiah(it.Nominal),
			derefStr(it.Keterangan),
			it.DibuatOleh,
		})
	}
	summary := fmt.Sprintf("Total: %s", formatRupiah(total))

	pdfBytes, err := generatePDF("Laporan Kas", headers, rows, summary)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"laporan-kas-%s.pdf", time.Now().Format("20060102")))
	w.Write(pdfBytes)
}

func (h *ExportHandler) ExportKasExcel(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []KasRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_kas rk").
		Select("rk.tanggal, rk.sumber_kas, rk.nominal, rk.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN users u ON u.id = rk.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rk.deleted_at IS NULL")
	applyDateFilterToQuery(query, f).Order("rk.tanggal ASC").Scan(&items)

	headers := []string{"Tanggal", "Sumber Kas", "Nominal", "Keterangan", "Dibuat Oleh"}
	rows := make([][]string, 0, len(items))
	var total float64
	for _, it := range items {
		total += it.nominal()
		rows = append(rows, []string{
			it.Tanggal.Format("02/01/2006"),
			it.SumberKas,
			formatRupiah(it.Nominal),
			derefStr(it.Keterangan),
			it.DibuatOleh,
		})
	}
	summaryRow := []string{"", "Total", formatRupiah(total), "", ""}

	excelBytes, err := generateExcel("Laporan Kas", headers, rows, summaryRow)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate Excel")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"laporan-kas-%s.xlsx", time.Now().Format("20060102")))
	w.Write(excelBytes)
}

func (h *ExportHandler) ExportIuranPDF(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []IuranRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_iuran ri").
		Select("ri.tanggal, ji.nama as jenis_iuran, ri.nominal, ri.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN jenis_iuran ji ON ji.id = ri.jenis_iuran_id AND ji.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = ri.dibuat_oleh AND u.deleted_at IS NULL").
		Where("ri.deleted_at IS NULL")
	query = applyDateFilterToQuery(query, f)
	if f.JenisIuranID != "" {
		query = query.Where("ri.jenis_iuran_id = ?", f.JenisIuranID)
	}
	query.Order("ri.tanggal ASC").Scan(&items)

	headers := []string{"Tanggal", "Jenis Iuran", "Nominal", "Keterangan", "Dibuat Oleh"}
	rows := make([][]string, 0, len(items))
	var total float64
	for _, it := range items {
		total += it.Nominal
		rows = append(rows, []string{
			it.Tanggal.Format("02/01/2006"),
			it.JenisIuran,
			formatRupiah(it.Nominal),
			derefStr(it.Keterangan),
			it.DibuatOleh,
		})
	}
	summary := fmt.Sprintf("Total: %s", formatRupiah(total))

	pdfBytes, err := generatePDF("Laporan Iuran", headers, rows, summary)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"laporan-iuran-%s.pdf", time.Now().Format("20060102")))
	w.Write(pdfBytes)
}

func (h *ExportHandler) ExportIuranExcel(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []IuranRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_iuran ri").
		Select("ri.tanggal, ji.nama as jenis_iuran, ri.nominal, ri.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN jenis_iuran ji ON ji.id = ri.jenis_iuran_id AND ji.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = ri.dibuat_oleh AND u.deleted_at IS NULL").
		Where("ri.deleted_at IS NULL")
	query = applyDateFilterToQuery(query, f)
	if f.JenisIuranID != "" {
		query = query.Where("ri.jenis_iuran_id = ?", f.JenisIuranID)
	}
	query.Order("ri.tanggal ASC").Scan(&items)

	headers := []string{"Tanggal", "Jenis Iuran", "Nominal", "Keterangan", "Dibuat Oleh"}
	rows := make([][]string, 0, len(items))
	var total float64
	for _, it := range items {
		total += it.Nominal
		rows = append(rows, []string{
			it.Tanggal.Format("02/01/2006"),
			it.JenisIuran,
			formatRupiah(it.Nominal),
			derefStr(it.Keterangan),
			it.DibuatOleh,
		})
	}
	summaryRow := []string{"", "Total", formatRupiah(total), "", ""}

	excelBytes, err := generateExcel("Laporan Iuran", headers, rows, summaryRow)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate Excel")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"laporan-iuran-%s.xlsx", time.Now().Format("20060102")))
	w.Write(excelBytes)
}

func (h *ExportHandler) ExportPengeluaranPDF(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []PengeluaranRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran rp").
		Select("rp.tanggal, kp.nama as kategori_pengeluaran, rp.nominal, rp.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN kategori_pengeluaran kp ON kp.id = rp.kategori_pengeluaran_id AND kp.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = rp.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rp.deleted_at IS NULL")
	query = applyDateFilterToQuery(query, f)
	if f.KategoriPengeluaranID != "" {
		query = query.Where("rp.kategori_pengeluaran_id = ?", f.KategoriPengeluaranID)
	}
	query.Order("rp.tanggal ASC").Scan(&items)

	headers := []string{"Tanggal", "Kategori", "Nominal", "Keterangan", "Dibuat Oleh"}
	rows := make([][]string, 0, len(items))
	var total float64
	for _, it := range items {
		total += it.Nominal
		rows = append(rows, []string{
			it.Tanggal.Format("02/01/2006"),
			it.KategoriPengeluaran,
			formatRupiah(it.Nominal),
			derefStr(it.Keterangan),
			it.DibuatOleh,
		})
	}
	summary := fmt.Sprintf("Total: %s", formatRupiah(total))

	pdfBytes, err := generatePDF("Laporan Pengeluaran", headers, rows, summary)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"laporan-pengeluaran-%s.pdf", time.Now().Format("20060102")))
	w.Write(pdfBytes)
}

func (h *ExportHandler) ExportPengeluaranExcel(w http.ResponseWriter, r *http.Request) {
	f := parseFilter(r)

	var items []PengeluaranRow
	query := h.DB.WithContext(r.Context()).
		Table("rekap_pengeluaran rp").
		Select("rp.tanggal, kp.nama as kategori_pengeluaran, rp.nominal, rp.keterangan, u.username as dibuat_oleh").
		Joins("LEFT JOIN kategori_pengeluaran kp ON kp.id = rp.kategori_pengeluaran_id AND kp.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = rp.dibuat_oleh AND u.deleted_at IS NULL").
		Where("rp.deleted_at IS NULL")
	query = applyDateFilterToQuery(query, f)
	if f.KategoriPengeluaranID != "" {
		query = query.Where("rp.kategori_pengeluaran_id = ?", f.KategoriPengeluaranID)
	}
	query.Order("rp.tanggal ASC").Scan(&items)

	headers := []string{"Tanggal", "Kategori", "Nominal", "Keterangan", "Dibuat Oleh"}
	rows := make([][]string, 0, len(items))
	var total float64
	for _, it := range items {
		total += it.Nominal
		rows = append(rows, []string{
			it.Tanggal.Format("02/01/2006"),
			it.KategoriPengeluaran,
			formatRupiah(it.Nominal),
			derefStr(it.Keterangan),
			it.DibuatOleh,
		})
	}
	summaryRow := []string{"", "Total", formatRupiah(total), "", ""}

	excelBytes, err := generateExcel("Laporan Pengeluaran", headers, rows, summaryRow)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate Excel")
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"laporan-pengeluaran-%s.xlsx", time.Now().Format("20060102")))
	w.Write(excelBytes)
}

func applyDateFilterToQuery(query *gorm.DB, f ReportFilter) *gorm.DB {
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

func formatRupiah(n float64) string {
	return fmt.Sprintf("Rp %s", formatNumber(n))
}

func formatNumber(n float64) string {
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}

	rounded := int64(n)
	s := fmt.Sprintf("%d", rounded)

	result := ""
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if count > 0 && count%3 == 0 {
			result = "." + result
		}
		result = string(s[i]) + result
		count++
	}

	if neg {
		result = "-" + result
	}
	return result
}

func derefStr(s *string) string {
	if s == nil {
		return "-"
	}
	if *s == "" {
		return "-"
	}
	return *s
}

func (k KasRow) nominal() float64 {
	return k.Nominal
}
