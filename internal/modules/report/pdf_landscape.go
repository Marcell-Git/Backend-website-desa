package report

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// PDFTable mendeskripsikan tabel PDF berorientasi landscape (A4) dengan lebar dan
// perataan kolom yang ditentukan pemanggil. Dipakai untuk data master (KK, anak
// kos) yang kolomnya terlalu banyak untuk muat di A4 potrait.
type PDFTable struct {
	Title    string
	Subtitle string
	Headers  []string
	Rows     [][]string
	// Widths lebar tiap kolom dalam mm. Kalau nil, lebar dibagi rata.
	Widths []float64
	// Aligns perataan tiap kolom: "L", "C", atau "R". Kalau nil, semua rata kiri
	// kecuali kolom pertama.
	Aligns []string
	// MaxLines batas jumlah baris per sel. Kalau 0, teks boleh sesuka hati.
	// Teks yang melebihi batas dipotong dan diberi tanda ellipsis.
	MaxLines int
	// TotalText ringkasan yang dicetak setelah tabel, misal "Total: 42 data".
	TotalText string
}

const (
	landscapeTotalWidth = 277.0
	landscapeBreakY     = 190.0
)

func landscapeWidths(n int) []float64 {
	widths := make([]float64, n)
	if n == 0 {
		return widths
	}
	each := landscapeTotalWidth / float64(n)
	for i := range widths {
		widths[i] = each
	}
	return widths
}

func landscapeAligns(n int) []string {
	aligns := make([]string, n)
	for i := range aligns {
		if i == 0 {
			aligns[i] = "C"
		} else {
			aligns[i] = "L"
		}
	}
	return aligns
}

func alignAt(aligns []string, i int) string {
	if i < len(aligns) {
		return aligns[i]
	}
	return "L"
}

const (
	cellLineHeight = 3.6
	cellPadding    = 1.5
)

func writeTableHeader(pdf *gofpdf.Fpdf, headers []string, widths []float64) {
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
}

// splitLongWord memotong satu kata yang lebih lebar dari kolom, dipisah di
// karakter terdekat. Tidak ada di path normal karena data warga kata terpanjangnya
// masih muat, tapi mencegah teks keluar kotak kalau nanti ada kolom sempit.
func splitLongWord(pdf *gofpdf.Fpdf, word string, avail float64) []string {
	if pdf.GetStringWidth(word) <= avail {
		return []string{word}
	}

	parts := make([]string, 0, 4)
	runes := []rune(word)
	mulai := 0
	for mulai < len(runes) {
		akhir := mulai
		for akhir < len(runes) && pdf.GetStringWidth(string(runes[mulai:akhir+1])) <= avail {
			akhir++
		}
		if akhir == mulai {
			akhir++
		}
		parts = append(parts, string(runes[mulai:akhir]))
		mulai = akhir
	}
	return parts
}

// wrapCell memecah teks sesuai lebar kolom. gofpdf tidak memotong sel otomatis,
// jadi tanpa ini teks panjang seperti alamat akan keluar dari kotak kolom.
// Kalau maxLines > 0, baris yang melebihi batas dipotong dan diberi "...".
func wrapCell(pdf *gofpdf.Fpdf, text string, width float64, maxLines int) string {
	avail := width - 2*cellPadding
	if avail <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	lines := make([]string, 0, len(words))
	current := ""
	for _, word := range words {
		for _, part := range splitLongWord(pdf, word, avail) {
			candidate := strings.TrimSpace(current + " " + part)
			if current == "" {
				current = part
			} else if pdf.GetStringWidth(candidate) <= avail {
				current = candidate
			} else {
				lines = append(lines, current)
				current = part
			}
		}
	}
	lines = append(lines, current)

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		last := maxLines - 1
		lines[last] = ellipsize(pdf, lines[last], avail)
	}
	return strings.Join(lines, "\n")
}

// ellipsize memotong teks dari belakang sampai muat di avail dan menambah "...".
func ellipsize(pdf *gofpdf.Fpdf, text string, avail float64) string {
	const dots = "..."
	dotsW := pdf.GetStringWidth(dots)
	if avail <= dotsW {
		return dots
	}

	runes := []rune(text)
	for len(runes) > 0 && pdf.GetStringWidth(string(runes)+dots) > avail {
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return dots
	}
	return strings.TrimSpace(string(runes)) + dots
}

func rowHeight(cells []string) float64 {
	tertinggi := 1
	for _, c := range cells {
		if n := strings.Count(c, "\n") + 1; n > tertinggi {
			tertinggi = n
		}
	}
	h := float64(tertinggi)*cellLineHeight + 2
	if h < 6 {
		return 6
	}
	return h
}

func wrapRow(pdf *gofpdf.Fpdf, row []string, widths []float64, maxLines int) []string {
	wrapped := make([]string, len(widths))
	for i := range widths {
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		wrapped[i] = wrapCell(pdf, cell, widths[i], maxLines)
	}
	return wrapped
}

// writeTableRow menggambar satu baris tabel sel per sel.
// Setiap baris teks digambar terpisah dengan CellFormat lalu kotak sel digambar
// dengan Rect. gofpdf MultiCell memperlakukan h sebagai tinggi per baris teks,
// bukan tinggi sel, sehingga tinggi baris tabel jadi tidak seragam.
func writeTableRow(pdf *gofpdf.Fpdf, row []string, widths []float64, aligns []string, maxLines int) {
	wrapped := wrapRow(pdf, row, widths, maxLines)
	lineCount := 1
	for _, c := range wrapped {
		if n := strings.Count(c, "\n") + 1; n > lineCount {
			lineCount = n
		}
	}

	// Semua sel memakai tinggi yang sama agar garis tabel tetap lurus.
	rowH := float64(lineCount)*cellLineHeight + 2*cellPadding
	if rowH < 6 {
		rowH = 6
	}

	startX, y := pdf.GetX(), pdf.GetY()
	x := startX
	for i := range widths {
		lines := strings.Split(wrapped[i], "\n")
		for li, line := range lines {
			pdf.SetXY(x, y+float64(li)*cellLineHeight+cellPadding)
			pdf.CellFormat(widths[i], cellLineHeight, line, "0", 0, alignAt(aligns, i), false, 0, "")
		}
		pdf.Rect(x, y, widths[i], rowH, "D")
		x += widths[i]
	}
	pdf.SetXY(startX, y+rowH)
}

func writeFooter(pdf *gofpdf.Fpdf) {
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Dicetak pada: "+formatTanggalCetak(), "", 1, "R", false, 0, "")
}

func formatTanggalCetak() string {
	return time.Now().Format("02 January 2006 15:04")
}

func generatePDFLandscape(t PDFTable) ([]byte, error) {
	if len(t.Headers) == 0 {
		return nil, fmt.Errorf("tabel pdf membutuhkan minimal satu kolom")
	}

	widths := t.Widths
	if len(widths) != len(t.Headers) {
		widths = landscapeWidths(len(t.Headers))
	}
	aligns := t.Aligns
	if len(aligns) != len(t.Headers) {
		aligns = landscapeAligns(len(t.Headers))
	}

	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 10, "RT 26 Kelurahan Tegalyoso", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, "Kecamatan Klaten Selatan, Kabupaten Klaten", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, t.Title, "", 1, "C", false, 0, "")
	pdf.Ln(2)

	if t.Subtitle != "" {
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 6, t.Subtitle, "", 1, "C", false, 0, "")
		pdf.Ln(4)
	}

	writeTableHeader(pdf, t.Headers, widths)

	pdf.SetFont("Helvetica", "", 8)
	for _, row := range t.Rows {
		if pdf.GetY() > landscapeBreakY {
			pdf.AddPage()
			writeTableHeader(pdf, t.Headers, widths)
			pdf.SetFont("Helvetica", "", 8)
		}
		writeTableRow(pdf, row, widths, aligns, t.MaxLines)
	}

	if t.TotalText != "" {
		pdf.Ln(3)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(0, 7, t.TotalText, "", 1, "L", false, 0, "")
	}

	writeFooter(pdf)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
