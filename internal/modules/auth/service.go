package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
)

var (
	ErrInvalidCredentials = errors.New("email atau password tidak valid")
	ErrInactiveAccount    = errors.New("akun anda dinonaktifkan")
	ErrInvalidRefresh     = errors.New("refresh token tidak valid")
)

type Service struct {
	Pool        *pgxpool.Pool
	JWTSecret   string
	AccessMin   int
	RefreshDays int
}

func NewService(pool *pgxpool.Pool, secret string, accessMin, refreshDays int) *Service {
	return &Service{Pool: pool, JWTSecret: secret, AccessMin: accessMin, RefreshDays: refreshDays}
}

func (s *Service) findByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	var role string
	err := s.Pool.QueryRow(ctx, `
		SELECT u.id, u.nama, u.email, u.password, u.nomor_hp, u.jabatan, u.role_id,
		       r.nama, u.status, u.last_login, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.email = $1`,
		email,
	).Scan(&u.ID, &u.Nama, &u.Email, &u.Password, &u.NomorHP, &u.Jabatan, &u.RoleID,
		&role, &u.Status, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Role = role
	return &u, nil
}

func (s *Service) loadPermissions(ctx context.Context, roleID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT p.modul || ':' || p.aksi
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1`,
		roleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (s *Service) generateTokens(u *User) (access, refresh string, err error) {
	perms, err := s.loadPermissions(context.Background(), u.RoleID)
	if err != nil {
		return "", "", err
	}
	u.Permission = perms

	accessClaims := middleware.Claims{
		UserID:      u.ID,
		Email:       u.Email,
		Nama:        u.Nama,
		RoleID:      u.RoleID,
		Role:        u.Role,
		Permissions: perms,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.AccessMin) * time.Minute)),
		},
	}
	access, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.JWTSecret))
	if err != nil {
		return "", "", err
	}

	refreshClaims := middleware.Claims{
		UserID: u.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.RefreshDays) * 24 * time.Hour)),
		},
	}
	refresh, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.JWTSecret))
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

func (s *Service) storeRefreshToken(ctx context.Context, userID, refreshToken string) error {
	expiresAt := time.Now().Add(time.Duration(s.RefreshDays) * 24 * time.Hour)
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, helper.HashToken(refreshToken), expiresAt)
	return err
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	u, err := s.findByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !helper.CheckPassword(u.Password, req.Password) {
		return nil, ErrInvalidCredentials
	}
	if !u.Status {
		return nil, ErrInactiveAccount
	}

	access, refresh, err := s.generateTokens(u)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	s.Pool.Exec(ctx, `UPDATE users SET last_login = $1, updated_at = $1 WHERE id = $2`, now, u.ID)
	if err := s.storeRefreshToken(ctx, u.ID, refresh); err != nil {
		return nil, err
	}

	u.LastLogin = &now
	return &LoginResponse{
		Token:        access,
		RefreshToken: refresh,
		User:         *u,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, ErrInvalidRefresh
	}

	hash := helper.HashToken(refreshToken)
	var revokedAt *time.Time
	err = s.Pool.QueryRow(ctx,
		`SELECT revoked_at FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2`,
		claims.UserID, hash).Scan(&revokedAt)
	if err != nil || revokedAt != nil {
		return nil, ErrInvalidRefresh
	}

	var u User
	err = s.Pool.QueryRow(ctx, `
		SELECT u.id, u.nama, u.email, u.password, u.nomor_hp, u.jabatan, u.role_id,
		       r.nama, u.status, u.last_login, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1`,
		claims.UserID,
	).Scan(&u.ID, &u.Nama, &u.Email, &u.Password, &u.NomorHP, &u.Jabatan, &u.RoleID,
		&u.Role, &u.Status, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, ErrInvalidRefresh
	}
	if !u.Status {
		return nil, ErrInactiveAccount
	}

	access, newRefresh, err := s.generateTokens(&u)
	if err != nil {
		return nil, err
	}
	s.Pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND token_hash = $2`,
		claims.UserID, hash)
	s.storeRefreshToken(ctx, u.ID, newRefresh)

	return &LoginResponse{Token: access, RefreshToken: newRefresh, User: u}, nil
}

func (s *Service) Logout(ctx context.Context, userID, refreshToken string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND token_hash = $2`,
		userID, helper.HashToken(refreshToken))
	return err
}
