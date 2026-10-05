package database

import (
	"time"
)

type WebsiteSettings struct {
	ID             string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NamaWebsite    string     `gorm:"type:varchar(200);default:'SIM RT Tegalyoso 26'" json:"nama_website"`
	NamaRT         string     `gorm:"type:varchar(100);default:'RT 26'" json:"nama_rt"`
	Desa           string     `gorm:"type:varchar(100);default:'Tegalyoso'" json:"desa"`
	Kecamatan      string     `gorm:"type:varchar(100);default:'Klaten Selatan'" json:"kecamatan"`
	Kabupaten      string     `gorm:"type:varchar(100);default:'Klaten'" json:"kabupaten"`
	Provinsi       string     `gorm:"type:varchar(100);default:'Jawa Tengah'" json:"provinsi"`
	Telepon        string     `gorm:"type:varchar(20)" json:"telepon"`
	Email          string     `gorm:"type:varchar(150)" json:"email"`
	Alamat         string     `gorm:"type:text" json:"alamat"`
	MapsLat        *float64   `gorm:"type:double precision;default:-7.7122778" json:"maps_lat"`
	MapsLng        *float64   `gorm:"type:double precision;default:110.5879722" json:"maps_lng"`
	Facebook       string     `gorm:"type:varchar(255)" json:"facebook"`
	Instagram      string     `gorm:"type:varchar(255)" json:"instagram"`
	Whatsapp       string     `gorm:"type:varchar(255)" json:"whatsapp"`
	Youtube        string     `gorm:"type:varchar(255)" json:"youtube"`
	Tiktok         string     `gorm:"type:varchar(255)" json:"tiktok"`
	LogoPath       string     `gorm:"type:varchar(255)" json:"logo_path"`
	Copyright      string     `gorm:"type:varchar(200);default:'© 2026 RT 26 Kel. Tegalyoso'" json:"copyright"`
	Visi           string     `gorm:"type:text" json:"visi"`
	Misi           StringList `gorm:"type:jsonb;default:'[]'" json:"misi"`
	ShowPengumuman bool       `gorm:"default:true" json:"show_pengumuman"`
	ShowLaporan    bool       `gorm:"default:true" json:"show_laporan"`
	ShowPengaduan  bool       `gorm:"default:true" json:"show_pengaduan"`
	ShowVisiMisi   bool       `gorm:"default:true" json:"show_visi_misi"`
	ShowPeraturan  bool       `gorm:"default:true" json:"show_peraturan"`
	ShowStatistik  bool       `gorm:"default:true" json:"show_statistik"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (WebsiteSettings) TableName() string { return "website_settings" }
