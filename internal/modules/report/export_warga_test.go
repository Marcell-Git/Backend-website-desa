package report

import "testing"

func TestAnggotaToCellsUrutanSesuaiHeader(t *testing.T) {
	pekerjaan := "Petani"
	ag := anggotaCetakRow{
		NIK:              "3273010101900001",
		NamaLengkap:      "Budi Santoso",
		JenisKelamin:     "L",
		TempatLahir:      "Klaten",
		TanggalLahir:     "12/04/1972",
		Agama:            "Islam",
		Pekerjaan:        &pekerjaan,
		HubunganKeluarga: "Kepala Keluarga",
		Status:           true,
	}

	cells := ag.toCells()
	if len(cells) != len(anggotaCetakHeaders) {
		t.Fatalf("jumlah sel %d harus sama dengan jumlah header %d", len(cells), len(anggotaCetakHeaders))
	}

	// Urutan sel harus persis mengikuti header, kalau tidak NIK dan Nama tertukar.
	ingin := map[string]string{
		"No":            "",
		"Nama":          "Budi Santoso",
		"NIK":           "3273010101900001",
		"Jenis Kelamin": "Laki-laki",
		"Tempat Lahir":  "Klaten",
		"Tgl Lahir":     "12/04/1972",
		"Agama":         "Islam",
		"Pekerjaan":     "Petani",
		"Hubungan":      "Kepala Keluarga",
		"Status":        "Aktif",
	}
	for i, h := range anggotaCetakHeaders {
		if cells[i] != ingin[h] {
			t.Errorf("kolom %q: dapat %q, ingin %q", h, cells[i], ingin[h])
		}
	}
}

func TestAnggotaToCellsToleranKolomKosong(t *testing.T) {
	ag := anggotaCetakRow{NIK: "123", NamaLengkap: "Tanpa Data"}
	cells := ag.toCells()
	if len(cells) != len(anggotaCetakHeaders) {
		t.Fatalf("baris harus tetap lengkap, dapat %d sel", len(cells))
	}
	if cells[1] != "Tanpa Data" {
		t.Fatalf("nama harus terisi, dapat %q", cells[1])
	}
	if cells[4] != "-" {
		t.Fatalf("tempat lahir kosong harus jadi -, dapat %q", cells[4])
	}
	if cells[7] != "-" {
		t.Fatalf("pekerjaan kosong harus jadi -, dapat %q", cells[7])
	}
	if cells[9] != "Nonaktif" {
		t.Fatalf("status bool false harus jadi Nonaktif, dapat %q", cells[9])
	}
}
