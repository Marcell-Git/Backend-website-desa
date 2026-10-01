package report

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

// KKBlock adalah satu Kartu Keluarga beserta seluruh anggotanya, dicetak sebagai
// satu blok: baris informasi KK lalu tabel anggota di bawahnya. Berbeda dengan
// PDFTable yang mendatar, dokumen ini berulang memperlambat setiap KK.
type KKBlock struct {
	NomorKK    string
	NamaKepala string
	NIKKepala  string
	Alamat     string
	RT         string
	RW         string
	NomorHP    string
	Anggota    [][]string
}

var anggotaCetakHeaders = []string{
	"No", "Nama", "NIK", "Jenis Kelamin", "Tempat Lahir",
	"Tgl Lahir", "Agama", "Pekerjaan", "Hubungan", "Status",
}

var anggotaCetakWidths = []float64{10, 48, 34, 22, 30, 24, 22, 33, 32, 22}

var anggotaCetakAligns = []string{"C", "L", "L", "C", "L", "C", "L", "L", "L", "C"}

const (
	kkBlockInfoHeight = 6.0
	kkBlockGap        = 4.0
)

func kkBlockHeight(anggota [][]string) float64 {
	h := 4*kkBlockInfoHeight + kkBlockGap
	if len(anggota) > 0 {
		h += 7 // baris header tabel
		h += float64(len(anggota)) * 6
	}
	return h
}

func writePageHeader(pdf *gofpdf.Fpdf, title, subtitle string) {
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 10, "RT 26 Kelurahan Tegalyoso", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, "Kecamatan Klaten Selatan, Kabupaten Klaten", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, title, "", 1, "C", false, 0, "")
	pdf.Ln(2)

	if subtitle != "" {
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 6, subtitle, "", 1, "C", false, 0, "")
		pdf.Ln(4)
	}
}

func writeContinuedHeader(pdf *gofpdf.Fpdf, title string) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, title, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "RT 26 Kelurahan Tegalyoso - "+formatTanggalCetak(), "", 1, "C", false, 0, "")
	pdf.Ln(4)
}

// writeKKInfo writes the block header: nomor KK barbold, lalu tiga baris
// label-value (dua pasang per baris).
func writeKKInfo(pdf *gofpdf.Fpdf, b KKBlock) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(241, 245, 249)
	pdf.CellFormat(landscapeTotalWidth, kkBlockInfoHeight, "KARTU KELUARGA  "+b.NomorKK, "1", 0, "L", true, 0, "")
	pdf.Ln(-1)

	for _, row := range [][2][2]string{
		{
			{"Kepala Keluarga", b.NamaKepala},
			{"Jumlah Anggota", fmt.Sprintf("%d orang", len(b.Anggota))},
		},
		{
			{"NIK Kepala Keluarga", b.NIKKepala},
			{"No. HP", b.NomorHP},
		},
		{
			{"Alamat", b.Alamat},
			{"RT / RW", "RT " + orDash(b.RT) + " / RW " + orDash(b.RW)},
		},
	} {
		writeLabeledRow(pdf, row[0], row[1])
	}
}

func writeLabeledRow(pdf *gofpdf.Fpdf, kiri, kanan [2]string) {
	labelW := 38.0
	valueW := (landscapeTotalWidth - 2*labelW) / 2

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetFillColor(241, 245, 249)
	pdf.CellFormat(labelW, kkBlockInfoHeight, kiri[0], "1", 0, "L", true, 0, "")
	pdf.CellFormat(valueW, kkBlockInfoHeight, orDash(kiri[1]), "1", 0, "L", false, 0, "")

	pdf.SetFillColor(241, 245, 249)
	pdf.CellFormat(labelW, kkBlockInfoHeight, kanan[0], "1", 0, "L", true, 0, "")
	pdf.CellFormat(valueW, kkBlockInfoHeight, orDash(kanan[1]), "1", 0, "L", false, 0, "")
	pdf.Ln(-1)
}

func generatePDFKKBlocks(title, subtitle string, blocks []KKBlock) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	writePageHeader(pdf, title, subtitle)

	for _, b := range blocks {
		if pdf.GetY()+kkBlockHeight(b.Anggota) > landscapeBreakY {
			pdf.AddPage()
			writeContinuedHeader(pdf, title)
		}

		writeKKInfo(pdf, b)
		pdf.Ln(kkBlockGap)

		if len(b.Anggota) == 0 {
			pdf.SetFont("Helvetica", "I", 8)
			pdf.CellFormat(0, 6, "Belum ada anggota keluarga pada KK ini.", "1", 0, "L", false, 0, "")
			pdf.Ln(kkBlockGap)
			continue
		}

		writeTableHeader(pdf, anggotaCetakHeaders, anggotaCetakWidths)
		pdf.SetFont("Helvetica", "", 8)
		for _, row := range b.Anggota {
			writeTableRow(pdf, row, anggotaCetakWidths, anggotaCetakAligns, 0)
		}
		pdf.Ln(kkBlockGap)
	}

	if len(blocks) == 0 {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.CellFormat(0, 8, "Belum ada data Kartu Keluarga.", "1", 1, "C", false, 0, "")
	}

	writeFooter(pdf)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
