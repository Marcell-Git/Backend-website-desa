package user

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

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
