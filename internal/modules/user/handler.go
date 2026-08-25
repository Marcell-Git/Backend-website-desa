package user

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9._]{3,50}$`)

func validateUsername(username string) bool {
	return usernamePattern.MatchString(username)
}

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

func userIDPtr(r *http.Request) *string {
	uid := middleware.GetUserID(r)
	return &uid
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
			"username":  req.Username,
			"nomor_hp":  req.NomorHP,
			"jabatan":   req.Jabatan,
			"role_id":   req.RoleID,
			"status":    status,
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

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))

	query := h.DB.WithContext(r.Context()).
		Table("roles r").
		Select("r.id, r.nama, r.deskripsi, r.status, r.created_at, r.updated_at, COUNT(DISTINCT u.id) as user_count, COUNT(DISTINCT rp.permission_id) as permission_count").
		Joins("LEFT JOIN users u ON u.role_id = r.id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN role_permissions rp ON rp.role_id = r.id").
		Where("r.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(r.nama ILIKE ? OR r.deskripsi ILIKE ?)", like, like)
	}

	var total int64
	countQuery := h.DB.WithContext(r.Context()).
		Table("roles r").
		Where("r.deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("(r.nama ILIKE ? OR r.deskripsi ILIKE ?)", like, like)
	}
	countQuery.Count(&total)

	query = query.Group("r.id")

	switch pg.Sort {
	case "nama":
		query = query.Order("r.nama ASC")
	case "-nama":
		query = query.Order("r.nama DESC")
	case "created_at":
		query = query.Order("r.created_at DESC")
	case "-created_at":
		query = query.Order("r.created_at ASC")
	default:
		query = query.Order("r.nama ASC")
	}

	list := make([]RoleResponse, 0)
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&list).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.SuccessPaginated(w, "Daftar Role", list, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama role wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.Role{}).
		Where("nama = ? AND deleted_at IS NULL", req.Nama).Count(&count)
	if count > 0 {
		response.ValidationError(w, map[string][]string{"nama": {"Nama role sudah digunakan"}})
		return
	}

	status := true
	if req.Status != nil {
		status = *req.Status
	}

	model := database.Role{
		Nama:      req.Nama,
		Deskripsi: req.Deskripsi,
		Status:    status,
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"nama": {"Nama role sudah digunakan"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Role Management", "Tambah Role",
		"Menambahkan role: "+req.Nama, nil, map[string]string{"nama": req.Nama})
	response.Created(w, "Role berhasil ditambahkan", map[string]string{"id": model.ID})
}

func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var role database.Role
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&role).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}

	var perms []PermissionItem
	h.DB.WithContext(r.Context()).
		Table("permissions p").
		Select("p.id, p.nama, p.modul, p.aksi").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Where("rp.role_id = ?", id).
		Order("p.modul ASC, p.aksi ASC").
		Scan(&perms)
	if perms == nil {
		perms = []PermissionItem{}
	}

	resp := RoleDetailResponse{
		ID:          role.ID,
		Nama:        role.Nama,
		Deskripsi:   role.Deskripsi,
		Status:      role.Status,
		Permissions: perms,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
	response.Success(w, http.StatusOK, "Detail Role", resp)
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama role wajib diisi"}
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	var existing database.Role
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.Role{}).
		Where("nama = ? AND id != ? AND deleted_at IS NULL", req.Nama, id).Count(&count)
	if count > 0 {
		response.ValidationError(w, map[string][]string{"nama": {"Nama role sudah digunakan"}})
		return
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	result := h.DB.WithContext(r.Context()).Model(&database.Role{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"nama":       req.Nama,
			"deskripsi":  req.Deskripsi,
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if helper.IsDuplicateErrorGORM(result.Error) {
			response.ValidationError(w, map[string][]string{"nama": {"Nama role sudah digunakan"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Role Management", "Ubah Role",
		"Memperbarui role: "+req.Nama, existing, req)
	response.Success(w, http.StatusOK, "Role berhasil diperbarui", nil)
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var existing database.Role
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}

	var userCount int64
	h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("role_id = ? AND deleted_at IS NULL", id).Count(&userCount)
	if userCount > 0 {
		response.Error(w, http.StatusConflict, "Role tidak dapat dihapus karena masih digunakan oleh pengguna")
		return
	}

	result := h.DB.WithContext(r.Context()).Model(&database.Role{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}

	h.DB.WithContext(r.Context()).Where("role_id = ?", id).
		Delete(&database.RolePermission{})

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Role Management", "Hapus Role",
		"Menghapus (soft) role: "+existing.Nama, existing, nil)
	response.Success(w, http.StatusOK, "Role berhasil dihapus", nil)
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	var perms []PermissionItem
	if err := h.DB.WithContext(r.Context()).
		Table("permissions").
		Select("id, nama, modul, aksi").
		Order("modul ASC, aksi ASC").
		Scan(&perms).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	grouped := map[string][]PermissionItem{}
	for _, p := range perms {
		grouped[p.Modul] = append(grouped[p.Modul], p)
	}

	response.Success(w, http.StatusOK, "Daftar Permissions", grouped)
}

func (h *Handler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var role database.Role
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&role).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}

	var permIDs []string
	h.DB.WithContext(r.Context()).
		Table("role_permissions").
		Select("permission_id").
		Where("role_id = ?", id).
		Pluck("permission_id", &permIDs)

	response.Success(w, http.StatusOK, "Permission Role", map[string]interface{}{
		"role_id":       id,
		"permission_ids": permIDs,
	})
}

func (h *Handler) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateRolePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	var role database.Role
	if err := h.DB.WithContext(r.Context()).
		Where("id = ? AND deleted_at IS NULL", id).First(&role).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Role tidak ditemukan")
		return
	}

	h.DB.WithContext(r.Context()).Where("role_id = ?", id).
		Delete(&database.RolePermission{})

	for _, permID := range req.PermissionIDs {
		rp := database.RolePermission{RoleID: id, PermissionID: permID}
		h.DB.WithContext(r.Context()).Create(&rp)
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "Role Management", "Ubah Permission",
		"Memperbarui permission role: "+role.Nama, nil, map[string]interface{}{
			"role_id":        id,
			"permission_ids": req.PermissionIDs,
		})
	response.Success(w, http.StatusOK, "Permission berhasil diperbarui", nil)
}
