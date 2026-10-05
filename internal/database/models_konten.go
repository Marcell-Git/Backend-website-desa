package database

import (
	"time"

	"gorm.io/gorm"
)

type Pengumuman struct {
	ID               string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Judul            string         `gorm:"type:varchar(200);not null" json:"judul"`
	Isi              string         `gorm:"type:text;not null" json:"isi"`
	TanggalPublikasi time.Time      `gorm:"type:date;not null" json:"tanggal_publikasi"`
	Status           string         `gorm:"type:varchar(20);not null;default:draft" json:"status"`
	DibuatOleh       string         `gorm:"type:uuid;not null" json:"dibuat_oleh"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Pengumuman) TableName() string { return "pengumuman" }

type Pengurus struct {
	ID          string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NamaLengkap string         `gorm:"type:varchar(150);not null" json:"nama_lengkap"`
	Jabatan     string         `gorm:"type:varchar(100);not null" json:"jabatan"`
	Foto        *string        `gorm:"type:varchar(255)" json:"foto,omitempty"`
	Urutan      int            `gorm:"default:0" json:"urutan"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Pengurus) TableName() string { return "pengurus" }

type BeritaKegiatan struct {
	ID         string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Kategori   string         `gorm:"type:varchar(50);not null" json:"kategori"`
	WarnaLabel string         `gorm:"type:varchar(7);not null;default:'#047857'" json:"warna_label"`
	Judul      string         `gorm:"type:varchar(250);not null" json:"judul"`
	Tanggal    time.Time      `gorm:"type:date;not null" json:"tanggal"`
	Jam        string         `gorm:"type:varchar(5);not null" json:"jam"`
	Lokasi     string         `gorm:"type:varchar(250);not null" json:"lokasi"`
	Keterangan string         `gorm:"type:text" json:"keterangan"`
	DibuatOleh string         `gorm:"type:uuid;not null" json:"dibuat_oleh"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (BeritaKegiatan) TableName() string { return "berita_kegiatan" }

type Inventaris struct {
	ID                string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NamaBarang        string         `gorm:"type:varchar(250);not null" json:"nama_barang"`
	Kategori          string         `gorm:"type:varchar(50);not null" json:"kategori"`
	WarnaLabel        string         `gorm:"type:varchar(7);not null;default:'#047857'" json:"warna_label"`
	JumlahBarang      int            `gorm:"not null;default:1" json:"jumlah_barang"`
	KondisiBarang     string         `gorm:"type:varchar(30);not null;default:'baik'" json:"kondisi_barang"`
	LokasiPenyimpanan string         `gorm:"type:varchar(250)" json:"lokasi_penyimpanan"`
	PenanggungJawab   string         `gorm:"type:varchar(150)" json:"penanggung_jawab"`
	PeriodePerolehan  string         `gorm:"type:varchar(7)" json:"periode_perolehan"`
	Foto              *string        `gorm:"type:varchar(255)" json:"foto,omitempty"`
	Keterangan        string         `gorm:"type:text" json:"keterangan"`
	DibuatOleh        string         `gorm:"type:uuid;not null" json:"dibuat_oleh"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Inventaris) TableName() string { return "inventaris" }

type Peraturan struct {
	ID         string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Judul      string         `gorm:"type:varchar(200);not null" json:"judul"`
	FilePDF    string         `gorm:"type:varchar(255);not null" json:"file_pdf"`
	DibuatOleh string         `gorm:"type:uuid;not null" json:"dibuat_oleh"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Peraturan) TableName() string { return "peraturan" }
