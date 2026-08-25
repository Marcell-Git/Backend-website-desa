package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxUserName ctxKey = "user_nama"
	CtxRoleID    ctxKey = "role_id"
	CtxRole      ctxKey = "role"
	CtxPerms     ctxKey = "permissions"
)

type Claims struct {
	UserID      string   `json:"uid"`
	Username    string   `json:"username"`
	RoleID      string   `json:"rid"`
	Role        string   `json:"role"`
	Permissions []string `json:"perms"`
	jwt.RegisteredClaims
}

func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"success":false,"message":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !token.Valid {
				http.Error(w, `{"success":false,"message":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if time.Now().After(claims.ExpiresAt.Time) {
				http.Error(w, `{"success":false,"message":"Sesi berakhir, silakan login kembali"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxUserName, claims.Username)
			ctx = context.WithValue(ctx, CtxRoleID, claims.RoleID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			ctx = context.WithValue(ctx, CtxPerms, claims.Permissions)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) string {
	v, _ := r.Context().Value(CtxUserID).(string)
	return v
}

func GetUserName(r *http.Request) string {
	v, _ := r.Context().Value(CtxUserName).(string)
	return v
}

func GetRoleID(r *http.Request) string {
	v, _ := r.Context().Value(CtxRoleID).(string)
	return v
}

func GetRoleName(r *http.Request) string {
	v, _ := r.Context().Value(CtxRole).(string)
	return v
}

func GetPermissions(r *http.Request) []string {
	v, _ := r.Context().Value(CtxPerms).([]string)
	return v
}

func HasPermission(r *http.Request, modul, aksi string) bool {
	key := modul + ":" + aksi
	for _, p := range GetPermissions(r) {
		if p == key {
			return true
		}
	}
	return false
}
