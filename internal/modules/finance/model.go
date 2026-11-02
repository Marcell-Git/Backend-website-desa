package finance

import "time"

type JenisIuran struct {
	ID           string     `json:"id"`
	Nama         string     `json:"nama"`
	Deskripsi    *string    `json:"deskripsi,omitempty"`
	Nominal      float64    `json:"nominal"`
	TipePeriode  string     `json:"tipe_periode"`
	BulanMulai   *time.Time `json:"bulan_mulai"`
	BulanSelesai *time.Time `json:"bulan_selesai"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Tipe periode jenis iuran: selamanya ditagih tiap bulan tanpa batas akhir,
// periode hanya ditagih dari bulan mulai sampai bulan selesai.
const (
	tipeSelamanya = "selamanya"
	tipePeriode   = "periode"
)

type CreateJenisIuranRequest struct {
	Nama         string  `json:"nama"`
	Deskripsi    *string `json:"deskripsi"`
	Nominal      float64 `json:"nominal"`
	TipePeriode  string  `json:"tipe_periode"`
	BulanMulai   string  `json:"bulan_mulai"`
	BulanSelesai string  `json:"bulan_selesai"`
}

type UpdateJenisIuranRequest = CreateJenisIuranRequest

type KategoriPengeluaran struct {
	ID        string    `json:"id"`
	Nama      string    `json:"nama"`
	Deskripsi *string   `json:"deskripsi,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateKategoriPengeluaranRequest struct {
	Nama      string  `json:"nama"`
	Deskripsi *string `json:"deskripsi"`
}

type UpdateKategoriPengeluaranRequest struct {
	Nama      string  `json:"nama"`
	Deskripsi *string `json:"deskripsi"`
}

type RekapKas struct {
	ID         string    `json:"id"`
	Tanggal    time.Time `json:"tanggal"`
	SumberKas  string    `json:"sumber_kas"`
	Nominal    float64   `json:"nominal"`
	Keterangan *string   `json:"keterangan,omitempty"`
	DibuatOleh string    `json:"dibuat_oleh"`
	NamaUser   string    `json:"nama_user,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateRekapKasRequest struct {
	Tanggal    string  `json:"tanggal"`
	SumberKas  string  `json:"sumber_kas"`
	Nominal    float64 `json:"nominal"`
	Keterangan *string `json:"keterangan"`
}

type UpdateRekapKasRequest struct {
	Tanggal    string  `json:"tanggal"`
	SumberKas  string  `json:"sumber_kas"`
	Nominal    float64 `json:"nominal"`
	Keterangan *string `json:"keterangan"`
}

type RekapIuran struct {
	ID             string    `json:"id"`
	Tanggal        time.Time `json:"tanggal"`
	JenisIuranID   string    `json:"jenis_iuran_id"`
	NamaJenisIuran string    `json:"nama_jenis_iuran,omitempty"`
	Nominal        float64   `json:"nominal"`
	Keterangan     *string   `json:"keterangan,omitempty"`
	DibuatOleh     string    `json:"dibuat_oleh"`
	NamaUser       string    `json:"nama_user,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateRekapIuranRequest struct {
	Tanggal      string  `json:"tanggal"`
	JenisIuranID string  `json:"jenis_iuran_id"`
	Nominal      float64 `json:"nominal"`
	Keterangan   *string `json:"keterangan"`
}

type UpdateRekapIuranRequest struct {
	Tanggal      string  `json:"tanggal"`
	JenisIuranID string  `json:"jenis_iuran_id"`
	Nominal      float64 `json:"nominal"`
	Keterangan   *string `json:"keterangan"`
}

type RekapPengeluaran struct {
	ID                      string    `json:"id"`
	Tanggal                 time.Time `json:"tanggal"`
	KategoriPengeluaranID   string    `json:"kategori_pengeluaran_id"`
	NamaKategoriPengeluaran string    `json:"nama_kategori_pengeluaran,omitempty"`
	Nominal                 float64   `json:"nominal"`
	Keterangan              *string   `json:"keterangan,omitempty"`
	DibuatOleh              string    `json:"dibuat_oleh"`
	NamaUser                string    `json:"nama_user,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type CreateRekapPengeluaranRequest struct {
	Tanggal               string  `json:"tanggal"`
	KategoriPengeluaranID string  `json:"kategori_pengeluaran_id"`
	Nominal               float64 `json:"nominal"`
	Keterangan            *string `json:"keterangan"`
}

type UpdateRekapPengeluaranRequest struct {
	Tanggal               string  `json:"tanggal"`
	KategoriPengeluaranID string  `json:"kategori_pengeluaran_id"`
	Nominal               float64 `json:"nominal"`
	Keterangan            *string `json:"keterangan"`
}
