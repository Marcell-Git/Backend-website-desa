package database

import (
	"time"

	"gorm.io/gorm"
)

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
	Nama      string         `gorm:"type:varchar(150)" json:"nama"`
	Email     string         `gorm:"type:varchar(150);uniqueIndex" json:"email"`
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
	ID                 string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NomorKK            string         `gorm:"type:varchar(16);uniqueIndex" json:"nomor_kk"`
	KepalaKeluargaID   *string        `gorm:"type:uuid" json:"kepala_keluarga_id,omitempty"`
	NomorHP            *string        `gorm:"type:varchar(20)" json:"nomor_hp,omitempty"`
	Alamat             string         `gorm:"type:text" json:"alamat"`
	RT                 string         `gorm:"type:varchar(5)" json:"rt"`
	RW                 string         `gorm:"type:varchar(5)" json:"rw"`
	Desa               string         `gorm:"type:varchar(100)" json:"desa"`
	Kecamatan          string         `gorm:"type:varchar(100)" json:"kecamatan"`
	Kabupaten          string         `gorm:"type:varchar(100)" json:"kabupaten"`
	Provinsi           string         `gorm:"type:varchar(100)" json:"provinsi"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KartuKeluarga) TableName() string { return "kartu_keluarga" }

type AnggotaKeluarga struct {
	ID                string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	KartuKeluargaID   string         `gorm:"type:uuid" json:"kartu_keluarga_id"`
	NIK               string         `gorm:"type:varchar(16);uniqueIndex" json:"nik"`
	NamaLengkap       string         `gorm:"type:varchar(150)" json:"nama_lengkap"`
	TempatLahir       string         `gorm:"type:varchar(100)" json:"tempat_lahir"`
	TanggalLahir      string         `gorm:"type:date" json:"tanggal_lahir"`
	JenisKelamin      string         `gorm:"type:varchar(1);check:jenis_kelamin IN ('L','P')" json:"jenis_kelamin"`
	Agama             string         `gorm:"type:varchar(30)" json:"agama"`
	Pekerjaan         *string        `gorm:"type:varchar(100)" json:"pekerjaan,omitempty"`
	NomorHP           *string        `gorm:"type:varchar(20)" json:"nomor_hp,omitempty"`
	HubunganKeluarga  string         `gorm:"type:varchar(50)" json:"hubungan_keluarga"`
	Status            bool           `gorm:"default:true" json:"status"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AnggotaKeluarga) TableName() string { return "anggota_keluarga" }
