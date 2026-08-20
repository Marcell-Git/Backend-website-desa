package masterdata

import "time"

type KartuKeluarga struct {
	ID                 string     `json:"id"`
	NomorKK            string     `json:"nomor_kk"`
	KepalaKeluargaID   *string    `json:"kepala_keluarga_id,omitempty"`
	NamaKepala         *string    `json:"nama_kepala_keluarga,omitempty"`
	NIKKepala          *string    `json:"nik_kepala_keluarga,omitempty"`
	TempatLahirKepala  *string    `json:"tempat_lahir_kepala,omitempty"`
	TanggalLahirKepala string     `json:"tanggal_lahir_kepala,omitempty"`
	NomorHP            *string    `json:"nomor_hp,omitempty"`
	Alamat             string     `json:"alamat"`
	RT                 string     `json:"rt"`
	RW                 string     `json:"rw"`
	Desa               string     `json:"desa"`
	Kecamatan          string     `json:"kecamatan"`
	Kabupaten          string     `json:"kabupaten"`
	Provinsi           string     `json:"provinsi"`
	JumlahAnggota      int        `json:"jumlah_anggota"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CreateKKRequest struct {
	NomorKK   string                `json:"nomor_kk"`
	NomorHP   *string               `json:"nomor_hp"`
	Alamat    string                `json:"alamat"`
	RT        string                `json:"rt"`
	RW        string                `json:"rw"`
	Desa      string                `json:"desa"`
	Kecamatan string                `json:"kecamatan"`
	Kabupaten string                `json:"kabupaten"`
	Provinsi  string                `json:"provinsi"`
	Kepala    *CreateAnggotaRequest `json:"kepala_keluarga"`
}

type UpdateKKRequest struct {
	NomorHP   *string `json:"nomor_hp"`
	Alamat    string  `json:"alamat"`
	RT        string  `json:"rt"`
	RW        string  `json:"rw"`
	Desa      string  `json:"desa"`
	Kecamatan string  `json:"kecamatan"`
	Kabupaten string  `json:"kabupaten"`
	Provinsi  string  `json:"provinsi"`
}

type Anggota struct {
	ID               string    `json:"id"`
	KartuKeluargaID  string    `json:"kartu_keluarga_id"`
	NIK              string    `json:"nik"`
	NamaLengkap      string    `json:"nama_lengkap"`
	TempatLahir      string    `json:"tempat_lahir"`
	TanggalLahir     string    `json:"tanggal_lahir"`
	JenisKelamin     string    `json:"jenis_kelamin"`
	Agama            string    `json:"agama"`
	Pekerjaan        *string   `json:"pekerjaan,omitempty"`
	NomorHP          *string   `json:"nomor_hp,omitempty"`
	HubunganKeluarga string    `json:"hubungan_keluarga"`
	Status           bool      `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Anggota) TableName() string { return "anggota_keluarga" }

type CreateAnggotaRequest struct {
	KartuKeluargaID  string  `json:"kartu_keluarga_id"`
	NIK              string  `json:"nik"`
	NamaLengkap      string  `json:"nama_lengkap"`
	TempatLahir      string  `json:"tempat_lahir"`
	TanggalLahir     string  `json:"tanggal_lahir"`
	JenisKelamin     string  `json:"jenis_kelamin"`
	Agama            string  `json:"agama"`
	Pekerjaan        *string `json:"pekerjaan"`
	NomorHP          *string `json:"nomor_hp"`
	HubunganKeluarga string  `json:"hubungan_keluarga"`
	Status           *bool   `json:"status"`
}

type UpdateAnggotaRequest struct {
	NamaLengkap      string  `json:"nama_lengkap"`
	NomorHP          *string `json:"nomor_hp"`
	Agama            string  `json:"agama"`
	Pekerjaan        *string `json:"pekerjaan"`
	HubunganKeluarga string  `json:"hubungan_keluarga"`
	Status           *bool   `json:"status"`
}

type MoveAnggotaRequest struct {
	KartuKeluargaID string `json:"kartu_keluarga_id"`
}
