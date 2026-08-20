package auth

import "time"

type User struct {
	ID         string     `json:"id"`
	Nama       string     `json:"nama"`
	Email      string     `json:"email"`
	Password   string     `json:"-"`
	NomorHP    *string    `json:"nomor_hp,omitempty"`
	Jabatan    *string    `json:"jabatan,omitempty"`
	RoleID     string     `json:"role_id"`
	Role       string     `json:"role"`
	Status     bool       `json:"status"`
	LastLogin  *time.Time `json:"last_login,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Permission []string   `json:"permissions,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
