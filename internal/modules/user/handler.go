package user

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

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
		Select("u.id, u.nama, u.email, u.nomor_hp, u.jabatan, u.role_id, r.nama as role_nama, u.status, u.last_login, u.created_at, u.updated_at").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL").
		Where("u.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(u.nama ILIKE ? OR u.email ILIKE ? OR u.jabatan ILIKE ?)", like, like, like)
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
		countQuery = countQuery.Where("(u.nama ILIKE ? OR u.email ILIKE ? OR u.jabatan ILIKE ?)", like, like, like)
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
		query = query.Order("u.nama ASC")
	case "-nama":
		query = query.Order("u.nama DESC")
	case "created_at":
		query = query.Order("u.created_at DESC")
	case "-created_at":
		query = query.Order("u.created_at ASC")
	default:
		query = query.Order("u.nama ASC")
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
		Select("u.id, u.nama, u.email, u.nomor_hp, u.jabatan, u.role_id, r.nama as role_nama, u.status, u.last_login, u.created_at, u.updated_at").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL").
		Where("u.id = ? AND u.deleted_at IS NULL", id).
		Scan(&user).Error
	if err != nil || user.ID == "" {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	response.Success(w, http.StatusOK, "Detail Pengguna", user)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Format request tidak valid")
		return
	}

	errs := map[string][]string{}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama wajib diisi"}
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		errs["email"] = []string{"Email wajib diisi"}
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

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("email = ? AND deleted_at IS NULL", req.Email).Count(&count)
	if count > 0 {
		response.ValidationError(w, map[string][]string{"email": {"Email telah digunakan"}})
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
		Nama:    req.Nama,
		Email:   req.Email,
		NomorHP: req.NomorHP,
		Jabatan: req.Jabatan,
		RoleID:  req.RoleID,
		Password: hash,
		Status:  status,
	}
	if err := h.DB.WithContext(r.Context()).Create(&model).Error; err != nil {
		if helper.IsDuplicateErrorGORM(err) {
			response.ValidationError(w, map[string][]string{"email": {"Email telah digunakan"}})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	database.RecordAudit(r.Context(), h.DB, userIDPtr(r), "User Management", "Tambah Pengguna",
		"Menambahkan pengguna: "+req.Nama, nil, map[string]string{
			"nama":  req.Nama,
			"email": req.Email,
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
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		errs["nama"] = []string{"Nama wajib diisi"}
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		errs["email"] = []string{"Email wajib diisi"}
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

	var count int64
	h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("email = ? AND id != ? AND deleted_at IS NULL", req.Email, id).Count(&count)
	if count > 0 {
		response.ValidationError(w, map[string][]string{"email": {"Email telah digunakan"}})
		return
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	result := h.DB.WithContext(r.Context()).Model(&database.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"nama":      req.Nama,
			"email":     req.Email,
			"nomor_hp":  req.NomorHP,
			"jabatan":   req.Jabatan,
			"role_id":   req.RoleID,
			"status":    status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if helper.IsDuplicateErrorGORM(result.Error) {
			response.ValidationError(w, map[string][]string{"email": {"Email telah digunakan"}})
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
		"Memperbarui pengguna: "+req.Nama, existing, req)
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
		"Mengubah status pengguna "+existing.Nama+" menjadi "+statusText, existing, map[string]bool{"status": req.Status})
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
		"Meriset password pengguna: "+existing.Nama, nil, nil)
	response.Success(w, http.StatusOK, "Password berhasil direset", nil)
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	type roleItem struct {
		ID   string `json:"id"`
		Nama string `json:"nama"`
	}

	var roles []roleItem
	if err := h.DB.WithContext(r.Context()).
		Table("roles").
		Select("id, nama").
		Where("deleted_at IS NULL AND status = ?", true).
		Order("nama ASC").
		Scan(&roles).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.Success(w, http.StatusOK, "Daftar Role", roles)
}
