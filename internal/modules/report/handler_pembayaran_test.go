package report

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func testTagihan(nama string, jumlah int) TagihanPembayaran {
	t := TagihanPembayaran{Nama: nama}
	for i := 0; i < jumlah; i++ {
		namaKepala := fmt.Sprintf("Warga %d", i+1)
		t.Warga = append(t.Warga, StatusPembayaranRow{
			NomorKK:            fmt.Sprintf("33100000000%05d", i),
			NamaKepalaKeluarga: &namaKepala,
			Alamat:             "Tegalyoso RT 26 RW 08, Klaten Selatan, Kabupaten Klaten, Jawa Tengah, alamat yang cukup panjang sampai dua baris atau bahkan lebih dari dua baris pada kolom alamat",
		})
	}
	return t
}

func TestGeneratePDFPembayaranGantiHalamanUtuh(t *testing.T) {
	pdf, err := generatePDFPembayaran("November 2026", testTagihan("Kas Bulanan", 120))
	if err != nil {
		t.Fatalf("generatePDFPembayaran: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("hasil bukan PDF")
	}
	// 120 baris dua-teks @ 10,2 mm, sekitar 16 baris per halaman: page break otomatis
	// di tengah baris akan menggandakan jumlah halaman jauh melebihi perkiraan ini.
	if pages := countPDFPages(t, pdf); pages < 7 || pages > 10 {
		t.Fatalf("jumlah halaman = %d, ingin 7-10", pages)
	}
}

func TestGeneratePDFPembayaranTanpaKKTidakGagal(t *testing.T) {
	pdf, err := generatePDFPembayaran("November 2026", TagihanPembayaran{Nama: "Kas Bulanan"})
	if err != nil {
		t.Fatalf("generatePDFPembayaran: %v", err)
	}
	if pages := countPDFPages(t, pdf); pages != 1 {
		t.Fatalf("jumlah halaman = %d, ingin 1", pages)
	}
}

func TestStatusPembayaranCells(t *testing.T) {
	nama := "Budi"
	nominal := 25000.0
	tanggal := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)

	lunas := StatusPembayaranRow{NomorKK: "123", NamaKepalaKeluarga: &nama, Alamat: "Jl. A", Lunas: true, TanggalBayar: &tanggal, Nominal: &nominal}
	want := []string{"Budi", "Jl. A", "Sudah bayar", "03-Nov-2026", "Rp 25.000"}
	if got := lunas.cells(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("cells lunas = %v, ingin %v", got, want)
	}

	belum := StatusPembayaranRow{NomorKK: "123"}
	want = []string{"KK 123", "-", "Belum bayar", "-", "-"}
	if got := belum.cells(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("cells belum = %v, ingin %v", got, want)
	}
}
