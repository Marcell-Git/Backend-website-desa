package report

import "time"

type ReportFilter struct {
	Bulan                  string `json:"bulan"`
	Tahun                  string `json:"tahun"`
	TanggalMulai           string `json:"tanggal_mulai"`
	TanggalAkhir           string `json:"tanggal_akhir"`
	JenisIuranID           string `json:"jenis_iuran_id"`
	KategoriPengeluaranID  string `json:"kategori_pengeluaran_id"`
}

type KasRow struct {
	Tanggal   time.Time `json:"tanggal"`
	SumberKas string    `json:"sumber_kas"`
	Nominal   float64   `json:"nominal"`
	Keterangan *string   `json:"keterangan,omitempty"`
	DibuatOleh string    `json:"dibuat_oleh"`
}

type IuranRow struct {
	Tanggal     time.Time `json:"tanggal"`
	JenisIuran  string    `json:"jenis_iuran"`
	Nominal     float64   `json:"nominal"`
	Keterangan  *string   `json:"keterangan,omitempty"`
	DibuatOleh  string    `json:"dibuat_oleh"`
}

type PengeluaranRow struct {
	Tanggal                time.Time `json:"tanggal"`
	KategoriPengeluaran    string    `json:"kategori_pengeluaran"`
	Nominal                float64   `json:"nominal"`
	Keterangan             *string   `json:"keterangan,omitempty"`
	DibuatOleh             string    `json:"dibuat_oleh"`
}

type KeuanganSummary struct {
	TotalKas         float64 `json:"total_kas"`
	TotalIuran       float64 `json:"total_iuran"`
	TotalPengeluaran float64 `json:"total_pengeluaran"`
	SaldoAkhir       float64 `json:"saldo_akhir"`
	Bulan            string  `json:"bulan"`
	Tahun            string  `json:"tahun"`
}

type KeuanganReport struct {
	Summary         KeuanganSummary  `json:"summary"`
	Kas             []KasRow         `json:"kas"`
	Iuran           []IuranRow       `json:"iuran"`
	Pengeluaran     []PengeluaranRow `json:"pengeluaran"`
}
