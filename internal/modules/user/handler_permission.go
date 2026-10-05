package user

import (
	"encoding/json"
	"net/http"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/response"
)

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
		"role_id":        id,
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
