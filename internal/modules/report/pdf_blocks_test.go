package report

import (
	"bytes"
	"fmt"
	"testing"
)

func testKKBlock(nomor string, jumlah int) KKBlock {
	b := KKBlock{
		NomorKK:    nomor,
		NamaKepala: "Budi Santoso",
		NIKKepala:  "3273010101900001",
		Alamat:     "Gg. Melati No. 12, Dukuh Tegalyoso Kulon",
		RT:         "26",
		RW:         "04",
		NomorHP:    "081234560101",
	}
	for i := 1; i <= jumlah; i++ {
		b.Anggota = append(b.Anggota, []string{
			fmt.Sprintf("%d", i),
			fmt.Sprintf("Anggota Nama %02d", i),
			fmt.Sprintf("32730102019%05d", i),
			"Laki-laki",
			"Klaten",
			"01/02/1990",
			"Islam",
			"Karyawan Swasta",
			"Anak",
			"Aktif",
		})
	}
	return b
}

func TestGeneratePDFKKBlocksSisipkanSemuaBlok(t *testing.T) {
	blocks := make([]KKBlock, 0, 13)
	for i := 1; i <= 13; i++ {
		blocks = append(blocks, testKKBlock(fmt.Sprintf("1521122001%010d", i), 3))
	}

	pdf, err := generatePDFKKBlocks("DAFTAR KARTU KELUARGA", "Warga RT 26", blocks)
	if err != nil {
		t.Fatalf("generatePDFKKBlocks: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("hasil bukan PDF")
	}
	if h := countPDFPages(t, pdf); h < 2 {
		t.Fatalf("13 blok sebaiknya butuh lebih dari 1 halaman, dapat %d", h)
	}
}

func TestGeneratePDFKKBlocksTanpaAnggotaTetapDorcet(t *testing.T) {
	pdf, err := generatePDFKKBlocks("DAFTAR KARTU KELUARGA", "", []KKBlock{testKKBlock("1521122001000101", 0)})
	if err != nil {
		t.Fatalf("KK tanpa anggota tidak boleh gagal: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("hasil bukan PDF")
	}
}

func TestGeneratePDFKKBlocksKosongTidakGagal(t *testing.T) {
	pdf, err := generatePDFKKBlocks("DAFTAR KARTU KELUARGA", "", nil)
	if err != nil {
		t.Fatalf("daftar KK kosong tidak boleh gagal: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("hasil bukan PDF")
	}
}

func TestKKBlockHeightCukupUntukInfoDanTabel(t *testing.T) {
	// 1 baris judul blok + 3 baris info + header tabel + 4 baris anggota + jarak.
	denganAnggota := kkBlockHeight(testKKBlock("1521122001000101", 4).Anggota)
	tanpaAnggota := kkBlockHeight(nil)
	if denganAnggota <= tanpaAnggota {
		t.Fatalf("blok dengan anggota harus lebih tinggi: %.1f vs %.1f", denganAnggota, tanpaAnggota)
	}
	if tanpaAnggota < 4*kkBlockInfoHeight {
		t.Fatalf("blok harus memuat judul dan 3 baris info, dapat %.1f", tanpaAnggota)
	}
	if got := kkBlockHeight(testKKBlock("x", 4).Anggota) - kkBlockHeight(testKKBlock("x", 2).Anggota); got != 12 {
		t.Fatalf("dua anggota tambahan harus menambah 12mm, dapat %.1f", got)
	}
}

func TestKolomAnggotaSesuaiLebarTotal(t *testing.T) {
	if len(anggotaCetakWidths) != len(anggotaCetakHeaders) || len(anggotaCetakAligns) != len(anggotaCetakHeaders) {
		t.Fatal("jumlah header, lebar, dan perataan harus sama")
	}
	total := 0.0
	for _, w := range anggotaCetakWidths {
		total += w
	}
	if total != landscapeTotalWidth {
		t.Fatalf("total lebar kolom %.1f harus sama dengan %.1f", total, landscapeTotalWidth)
	}
}
