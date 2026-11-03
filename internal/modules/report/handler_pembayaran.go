package report

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/modules/finance"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

var namaBulan = [12]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

func labelPeriode(t time.Time) string {
	return fmt.Sprintf("%s %d", namaBulan[t.Month()-1], t.Year())
}

// parsePeriodeLaporan membaca query periode (YYYY-MM). Bila tidak valid, respons
// error langsung ditulis dan ok bernilai false.
func parsePeriodeLaporan(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	periode, err := time.Parse("2006-01", r.URL.Query().Get("periode"))
	if err != nil {
		response.ValidationError(w, map[string][]string{"periode": {"Format periode harus YYYY-MM"}})
		return time.Time{}, false
	}
	return periode, true
}

// loadTagihan mengambil seluruh KK aktif beserta status bayarnya untuk satu tagihan.
func loadTagihan(ctx context.Context, db *gorm.DB, periode time.Time, jenis string, jenisIuranID *string, nama string) (TagihanPembayaran, error) {
	joinCond := "LEFT JOIN pembayaran_bulanan p ON p.kartu_keluarga_id = kk.id AND p.jenis = ? AND p.periode = ? AND p.deleted_at IS NULL"
	joinArgs := []interface{}{jenis, periode}
	if jenisIuranID != nil {
		joinCond += " AND p.jenis_iuran_id = ?"
		joinArgs = append(joinArgs, *jenisIuranID)
	}

	warga := make([]StatusPembayaranRow, 0)
	err := db.WithContext(ctx).
		Table("kartu_keluarga kk").
		Select(`kk.nomor_kk, a.nama_lengkap AS nama_kepala_keluarga, kk.alamat,
			(p.id IS NOT NULL) AS lunas, p.tanggal_bayar, p.nominal`).
		Joins("LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id AND a.deleted_at IS NULL").
		Joins(joinCond, joinArgs...).
		Where("kk.deleted_at IS NULL").
		Order("(p.id IS NOT NULL) ASC, a.nama_lengkap ASC NULLS LAST, kk.nomor_kk ASC").
		Scan(&warga).Error
	if err != nil {
		return TagihanPembayaran{}, err
	}

	t := TagihanPembayaran{Nama: nama, Warga: warga}
	for _, w := range warga {
		if !w.Lunas {
			t.BelumBayar++
			continue
		}
		t.SudahBayar++
		if w.Nominal != nil {
			t.TotalTerkumpul += *w.Nominal
		}
	}
	return t, nil
}

func (s StatusPembayaranRow) namaWarga() string {
	if s.NamaKepalaKeluarga != nil && *s.NamaKepalaKeluarga != "" {
		return *s.NamaKepalaKeluarga
	}
	return "KK " + s.NomorKK
}

// cells mengembalikan nama, alamat, status, tanggal bayar, dan nominal.
func (s StatusPembayaranRow) cells() []string {
	status, tanggal, nominal := "Belum bayar", "-", "-"
	if s.Lunas {
		status = "Sudah bayar"
		if s.TanggalBayar != nil {
			tanggal = helper.FormatTanggal(*s.TanggalBayar)
		}
		if s.Nominal != nil {
			nominal = formatRupiah(*s.Nominal)
		}
	}
	return []string{s.namaWarga(), orDash(s.Alamat), status, tanggal, nominal}
}

var (
	pembayaranHeaders = []string{"No", "Kepala Keluarga", "Alamat", "Status", "Tanggal Bayar", "Nominal"}
	pembayaranWidths  = []float64{10, 80, 100, 27, 30, 30}
	pembayaranAligns  = []string{"C", "L", "L", "C", "C", "R"}
)

const (
	pembayaranMaxLines = 2
	// Tinggi baris terbesar dengan pembayaranMaxLines. Halaman harus diganti sebelum
	// baris digambar, karena writeTableRow menggambar sel per sel dan page break
	// otomatis gofpdf di tengah baris akan merusak tabel.
	pembayaranRowMaxHeight = pembayaranMaxLines*cellLineHeight + 2*cellPadding
)

func generatePDFPembayaran(label string, t TagihanPembayaran) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	writePageHeader(pdf, "Status Pembayaran "+t.Nama, "Periode "+label)

	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 6, fmt.Sprintf("Sudah bayar: %d KK  -  Belum bayar: %d KK  -  Terkumpul: %s",
		t.SudahBayar, t.BelumBayar, formatRupiah(t.TotalTerkumpul)), "", 1, "L", false, 0, "")

	writeTableHeader(pdf, pembayaranHeaders, pembayaranWidths)
	pdf.SetFont("Helvetica", "", 8)
	for no, warga := range t.Warga {
		if pdf.GetY()+pembayaranRowMaxHeight > landscapeBreakY {
			pdf.AddPage()
			writeTableHeader(pdf, pembayaranHeaders, pembayaranWidths)
			pdf.SetFont("Helvetica", "", 8)
		}
		row := append([]string{fmt.Sprintf("%d", no+1)}, warga.cells()...)
		writeTableRow(pdf, row, pembayaranWidths, pembayaranAligns, pembayaranMaxLines)
	}

	writeFooter(pdf)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// tagihanKas memuat status pembayaran kas bulanan dari request ekspor.
func (h *ExportHandler) tagihanKas(w http.ResponseWriter, r *http.Request) (time.Time, TagihanPembayaran, bool) {
	periode, ok := parsePeriodeLaporan(w, r)
	if !ok {
		return periode, TagihanPembayaran{}, false
	}
	t, err := loadTagihan(r.Context(), h.DB, periode, "kas", nil, "Kas Bulanan")
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat status pembayaran")
		return periode, t, false
	}
	return periode, t, true
}

// tagihanIuran memuat status pembayaran satu jenis iuran dari request ekspor.
func (h *ExportHandler) tagihanIuran(w http.ResponseWriter, r *http.Request) (time.Time, TagihanPembayaran, bool) {
	periode, ok := parsePeriodeLaporan(w, r)
	if !ok {
		return periode, TagihanPembayaran{}, false
	}

	var ji database.JenisIuran
	query := finance.WhereIuranBerlaku(h.DB.WithContext(r.Context()).Where("id = ?", r.URL.Query().Get("jenis_iuran_id")), periode)
	if err := query.First(&ji).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Jenis iuran tidak ditagihkan pada "+labelPeriode(periode))
		return periode, TagihanPembayaran{}, false
	}

	t, err := loadTagihan(r.Context(), h.DB, periode, "iuran", &ji.ID, ji.Nama)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat status pembayaran")
		return periode, t, false
	}
	return periode, t, true
}

func writePembayaranPDF(w http.ResponseWriter, r *http.Request, db *gorm.DB, modul string, periode time.Time, t TagihanPembayaran) {
	pdfBytes, err := generatePDFPembayaran(labelPeriode(periode), t)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate PDF")
		return
	}

	auditPembayaran(r, db, modul, "PDF", periode, t)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"status-pembayaran-%s.pdf\"", periode.Format("2006-01")))
	w.Write(pdfBytes)
}

func writePembayaranExcel(w http.ResponseWriter, r *http.Request, db *gorm.DB, modul string, periode time.Time, t TagihanPembayaran) {
	headers := []string{"Tagihan", "Kepala Keluarga", "Alamat", "Status", "Tanggal Bayar", "Nominal"}
	rows := make([][]string, 0, len(t.Warga))
	for _, warga := range t.Warga {
		rows = append(rows, append([]string{t.Nama}, warga.cells()...))
	}
	summaryRow := []string{"Total Terkumpul", "", "", "", "", formatRupiah(t.TotalTerkumpul)}

	excelBytes, err := generateExcel("Pembayaran "+labelPeriode(periode), headers, rows, summaryRow)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate Excel")
		return
	}

	auditPembayaran(r, db, modul, "Excel", periode, t)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"status-pembayaran-%s.xlsx\"", periode.Format("2006-01")))
	w.Write(excelBytes)
}

func auditPembayaran(r *http.Request, db *gorm.DB, modul, format string, periode time.Time, t TagihanPembayaran) {
	uid := middleware.GetUserID(r)
	database.RecordAudit(r.Context(), db, &uid, modul, "Export Status Pembayaran",
		fmt.Sprintf("Mengekspor %s status pembayaran %s %s", format, t.Nama, labelPeriode(periode)),
		nil, map[string]any{"jumlah_data": len(t.Warga)})
}

func (h *ExportHandler) ExportPembayaranKasPDF(w http.ResponseWriter, r *http.Request) {
	if periode, t, ok := h.tagihanKas(w, r); ok {
		writePembayaranPDF(w, r, h.DB, "Rekap Kas", periode, t)
	}
}

func (h *ExportHandler) ExportPembayaranKasExcel(w http.ResponseWriter, r *http.Request) {
	if periode, t, ok := h.tagihanKas(w, r); ok {
		writePembayaranExcel(w, r, h.DB, "Rekap Kas", periode, t)
	}
}

func (h *ExportHandler) ExportPembayaranIuranPDF(w http.ResponseWriter, r *http.Request) {
	if periode, t, ok := h.tagihanIuran(w, r); ok {
		writePembayaranPDF(w, r, h.DB, "Rekap Iuran", periode, t)
	}
}

func (h *ExportHandler) ExportPembayaranIuranExcel(w http.ResponseWriter, r *http.Request) {
	if periode, t, ok := h.tagihanIuran(w, r); ok {
		writePembayaranExcel(w, r, h.DB, "Rekap Iuran", periode, t)
	}
}
