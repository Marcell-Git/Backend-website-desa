package database

import (
	"time"

	"gorm.io/gorm"
)

type KartuKeluarga struct {
	ID               string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NomorKK          string         `gorm:"type:varchar(16);uniqueIndex" json:"nomor_kk"`
	KepalaKeluargaID *string        `gorm:"type:uuid" json:"kepala_keluarga_id,omitempty"`
	NomorHP          *string        `gorm:"type:varchar(20)" json:"nomor_hp,omitempty"`
	Alamat           string         `gorm:"type:text" json:"alamat"`
	RT               string         `gorm:"type:varchar(5)" json:"rt"`
	RW               string         `gorm:"type:varchar(5)" json:"rw"`
	Desa             string         `gorm:"type:varchar(100)" json:"desa"`
	Kecamatan        string         `gorm:"type:varchar(100)" json:"kecamatan"`
	Kabupaten        string         `gorm:"type:varchar(100)" json:"kabupaten"`
	Provinsi         string         `gorm:"type:varchar(100)" json:"provinsi"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KartuKeluarga) TableName() string { return "kartu_keluarga" }

type AnggotaKeluarga struct {
	ID               string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	KartuKeluargaID  string         `gorm:"type:uuid" json:"kartu_keluarga_id"`
	NIK              string         `gorm:"type:varchar(16);uniqueIndex" json:"nik"`
	NamaLengkap      string         `gorm:"type:varchar(150)" json:"nama_lengkap"`
	TempatLahir      string         `gorm:"type:varchar(100)" json:"tempat_lahir"`
	TanggalLahir     string         `gorm:"type:date" json:"tanggal_lahir"`
	JenisKelamin     string         `gorm:"type:varchar(1);check:jenis_kelamin IN ('L','P')" json:"jenis_kelamin"`
	Agama            string         `gorm:"type:varchar(30)" json:"agama"`
	Pekerjaan        *string        `gorm:"type:varchar(100)" json:"pekerjaan,omitempty"`
	NomorHP          *string        `gorm:"type:varchar(20)" json:"nomor_hp,omitempty"`
	HubunganKeluarga string         `gorm:"type:varchar(50)" json:"hubungan_keluarga"`
	Status           bool           `gorm:"default:true" json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AnggotaKeluarga) TableName() string { return "anggota_keluarga" }

type AnakKos struct {
	ID           string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NIK          string         `gorm:"type:varchar(16);uniqueIndex" json:"nik"`
	NamaLengkap  string         `gorm:"type:varchar(150)" json:"nama_lengkap"`
	JenisKelamin string         `gorm:"type:varchar(1);check:jenis_kelamin IN ('L','P')" json:"jenis_kelamin"`
	TanggalLahir string         `gorm:"type:date" json:"tanggal_lahir"`
	Agama        string         `gorm:"type:varchar(30)" json:"agama"`
	Pekerjaan    *string        `gorm:"type:varchar(100)" json:"pekerjaan,omitempty"`
	NomorHP      *string        `gorm:"type:varchar(20)" json:"nomor_hp,omitempty"`
	AlamatKos    string         `gorm:"type:text;not null;default:''" json:"alamat_kos"`
	AlamatAsal   string         `gorm:"type:text" json:"alamat_asal"`
	Status       bool           `gorm:"default:true" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AnakKos) TableName() string { return "anak_kos" }
