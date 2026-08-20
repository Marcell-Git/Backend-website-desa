package announcement

import "time"

type Pengumuman struct {
	ID               string    `json:"id"`
	Judul            string    `json:"judul"`
	Isi              string    `json:"isi"`
	TanggalPublikasi time.Time `json:"tanggal_publikasi"`
	Status           string    `json:"status"`
	DibuatOleh       string    `json:"dibuat_oleh"`
	NamaUser         string    `json:"nama_user,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreatePengumumanRequest struct {
	Judul            string `json:"judul"`
	Isi              string `json:"isi"`
	TanggalPublikasi string `json:"tanggal_publikasi"`
	Status           string `json:"status"`
}

type UpdatePengumumanRequest struct {
	Judul            string `json:"judul"`
	Isi              string `json:"isi"`
	TanggalPublikasi string `json:"tanggal_publikasi"`
	Status           string `json:"status"`
}
