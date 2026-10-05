package database

import (
	"time"

	"gorm.io/gorm"
)

type Pengaduan struct {
	ID          string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NamaPelapor string         `gorm:"type:varchar(150);not null" json:"nama_pelapor"`
	Lokasi      string         `gorm:"type:text;not null" json:"lokasi"`
	Keterangan  string         `gorm:"type:text;not null" json:"keterangan"`
	BuktiFile   *string        `gorm:"type:varchar(255)" json:"bukti_file,omitempty"`
	Status      string         `gorm:"type:varchar(20);not null;default:menunggu" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Pengaduan) TableName() string { return "pengaduan" }

type PengaduanRiwayat struct {
	ID            string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PengaduanID   string    `gorm:"type:uuid;not null" json:"pengaduan_id"`
	Status        string    `gorm:"type:varchar(20);not null" json:"status"`
	Catatan       string    `gorm:"type:text;not null" json:"catatan"`
	DitanganiOleh string    `gorm:"type:varchar(150);not null" json:"ditangani_oleh"`
	CreatedAt     time.Time `json:"created_at"`
}

func (PengaduanRiwayat) TableName() string { return "pengaduan_riwayat" }
