package database

import (
	"time"

	"gorm.io/gorm"
)

type JenisIuran struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Nama      string         `gorm:"type:varchar(100);uniqueIndex" json:"nama"`
	Deskripsi *string        `gorm:"type:text" json:"deskripsi,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (JenisIuran) TableName() string { return "jenis_iuran" }

type KategoriPengeluaran struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Nama      string         `gorm:"type:varchar(100);uniqueIndex" json:"nama"`
	Deskripsi *string        `gorm:"type:text" json:"deskripsi,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KategoriPengeluaran) TableName() string { return "kategori_pengeluaran" }

type RekapKas struct {
	ID         string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Tanggal    time.Time      `gorm:"type:date" json:"tanggal"`
	SumberKas  string         `gorm:"type:varchar(150)" json:"sumber_kas"`
	Nominal    float64        `gorm:"type:numeric(15,2)" json:"nominal"`
	Keterangan *string        `gorm:"type:text" json:"keterangan,omitempty"`
	DibuatOleh string         `gorm:"type:uuid" json:"dibuat_oleh"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RekapKas) TableName() string { return "rekap_kas" }

type RekapIuran struct {
	ID           string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Tanggal      time.Time      `gorm:"type:date" json:"tanggal"`
	JenisIuranID string         `gorm:"type:uuid" json:"jenis_iuran_id"`
	Nominal      float64        `gorm:"type:numeric(15,2)" json:"nominal"`
	Keterangan   *string        `gorm:"type:text" json:"keterangan,omitempty"`
	DibuatOleh   string         `gorm:"type:uuid" json:"dibuat_oleh"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RekapIuran) TableName() string { return "rekap_iuran" }

type RekapPengeluaran struct {
	ID                    string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Tanggal               time.Time      `gorm:"type:date" json:"tanggal"`
	KategoriPengeluaranID string         `gorm:"type:uuid" json:"kategori_pengeluaran_id"`
	Nominal               float64        `gorm:"type:numeric(15,2)" json:"nominal"`
	Keterangan            *string        `gorm:"type:text" json:"keterangan,omitempty"`
	DibuatOleh            string         `gorm:"type:uuid" json:"dibuat_oleh"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RekapPengeluaran) TableName() string { return "rekap_pengeluaran" }
