package user

import (
	"github.com/go-chi/chi/v5"

	"rt26/backend/internal/middleware"
)

// Register mendaftarkan route yang butuh login.
func Register(r chi.Router, h *Handler) {
	r.Route("/users", func(r chi.Router) {
		r.With(middleware.RequirePermission("user", "view")).Get("/", h.ListUsers)
		r.With(middleware.RequirePermission("user", "create")).Post("/", h.CreateUser)
		r.With(middleware.RequirePermission("user", "view")).Get("/{id}", h.GetUser)
		r.With(middleware.RequirePermission("user", "update")).Put("/{id}", h.UpdateUser)
		r.With(middleware.RequirePermission("user", "update")).Patch("/{id}/status", h.UpdateUserStatus)
		r.With(middleware.RequirePermission("user", "update")).Patch("/{id}/password", h.ResetPassword)
		r.With(middleware.RequirePermission("user", "delete")).Delete("/{id}", h.DeleteUser)
	})

	r.Route("/roles", func(r chi.Router) {
		r.With(middleware.RequirePermission("role", "view")).Get("/", h.ListRoles)
		r.With(middleware.RequirePermission("role", "create")).Post("/", h.CreateRole)
		r.With(middleware.RequirePermission("role", "view")).Get("/{id}", h.GetRole)
		r.With(middleware.RequirePermission("role", "update")).Put("/{id}", h.UpdateRole)
		r.With(middleware.RequirePermission("role", "delete")).Delete("/{id}", h.DeleteRole)
		r.With(middleware.RequirePermission("role", "view")).Get("/{id}/permissions", h.GetRolePermissions)
		r.With(middleware.RequirePermission("role", "update")).Put("/{id}/permissions", h.UpdateRolePermissions)
	})

	r.With(middleware.RequirePermission("permission", "view")).Get("/permissions", h.ListPermissions)
}
