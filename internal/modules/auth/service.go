package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
)

var (
	ErrInvalidCredentials = errors.New("email atau password tidak valid")
	ErrInactiveAccount    = errors.New("akun anda dinonaktifkan")
	ErrInvalidRefresh     = errors.New("refresh token tidak valid")
)

type Service struct {
	DB          *gorm.DB
	JWTSecret   string
	AccessMin   int
	RefreshDays int
}

func NewService(db *gorm.DB, secret string, accessMin, refreshDays int) *Service {
	return &Service{DB: db, JWTSecret: secret, AccessMin: accessMin, RefreshDays: refreshDays}
}

func (s *Service) findByEmail(ctx context.Context, email string) (*User, error) {
	var dbUser database.User
	err := s.DB.WithContext(ctx).
		Where("email = ?", email).
		First(&dbUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}

	var u User
	u.ID = dbUser.ID
	u.Nama = dbUser.Nama
	u.Email = dbUser.Email
	u.Password = dbUser.Password
	u.NomorHP = dbUser.NomorHP
	u.Jabatan = dbUser.Jabatan
	u.RoleID = dbUser.RoleID
	u.Status = dbUser.Status
	u.LastLogin = dbUser.LastLogin
	u.CreatedAt = dbUser.CreatedAt
	u.UpdatedAt = dbUser.UpdatedAt

	var roleName string
	s.DB.WithContext(ctx).Table("roles").Select("nama").Where("id = ?", dbUser.RoleID).Scan(&roleName)
	u.Role = roleName

	return &u, nil
}

func (s *Service) loadPermissions(ctx context.Context, roleID string) ([]string, error) {
	var perms []string
	err := s.DB.WithContext(ctx).
		Table("role_permissions rp").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("rp.role_id = ?", roleID).
		Select("p.modul || ':' || p.aksi").
		Pluck("p.modul || ':' || p.aksi", &perms).Error
	return perms, err
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

	refreshJTI, err := helper.RandomTokenID()
	if err != nil {
		return "", "", err
	}
	refreshClaims := middleware.Claims{
		UserID: u.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        refreshJTI,
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
	token := database.RefreshToken{
		UserID:    userID,
		TokenHash: helper.HashToken(refreshToken),
		ExpiresAt: expiresAt,
	}
	return s.DB.WithContext(ctx).Create(&token).Error
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
	s.DB.WithContext(ctx).Model(&database.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"last_login": now,
		"updated_at": now,
	})
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
	var rt database.RefreshToken
	err = s.DB.WithContext(ctx).
		Where("user_id = ? AND token_hash = ? AND revoked_at IS NULL", claims.UserID, hash).
		First(&rt).Error
	if err != nil {
		return nil, ErrInvalidRefresh
	}

	var dbUser database.User
	err = s.DB.WithContext(ctx).Where("id = ?", claims.UserID).First(&dbUser).Error
	if err != nil {
		return nil, ErrInvalidRefresh
	}
	if !dbUser.Status {
		return nil, ErrInactiveAccount
	}

	var u User
	u.ID = dbUser.ID
	u.Nama = dbUser.Nama
	u.Email = dbUser.Email
	u.Password = dbUser.Password
	u.NomorHP = dbUser.NomorHP
	u.Jabatan = dbUser.Jabatan
	u.RoleID = dbUser.RoleID
	u.Status = dbUser.Status
	u.LastLogin = dbUser.LastLogin
	u.CreatedAt = dbUser.CreatedAt
	u.UpdatedAt = dbUser.UpdatedAt
	var roleName string
	s.DB.WithContext(ctx).Table("roles").Select("nama").Where("id = ?", dbUser.RoleID).Scan(&roleName)
	u.Role = roleName

	access, newRefresh, err := s.generateTokens(&u)
	if err != nil {
		return nil, err
	}
	s.DB.WithContext(ctx).Model(&database.RefreshToken{}).
		Where("user_id = ? AND token_hash = ?", claims.UserID, hash).
		Update("revoked_at", time.Now())
	s.storeRefreshToken(ctx, u.ID, newRefresh)

	return &LoginResponse{Token: access, RefreshToken: newRefresh, User: u}, nil
}

func (s *Service) Logout(ctx context.Context, userID, refreshToken string) error {
	return s.DB.WithContext(ctx).Model(&database.RefreshToken{}).
		Where("user_id = ? AND token_hash = ?", userID, helper.HashToken(refreshToken)).
		Update("revoked_at", time.Now()).Error
}
