package user

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9._]{3,50}$`)

func validateUsername(username string) bool {
	return usernamePattern.MatchString(username)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	roleID := r.URL.Query().Get("role_id")
	status := r.URL.Query().Get("status")

	query := h.DB.WithContext(r.Context()).
		Table("users u").
		Select("u.id, u.username, u.nomor_hp, u.jabatan, u.role_id, r.nama as role_nama, u.status, u.last_login, u.created_at, u.updated_at").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL").
		Where("u.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(u.username ILIKE ? OR u.jabatan ILIKE ?)", like, like)
	}
	if roleID != "" {
		query = query.Where("u.role_id = ?", roleID)
	}
	if status != "" {
		query = query.Where("u.status = ?", status == "true")
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("users u").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL").
		Where("u.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(u.username ILIKE ? OR u.jabatan ILIKE ?)", like, like)
	}
	if roleID != "" {
		countQuery = countQuery.Where("u.role_id = ?", roleID)
	}
	if status != "" {
		countQuery = countQuery.Where("u.status = ?", status == "true")
	}
	countQuery.Count(&total)

	switch pg.Sort {
	case "nama":
		query = query.Order("u.username ASC")
	case "-nama":
		query = query.Order("u.username DESC")
	case "created_at":
		query = query.Order("u.created_at DESC")
	case "-created_at":
		query = query.Order("u.created_at ASC")
	default:
		query = query.Order("u.username ASC")
	}

	list := make([]UserResponse, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Pengguna", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var user UserResponse
	err := h.DB.WithContext(r.Context()).
		Table("users u").
		Select("u.id, u.username, u.nomor_hp, u.jabatan, u.role_id, r.nama as role_nama, u.status, u.last_login, u.created_at, u.updated_at").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL").
		Where("u.id = ? AND u.deleted_at IS NULL", id).
		Scan(&user).Error
	if err != nil || user.ID == "" {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	response.Success(w, http.StatusOK, "Detail Pengguna", user)
}

func (h *Handler) isSuperAdminRole(roleID string) bool {
	var nama string
	h.DB.Table("roles").Select("nama").Where("id = ? AND deleted_at IS NULL", roleID).Scan(&nama)
	return nama == "Super Admin"
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	if req.Username == "" {
		errs["username"] = []string{"Username wajib diisi"}
	} else if !validateUsername(req.Username) {
		errs["username"] = []string{"Username hanya boleh berisi huruf kecil, angka, titik, dan underscore (3-50 karakter)"}
	}
	if req.RoleID == "" {
		errs["role_id"] = []string{"Role wajib dipilih"}
	}
	if len(req.Password) < 8 {
		errs["password"] = []string{"Password minimal 8 karakter"}
	}
	if req.Password != req.ConfirmPassword {
		errs["confirm_password"] = []string{"Konfirmasi password tidak cocok"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	if h.isSuperAdminRole(req.RoleID) && middleware.GetRoleName(r) != "Super Admin" {
		response.Error(w, http.StatusForbidden, "Hanya Super Admin yang dapat menetapkan role Super Admin")
		return
	}

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("username = ? AND deleted_at IS NULL", req.Username).Count(&count)
	if count > 0 {
		response.ValidationError(w, map[string][]string{"username": {"Username telah digunakan"}})
		return
	}

	hash, err := helper.HashPassword(req.Password)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	status := true
	if req.Status != nil {
		status = *req.Status
	}

	model := database.User{
		Username: req.Username,
		NomorHP:  req.NomorHP,
		Jabatan:  req.Jabatan,
		RoleID:   req.RoleID,
		Password: hash,
		Status:   status,
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"username": {"Username telah digunakan"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "User Management", "Tambah Pengguna",
		"Menambahkan pengguna: "+req.Username, nil, map[string]string{
			"username": req.Username,
		})
	response.Created(w, "Pengguna berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	if req.Username == "" {
		errs["username"] = []string{"Username wajib diisi"}
	} else if !validateUsername(req.Username) {
		errs["username"] = []string{"Username hanya boleh berisi huruf kecil, angka, titik, dan underscore (3-50 karakter)"}
	}
	if req.RoleID == "" {
		errs["role_id"] = []string{"Role wajib dipilih"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var existing database.User
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	if h.isSuperAdminRole(req.RoleID) && middleware.GetRoleName(r) != "Super Admin" {
		response.Error(w, http.StatusForbidden, "Hanya Super Admin yang dapat menetapkan role Super Admin")
		return
	}

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("username = ? AND id != ? AND deleted_at IS NULL", req.Username, id).Count(&count)
	if count > 0 {
		response.ValidationError(w, map[string][]string{"username": {"Username telah digunakan"}})
		return
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	result := h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"username":   req.Username,
			"nomor_hp":   req.NomorHP,
			"jabatan":    req.Jabatan,
			"role_id":    req.RoleID,
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if helper.IsDuplicateErrorGORM(result.Error) {
			response.ValidationError(w, map[string][]string{"username": {"Username telah digunakan"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "User Management", "Ubah Pengguna",
		"Memperbarui pengguna: "+req.Username, existing, req)
	response.Success(w, http.StatusOK, "Pengguna berhasil diperbarui", nil)
}

func (h *Handler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	var existing database.User
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"status":     req.Status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	statusText := "aktif"
	if !req.Status {
		statusText = "nonaktif"
	}
	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "User Management", "Ubah Status Pengguna",
		"Mengubah status pengguna "+existing.Username+" menjadi "+statusText, existing, map[string]bool{"status": req.Status})
	response.Success(w, http.StatusOK, "Status pengguna berhasil diperbarui", nil)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	if len(req.NewPassword) < 8 {
		errs["new_password"] = []string{"Password baru minimal 8 karakter"}
	}
	if req.NewPassword != req.ConfirmPassword {
		errs["confirm_password"] = []string{"Konfirmasi password tidak cocok"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var existing database.User
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	hash, err := helper.HashPassword(req.NewPassword)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"password":   hash,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	// Cabut sesi aktif pengguna agar password baru langsung berlaku.
	if err := h.DB.WithContext(r.Context()).Model(&database.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL AND deleted_at IS NULL", id).
		Update("revoked_at", time.Now()).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Password direset, tetapi gagal mencabut sesi aktif")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "User Management", "Reset Password",
		"Meriset password pengguna: "+existing.Username, nil, nil)
	response.Success(w, http.StatusOK, "Password berhasil direset", nil)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var existing database.User
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	var role database.Role
	h.DB.WithContext(r.Context()).Where("id = ? AND deleted_at IS NULL", existing.RoleID).First(&role)
	if role.Nama == "Super Admin" {
		var superAdminCount int64
		h.DB.WithContext(r.Context()).
			Table("users u").
			Joins("JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL").
			Where("u.deleted_at IS NULL AND r.nama = ? AND u.id != ?", "Super Admin", id).
			Count(&superAdminCount)
		if superAdminCount == 0 {
			response.Error(w, http.StatusConflict, "Tidak dapat menghapus akun Super Admin terakhir")
			return
		}
	}

	result := h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "User Management", "Hapus Pengguna",
		"Menghapus (soft) pengguna: "+existing.Username, existing, nil)
	response.Success(w, http.StatusOK, "Pengguna berhasil dihapus", nil)
}
