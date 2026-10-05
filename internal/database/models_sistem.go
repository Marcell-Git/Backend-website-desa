package database

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StringList persists a list of strings as a PostgreSQL JSONB value.
// Dipakai oleh kolom website_settings.misi.
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

type Backup struct {
	ID         string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Filename   string         `gorm:"type:varchar(255);uniqueIndex" json:"nama_file"`
	FileSize   int64          `json:"ukuran_file"`
	DibuatOleh *string        `gorm:"type:uuid" json:"dibuat_oleh,omitempty"`
	CreatedAt  time.Time      `json:"tanggal_backup"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Backup) TableName() string { return "backups" }
