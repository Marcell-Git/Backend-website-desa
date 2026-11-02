package database

import (
	"time"

	"gorm.io/gorm"
)

type JenisIuran struct {
	ID           string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Nama         string         `gorm:"type:varchar(100);uniqueIndex" json:"nama"`
	Deskripsi    *string        `gorm:"type:text" json:"deskripsi,omitempty"`
	Nominal      float64        `gorm:"type:numeric(15,2);not null;default:0" json:"nominal"`
	TipePeriode  string         `gorm:"type:varchar(20);not null;default:'selamanya'" json:"tipe_periode"`
	BulanMulai   *time.Time     `gorm:"type:date" json:"bulan_mulai,omitempty"`
	BulanSelesai *time.Time     `gorm:"type:date" json:"bulan_selesai,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
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

// PengaturanKas menyimpan nominal kas bulanan yang ditagihkan ke setiap KK.
// Tabel ini hanya berisi satu baris.
type PengaturanKas struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NominalBulanan float64   `gorm:"type:numeric(15,2);not null;default:0" json:"nominal_bulanan"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (PengaturanKas) TableName() string { return "pengaturan_kas" }

// PembayaranBulanan mencatat konfirmasi pembayaran kas/iuran satu KK untuk satu
// bulan. Setiap konfirmasi juga membuat entri di rekap_kas atau rekap_iuran.
type PembayaranBulanan struct {
	ID              string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	KartuKeluargaID string         `gorm:"type:uuid;not null;index" json:"kartu_keluarga_id"`
	Jenis           string         `gorm:"type:varchar(10);not null" json:"jenis"`
	JenisIuranID    *string        `gorm:"type:uuid" json:"jenis_iuran_id,omitempty"`
	Periode         time.Time      `gorm:"type:date;not null;index" json:"periode"`
	Nominal         float64        `gorm:"type:numeric(15,2)" json:"nominal"`
	NominalTagihan  float64        `gorm:"type:numeric(15,2);not null;default:0" json:"nominal_tagihan"`
	TanggalBayar    time.Time      `gorm:"type:date" json:"tanggal_bayar"`
	RekapKasID      *string        `gorm:"type:uuid;index" json:"rekap_kas_id,omitempty"`
	RekapIuranID    *string        `gorm:"type:uuid;index" json:"rekap_iuran_id,omitempty"`
	DibuatOleh      string         `gorm:"type:uuid" json:"dibuat_oleh"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PembayaranBulanan) TableName() string { return "pembayaran_bulanan" }
