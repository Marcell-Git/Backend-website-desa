package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

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

	req.Username = strings.TrimSpace(req.Username)
	errorsMap := map[string][]string{}
	if req.Username == "" {
		errorsMap["username"] = []string{"Username wajib diisi"}
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
		database.RecordAudit(r.Context(), h.Service.DB, userID, "Authentication", "Gagal Login", "Percobaan login gagal untuk "+req.Username, nil, nil)

		switch {
		case errors.Is(err, ErrInactiveAccount):
			response.Error(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, "Username atau password tidak valid.")
		default:
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}

	uid := result.User.ID
	database.RecordAudit(r.Context(), h.Service.DB, &uid, "Authentication", "Login", "Login berhasil sebagai "+result.User.Role, nil, nil)
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
	database.RecordAudit(r.Context(), h.Service.DB, &uid, "Authentication", "Logout", "Pengguna keluar dari sistem", nil, nil)
	response.Success(w, http.StatusOK, "Logout berhasil", nil)
}

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	var dbUser database.User
	err := h.Service.DB.WithContext(r.Context()).
		Where("id = ?", middleware.GetUserID(r)).
		First(&dbUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var u User
	u.ID = dbUser.ID
	u.Username = dbUser.Username
	u.NomorHP = dbUser.NomorHP
	u.Jabatan = dbUser.Jabatan
	u.RoleID = dbUser.RoleID
	u.Status = dbUser.Status
	u.LastLogin = dbUser.LastLogin
	u.CreatedAt = dbUser.CreatedAt
	u.UpdatedAt = dbUser.UpdatedAt

	var roleName string
	h.Service.DB.WithContext(r.Context()).Table("roles").Select("nama").Where("id = ?", dbUser.RoleID).Scan(&roleName)
	u.Role = roleName

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
	err := h.Service.DB.WithContext(r.Context()).
		Table("users").Select("password").Where("id = ?", middleware.GetUserID(r)).Scan(&hashed).Error
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
	err = h.Service.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("id = ?", middleware.GetUserID(r)).
		Update("password", newHash).Error
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	uid := middleware.GetUserID(r)
	database.RecordAudit(r.Context(), h.Service.DB, &uid, "Authentication", "Perubahan Password", "Pengguna mengubah password sendiri", nil, nil)
	response.Success(w, http.StatusOK, "Password berhasil diperbarui", nil)
}
