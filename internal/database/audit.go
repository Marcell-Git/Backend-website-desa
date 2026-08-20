package database

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	ID          string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      *string        `gorm:"type:uuid" json:"user_id"`
	Modul       string         `gorm:"type:varchar(100)" json:"modul"`
	Aktivitas   string         `gorm:"type:varchar(100)" json:"aktivitas"`
	Deskripsi   string         `gorm:"type:text" json:"deskripsi"`
	DataSebelum map[string]any `gorm:"type:jsonb" json:"data_sebelum"`
	DataSesudah map[string]any `gorm:"type:jsonb" json:"data_sesudah"`
	IPAddress   string         `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent   string         `gorm:"type:text" json:"user_agent"`
	CreatedAt   time.Time      `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

func RecordAudit(ctx context.Context, db *gorm.DB, userID *string, modul, aktivitas, deskripsi string, before, after interface{}) {
	var beforeJSON, afterJSON map[string]any

	if before != nil {
		b, err := json.Marshal(before)
		if err == nil {
			_ = json.Unmarshal(b, &beforeJSON)
		}
	}

	if after != nil {
		a, err := json.Marshal(after)
		if err == nil {
			_ = json.Unmarshal(a, &afterJSON)
		}
	}

	entry := AuditLog{
		UserID:      userID,
		Modul:       modul,
		Aktivitas:   aktivitas,
		Deskripsi:   deskripsi,
		DataSebelum: beforeJSON,
		DataSesudah: afterJSON,
		IPAddress:   "",
		UserAgent:   "",
	}
	_ = db.WithContext(ctx).Create(&entry).Error
}
