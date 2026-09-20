package beritakegiatan

import "time"

type Item struct {
	ID         string    `json:"id"`
	Kategori   string    `json:"kategori"`
	WarnaLabel string    `json:"warna_label"`
	Judul      string    `json:"judul"`
	Tanggal    time.Time `json:"tanggal"`
	Jam        string    `json:"jam"`
	Lokasi     string    `json:"lokasi"`
	Keterangan string    `json:"keterangan"`
	DibuatOleh string    `json:"dibuat_oleh"`
	NamaUser   string    `json:"nama_user,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UpsertRequest struct {
	Kategori   string `json:"kategori"`
	WarnaLabel string `json:"warna_label"`
	Judul      string `json:"judul"`
	Tanggal    string `json:"tanggal"`
	Jam        string `json:"jam"`
	Lokasi     string `json:"lokasi"`
	Keterangan string `json:"keterangan"`
}
