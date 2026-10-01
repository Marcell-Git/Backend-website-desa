package report

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/jung-kurt/gofpdf"
)

func TestGeneratePDFLandscapeSisipkanSemuaBaris(t *testing.T) {
	rows := make([][]string, 0, 120)
	for i := 1; i <= 120; i++ {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i), "1521122006000606", "Sukarno", "3273010101900001",
			"Gg. Kenanga No. 4, Dukuh Tegalyoso Kulon", "RT 26 / RW 04", "4", "081234567890",
		})
	}

	pdf, err := generatePDFLandscape(PDFTable{
		Title:     "DAFTAR KARTU KELUARGA",
		Subtitle:  "Warga RT 26 Kelurahan Tegalyoso",
		Headers:   []string{"No", "Nomor KK", "Kepala Keluarga", "NIK", "Alamat", "RT/RW", "Jml Anggota", "No. HP"},
		Rows:      rows,
		Widths:    []float64{10, 28, 45, 32, 70, 18, 20, 35},
		Aligns:    []string{"C", "L", "L", "L", "L", "C", "C", "L"},
		TotalText: "Total: 120 Kartu Keluarga dengan 480 anggota",
	})
	if err != nil {
		t.Fatalf("generatePDFLandscape: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("hasil bukan PDF")
	}
	if h := countPDFPages(t, pdf); h < 3 {
		t.Fatalf("120 baris seharusnya menjadi beberapa halaman, dapat %d", h)
	}
}

func TestGeneratePDFLandscapeTanpaKolomDitolak(t *testing.T) {
	if _, err := generatePDFLandscape(PDFTable{Title: "x"}); err == nil {
		t.Fatal("harus error saat headers kosong")
	}
}

func TestLebarKolomDefaultMembagiRata(t *testing.T) {
	for i, v := range landscapeWidths(4) {
		if v != landscapeTotalWidth/4 {
			t.Fatalf("kolom %d lebar %.2f, ingin %.2f", i, v, landscapeTotalWidth/4)
		}
	}
	if len(landscapeAligns(3)) != 3 {
		t.Fatal("landscapeAligns harus sepanjang jumlah kolom")
	}
	if alignAt([]string{"L", "C"}, 5) != "L" {
		t.Fatal("alignAt harus fallback ke L di luar rentang")
	}
}

func TestWrapCellMemotongSesuaiLebarKolom(t *testing.T) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 8)

	teks := "Ds. Srandakan, Kec. Kalasan, Kab. Sleman"
	hasil := wrapCell(pdf, teks, 30, 0)
	if hasil == teks {
		t.Fatal("teks panjang harus dipecah")
	}
	for _, baris := range bytes.Split([]byte(hasil), []byte("\n")) {
		if w := pdf.GetStringWidth(string(baris)); w > 30-2*cellPadding {
			t.Fatalf("baris %.1fmm melebihi lebar efektif %.1fmm: %q", w, 30-2*cellPadding, baris)
		}
	}

	if got := wrapCell(pdf, "test", 30, 0); got != "test" {
		t.Fatalf("teks pendek harus tetap utuh, dapat %q", got)
	}
	if got := wrapCell(pdf, "", 30, 0); got != "" {
		t.Fatalf("teks kosong harus jadi kosong, dapat %q", got)
	}
}

func TestWrapCellMembatasiJumlahBaris(t *testing.T) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 8)

	panjang := "Ds. Srandakan Kec. Kalasan Kab. Slemanwritersudirman Nomor Satu Jakarta Timur"
	hasil := wrapCell(pdf, panjang, 40, 2)
	lines := bytes.Split([]byte(hasil), []byte("\n"))

	if len(lines) != 2 {
		t.Fatalf("harus tepat 2 baris, dapat %d: %q", len(lines), hasil)
	}
	if !bytes.HasSuffix(lines[1], []byte("...")) {
		t.Fatalf("baris terakhir harus diakhiri ellipsis, dapat %q", lines[1])
	}
	for _, baris := range lines {
		if w := pdf.GetStringWidth(string(baris)); w > 40-2*cellPadding {
			t.Fatalf("baris %.1fmm melebihi lebar efektif %.1fmm: %q", w, 40-2*cellPadding, baris)
		}
	}

	toleran := wrapCell(pdf, "Budi Santoso", 40, 2)
	if bytes.Contains([]byte(toleran), []byte("...")) {
		t.Fatalf("teks yang sudah muat tidak boleh dipotong: %q", toleran)
	}
}

func TestSplitLongWordMemotongKataPanjang(t *testing.T) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 8)

	parts := splitLongWord(pdf, "Karanganyarrejo", 20)
	if len(parts) < 2 {
		t.Fatalf("kata panjang harus dipecah, dapat %v", parts)
	}
	for _, p := range parts {
		if w := pdf.GetStringWidth(p); w > 20 {
			t.Fatalf("bagian %.1fmm melebihi 20mm: %q", w, p)
		}
	}
	if got := splitLongWord(pdf, "Klan", 20); len(got) != 1 || got[0] != "Klan" {
		t.Fatalf("kata yang muat tidak boleh dipecah, dapat %v", got)
	}
}

func TestRowHeightMenyesuaikanBarisTerpanjang(t *testing.T) {
	duaBaris := []string{"a\nb\nc", "x"}
	if h := rowHeight(duaBaris); h <= 6 {
		t.Fatalf("baris 3 baris harus lebih tinggi dari 6mm, dapat %.2f", h)
	}
	if h := rowHeight([]string{"a", "b"}); h != 6 {
		t.Fatalf("baris tunggal harus 6mm, dapat %.2f", h)
	}
}

func TestWriteTableRowToleranBarisPendek(t *testing.T) {
	pdf, err := generatePDFLandscape(PDFTable{
		Title:   "Uji",
		Headers: []string{"A", "B", "C"},
		Rows:    [][]string{{"1"}},
		Widths:  []float64{20, 20, 20},
	})
	if err != nil {
		t.Fatalf("baris dengan kolom kurang harus tetap bisa dicetak: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("hasil bukan PDF")
	}
}

func countPDFPages(t *testing.T, pdf []byte) int {
	t.Helper()
	re := regexp.MustCompile(`/Type\s*/Pages[^>]*?/Count\s+(\d+)`)
	if m := re.FindSubmatch(pdf); m != nil {
		var n int
		fmt.Sscanf(string(m[1]), "%d", &n)
		return n
	}
	return bytes.Count(pdf, []byte("/Type /Page"))
}
