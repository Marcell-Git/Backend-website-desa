package inventaris

import "time"

type Item struct {
	ID                string    `json:"id"`
	NamaBarang        string    `json:"nama_barang"`
	Kategori          string    `json:"kategori"`
	WarnaLabel        string    `json:"warna_label"`
	JumlahBarang      int       `json:"jumlah_barang"`
	KondisiBarang     string    `json:"kondisi_barang"`
	LokasiPenyimpanan string    `json:"lokasi_penyimpanan"`
	PenanggungJawab   string    `json:"penanggung_jawab"`
	PeriodePerolehan  string    `json:"periode_perolehan"`
	Foto              *string   `json:"foto,omitempty"`
	Keterangan        string    `json:"keterangan"`
	DibuatOleh        string    `json:"dibuat_oleh"`
	NamaUser          string    `json:"nama_user,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
