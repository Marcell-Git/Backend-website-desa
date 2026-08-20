package auditlog

import "time"

type AuditLogResponse struct {
	ID          string         `json:"id"`
	UserID      *string        `json:"user_id"`
	NamaUser    string         `json:"nama_user,omitempty"`
	RoleUser    string         `json:"role_user,omitempty"`
	Modul       string         `json:"modul"`
	Aktivitas   string         `json:"aktivitas"`
	Deskripsi   string         `json:"deskripsi"`
	DataSebelum map[string]any `json:"data_sebelum,omitempty"`
	DataSesudah map[string]any `json:"data_sesudah,omitempty"`
	IPAddress   string         `json:"ip_address"`
	UserAgent   string         `json:"user_agent"`
	CreatedAt   time.Time      `json:"created_at"`
}
