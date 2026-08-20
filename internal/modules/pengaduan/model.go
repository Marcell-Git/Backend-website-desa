package pengaduan

import "time"

type Pengaduan struct {
	ID           string    `json:"id"`
	NamaPelapor  string    `json:"nama_pelapor"`
	Lokasi       string    `json:"lokasi"`
	Keterangan   string    `json:"keterangan"`
	BuktiFile    *string   `json:"bukti_file,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PengaduanDetail struct {
	Pengaduan
	Riwayat []PengaduanRiwayat `json:"riwayat"`
}

type PengaduanRiwayat struct {
	ID            string    `json:"id"`
	PengaduanID   string    `json:"pengaduan_id"`
	Status        string    `json:"status"`
	Catatan       string    `json:"catatan"`
	DitanganiOleh string    `json:"ditangani_oleh"`
	CreatedAt     time.Time `json:"created_at"`
}

type UpdateStatusRequest struct {
	Status  string `json:"status"`
	Catatan string `json:"catatan"`
}

type AddCatatanRequest struct {
	Catatan string `json:"catatan"`
}
