package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	errorsMap := map[string][]string{}
	if req.Email == "" {
		errorsMap["email"] = []string{"Email wajib diisi"}
	}
	if req.Password == "" {
		errorsMap["password"] = []string{"Password wajib diisi"}
	}
	if len(errorsMap) > 0 {
		response.ValidationError(w, errorsMap)
		return
	}

	result, err := h.Service.Login(r.Context(), req)
	if err != nil {
		var uid string
		if u := r.Context().Value(middleware.CtxUserID); u != nil {
			uid, _ = u.(string)
		}
		userID := &uid
		if uid == "" {
			userID = nil
		}
		database.RecordAudit(r.Context(), h.Service.Pool, userID, "Authentication", "Gagal Login", "Percobaan login gagal untuk "+req.Email, nil, nil)

		switch {
		case errors.Is(err, ErrInactiveAccount):
			response.Error(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, "Email atau password tidak valid.")
		default:
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}

	uid := result.User.ID
	database.RecordAudit(r.Context(), h.Service.Pool, &uid, "Authentication", "Login", "Login berhasil sebagai "+result.User.Role, nil, nil)
	response.Success(w, http.StatusOK, "Login berhasil", result)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}
	result, err := h.Service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Refresh token tidak valid")
		return
	}
	response.Success(w, http.StatusOK, "Token berhasil diperbarui", result)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
		h.Service.Logout(r.Context(), middleware.GetUserID(r), req.RefreshToken)
	}
	uid := middleware.GetUserID(r)
	database.RecordAudit(r.Context(), h.Service.Pool, &uid, "Authentication", "Logout", "Pengguna keluar dari sistem", nil, nil)
	response.Success(w, http.StatusOK, "Logout berhasil", nil)
}

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	var u User
	err := h.Service.Pool.QueryRow(r.Context(), `
		SELECT u.id, u.nama, u.email, u.nomor_hp, u.jabatan, u.role_id, r.nama,
		       u.status, u.last_login, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1`,
		middleware.GetUserID(r),
	).Scan(&u.ID, &u.Nama, &u.Email, &u.NomorHP, &u.Jabatan, &u.RoleID, &u.Role,
		&u.Status, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}
	perms, _ := h.Service.loadPermissions(r.Context(), u.RoleID)
	u.Permission = perms

	response.Success(w, http.StatusOK, "Profil pengguna", u)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errorsMap := map[string][]string{}
	if req.OldPassword == "" {
		errorsMap["old_password"] = []string{"Password lama wajib diisi"}
	}
	if len(req.NewPassword) < 8 {
		errorsMap["new_password"] = []string{"Password baru minimal 8 karakter"}
	}
	if req.NewPassword == req.OldPassword {
		errorsMap["new_password"] = append(errorsMap["new_password"], "Password baru tidak boleh sama dengan password lama")
	}
	if len(errorsMap) > 0 {
		response.ValidationError(w, errorsMap)
		return
	}

	var hashed string
	err := h.Service.Pool.QueryRow(r.Context(),
		`SELECT password FROM users WHERE id = $1`, middleware.GetUserID(r)).Scan(&hashed)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}
	if !helper.CheckPassword(hashed, req.OldPassword) {
		response.ValidationError(w, map[string][]string{"old_password": {"Password lama tidak sesuai"}})
		return
	}

	newHash, err := helper.HashPassword(req.NewPassword)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	_, err = h.Service.Pool.Exec(r.Context(),
		`UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2`, newHash, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	uid := middleware.GetUserID(r)
	database.RecordAudit(r.Context(), h.Service.Pool, &uid, "Authentication", "Perubahan Password", "Pengguna mengubah password sendiri", nil, nil)
	response.Success(w, http.StatusOK, "Password berhasil diperbarui", nil)
}
