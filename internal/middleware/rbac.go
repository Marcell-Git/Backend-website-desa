package middleware

import (
	"net/http"

	"rt26/backend/internal/pkg/response"
)

func RequirePermission(modul, aksi string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasPermission(r, modul, aksi) {
				response.Error(w, http.StatusForbidden, "Anda tidak memiliki hak akses untuk tindakan ini")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
