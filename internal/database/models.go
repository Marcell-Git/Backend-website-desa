package database

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StringList persists a list of strings as a PostgreSQL JSONB value.
type StringList []string

func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("StringList: unsupported Scan type %T", value)
	}
	return json.Unmarshal(bytes, s)
}

func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

type Role struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Nama      string         `gorm:"type:varchar(100);uniqueIndex" json:"nama"`
	Deskripsi string         `gorm:"type:text" json:"deskripsi"`
	Status    bool           `gorm:"default:true" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Nama      string    `gorm:"type:varchar(100);uniqueIndex" json:"nama"`
	Modul     string    `gorm:"type:varchar(100);uniqueIndex:idx_modul_aksi" json:"modul"`
	Aksi      string    `gorm:"type:varchar(50);uniqueIndex:idx_modul_aksi" json:"aksi"`
	CreatedAt time.Time `json:"created_at"`
}

func (Permission) TableName() string { return "permissions" }

type RolePermission struct {
	RoleID       string `gorm:"type:uuid;primaryKey"`
	PermissionID string `gorm:"type:uuid;primaryKey"`
}

func (RolePermission) TableName() string { return "role_permissions" }

type User struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Username  string         `gorm:"type:varchar(50);uniqueIndex" json:"username"`
	Password  string         `gorm:"type:varchar(255)" json:"-"`
	NomorHP   *string        `gorm:"type:varchar(20)" json:"nomor_hp,omitempty"`
	Jabatan   *string        `gorm:"type:varchar(100)" json:"jabatan,omitempty"`
	RoleID    string         `gorm:"type:uuid" json:"role_id"`
	Status    bool           `gorm:"default:true" json:"status"`
	LastLogin *time.Time     `json:"last_login,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

type RefreshToken struct {
	ID        string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    string         `gorm:"type:uuid" json:"user_id"`
	TokenHash string         `gorm:"type:varchar(255)" json:"-"`
	ExpiresAt time.Time      `json:"expires_at"`
	RevokedAt *time.Time     `json:"revoked_at"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

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

type AnakKos struct {
	ID           string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NIK          string         `gorm:"type:varchar(16);uniqueIndex" json:"nik"`
	NamaLengkap  string         `gorm:"type:varchar(150)" json:"nama_lengkap"`
	JenisKelamin string         `gorm:"type:varchar(1);check:jenis_kelamin IN ('L','P')" json:"jenis_kelamin"`
	TanggalLahir string         `gorm:"type:date" json:"tanggal_lahir"`
	Agama        string         `gorm:"type:varchar(30)" json:"agama"`
	Pekerjaan    *string        `gorm:"type:varchar(100)" json:"pekerjaan,omitempty"`
	NomorHP      *string        `gorm:"type:varchar(20)" json:"nomor_hp,omitempty"`
	AlamatAsal   string         `gorm:"type:text" json:"alamat_asal"`
	Status       bool           `gorm:"default:true" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AnakKos) TableName() string { return "anak_kos" }

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
	LogoPath       string     `gorm:"type:varchar(255)" json:"logo_path"`
	Copyright      string     `gorm:"type:varchar(200);default:'© 2026 RT 26 Kel. Tegalyoso'" json:"copyright"`
	Visi           string     `gorm:"type:text" json:"visi"`
	Misi           StringList `gorm:"type:jsonb;default:'[]'" json:"misi"`
	ShowPengumuman bool       `gorm:"default:true" json:"show_pengumuman"`
	ShowLaporan    bool       `gorm:"default:true" json:"show_laporan"`
	ShowPengaduan  bool       `gorm:"default:true" json:"show_pengaduan"`
	ShowVisiMisi   bool       `gorm:"default:true" json:"show_visi_misi"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (WebsiteSettings) TableName() string { return "website_settings" }

type Backup struct {
	ID         string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Filename   string         `gorm:"type:varchar(255);uniqueIndex" json:"nama_file"`
	FileSize   int64          `json:"ukuran_file"`
	DibuatOleh *string        `gorm:"type:uuid" json:"dibuat_oleh,omitempty"`
	CreatedAt  time.Time      `json:"tanggal_backup"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Backup) TableName() string { return "backups" }
