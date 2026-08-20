package user

import "time"

type UserResponse struct {
	ID        string     `json:"id"`
	Nama      string     `json:"nama"`
	Email     string     `json:"email"`
	NomorHP   *string    `json:"nomor_hp,omitempty"`
	Jabatan   *string    `json:"jabatan,omitempty"`
	RoleID    string     `json:"role_id"`
	RoleNama  string     `json:"role_nama"`
	Status    bool       `json:"status"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CreateUserRequest struct {
	Nama            string  `json:"nama"`
	Email           string  `json:"email"`
	NomorHP         *string `json:"nomor_hp,omitempty"`
	Jabatan         *string `json:"jabatan,omitempty"`
	RoleID          string  `json:"role_id"`
	Password        string  `json:"password"`
	ConfirmPassword string  `json:"confirm_password"`
	Status          *bool   `json:"status,omitempty"`
}

type UpdateUserRequest struct {
	Nama    string  `json:"nama"`
	Email   string  `json:"email"`
	NomorHP *string `json:"nomor_hp,omitempty"`
	Jabatan *string `json:"jabatan,omitempty"`
	RoleID  string  `json:"role_id"`
	Status  *bool   `json:"status,omitempty"`
}

type ResetPasswordRequest struct {
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type UpdateStatusRequest struct {
	Status bool `json:"status"`
}

type RoleResponse struct {
	ID              string    `json:"id"`
	Nama            string    `json:"nama"`
	Deskripsi       string    `json:"deskripsi"`
	Status          bool      `json:"status"`
	UserCount       int64     `json:"user_count"`
	PermissionCount int       `json:"permission_count"`
	CreatedAt       time.Time `json:"created_at"`
}

type RoleDetailResponse struct {
	ID          string             `json:"id"`
	Nama        string             `json:"nama"`
	Deskripsi   string             `json:"deskripsi"`
	Status      bool               `json:"status"`
	Permissions []PermissionItem   `json:"permissions"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type PermissionItem struct {
	ID    string `json:"id"`
	Nama  string `json:"nama"`
	Modul string `json:"modul"`
	Aksi  string `json:"aksi"`
}

type CreateRoleRequest struct {
	Nama      string `json:"nama"`
	Deskripsi string `json:"deskripsi"`
	Status    *bool  `json:"status,omitempty"`
}

type UpdateRoleRequest struct {
	Nama      string `json:"nama"`
	Deskripsi string `json:"deskripsi"`
	Status    *bool  `json:"status,omitempty"`
}

type UpdateRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}
