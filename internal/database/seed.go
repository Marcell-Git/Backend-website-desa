package database

import (
	"context"
	"fmt"

	"rt26/backend/internal/pkg/helper"
)

type permissionDef struct {
	modul string
	aksi  string
	nama  string
}

var allPermissions = []permissionDef{
	{"dashboard", "view", "View Dashboard"},
	{"kartu-keluarga", "view", "View Kartu Keluarga"},
	{"kartu-keluarga", "create", "Create Kartu Keluarga"},
	{"kartu-keluarga", "update", "Update Kartu Keluarga"},
	{"kartu-keluarga", "delete", "Delete Kartu Keluarga"},
	{"kartu-keluarga", "export", "Export Kartu Keluarga"},
	{"anggota", "view", "View Anggota Keluarga"},
	{"anggota", "create", "Create Anggota Keluarga"},
	{"anggota", "update", "Update Anggota Keluarga"},
	{"anggota", "delete", "Delete Anggota Keluarga"},
	{"anggota", "export", "Export Anggota Keluarga"},
	{"kas", "view", "View Rekap Kas"},
	{"kas", "create", "Create Rekap Kas"},
	{"kas", "update", "Update Rekap Kas"},
	{"kas", "delete", "Delete Rekap Kas"},
	{"kas", "export", "Export Rekap Kas"},
	{"iuran", "view", "View Rekap Iuran"},
	{"iuran", "create", "Create Rekap Iuran"},
	{"iuran", "update", "Update Rekap Iuran"},
	{"iuran", "delete", "Delete Rekap Iuran"},
	{"iuran", "export", "Export Rekap Iuran"},
	{"pengeluaran", "view", "View Rekap Pengeluaran"},
	{"pengeluaran", "create", "Create Rekap Pengeluaran"},
	{"pengeluaran", "update", "Update Rekap Pengeluaran"},
	{"pengeluaran", "delete", "Delete Rekap Pengeluaran"},
	{"pengeluaran", "export", "Export Rekap Pengeluaran"},
	{"jenis-iuran", "view", "View Jenis Iuran"},
	{"jenis-iuran", "create", "Create Jenis Iuran"},
	{"jenis-iuran", "update", "Update Jenis Iuran"},
	{"jenis-iuran", "delete", "Delete Jenis Iuran"},
	{"kategori-pengeluaran", "view", "View Kategori Pengeluaran"},
	{"kategori-pengeluaran", "create", "Create Kategori Pengeluaran"},
	{"kategori-pengeluaran", "update", "Update Kategori Pengeluaran"},
	{"kategori-pengeluaran", "delete", "Delete Kategori Pengeluaran"},
	{"pengumuman", "view", "View Pengumuman"},
	{"pengumuman", "create", "Create Pengumuman"},
	{"pengumuman", "update", "Update Pengumuman"},
	{"pengumuman", "delete", "Delete Pengumuman"},
	{"pengumuman", "export", "Export Pengumuman"},
	{"pengaduan", "view", "View Pengaduan"},
	{"pengaduan", "create", "Create Pengaduan"},
	{"pengaduan", "update", "Update Pengaduan"},
	{"pengaduan", "delete", "Delete Pengaduan"},
	{"laporan", "view", "View Laporan"},
	{"laporan", "export", "Export Laporan"},
	{"user", "view", "View User"},
	{"user", "create", "Create User"},
	{"user", "update", "Update User"},
	{"user", "delete", "Delete User"},
	{"user", "export", "Export User"},
	{"role", "view", "View Role"},
	{"role", "create", "Create Role"},
	{"role", "update", "Update Role"},
	{"role", "delete", "Delete Role"},
	{"permission", "view", "View Permission"},
	{"permission", "update", "Update Permission"},
	{"website-settings", "view", "View Website Settings"},
	{"website-settings", "update", "Update Website Settings"},
	{"backup", "view", "View Backup"},
	{"backup", "create", "Create Backup"},
	{"backup", "delete", "Delete Backup"},
	{"backup", "restore", "Restore Backup"},
	{"audit-log", "view", "View Audit Log"},
	{"audit-log", "export", "Export Audit Log"},
}

func (d *Database) Seed(ctx context.Context) error {
	permIDs := map[string]string{}

	for _, p := range allPermissions {
		var id string
		err := d.Pool.QueryRow(ctx, `
			INSERT INTO permissions (nama, modul, aksi) VALUES ($1, $2, $3)
			ON CONFLICT (modul, aksi) DO UPDATE SET nama = EXCLUDED.nama
			RETURNING id`, p.nama, p.modul, p.aksi).Scan(&id)
		if err != nil {
			return err
		}
		permIDs[p.modul+":"+p.aksi] = id
	}

	superAdminID, err := d.ensureRole(ctx, "Super Admin", "Administrator sistem dengan akses penuh")
	if err != nil {
		return err
	}
	pengurusID, err := d.ensureRole(ctx, "Pengurus", "Pengurus RT yang mengelola data operasional")
	if err != nil {
		return err
	}

	for _, pid := range permIDs {
		if _, err := d.Pool.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, superAdminID, pid); err != nil {
			return err
		}
	}

	pengurusPerms := []string{
		"dashboard:view",
		"kartu-keluarga:view", "kartu-keluarga:create", "kartu-keluarga:update", "kartu-keluarga:delete", "kartu-keluarga:export",
		"anggota:view", "anggota:create", "anggota:update", "anggota:delete", "anggota:export",
		"kas:view", "kas:create", "kas:update", "kas:delete", "kas:export",
		"iuran:view", "iuran:create", "iuran:update", "iuran:delete", "iuran:export",
		"pengeluaran:view", "pengeluaran:create", "pengeluaran:update", "pengeluaran:delete", "pengeluaran:export",
		"jenis-iuran:view", "jenis-iuran:create", "jenis-iuran:update", "jenis-iuran:delete",
		"kategori-pengeluaran:view", "kategori-pengeluaran:create", "kategori-pengeluaran:update", "kategori-pengeluaran:delete",
		"pengumuman:view", "pengumuman:create", "pengumuman:update", "pengumuman:delete", "pengumuman:export",
		"pengaduan:view", "pengaduan:create", "pengaduan:update", "pengaduan:delete",
		"laporan:view", "laporan:export",
		"website-settings:view",
		"audit-log:view",
	}
	for _, key := range pengurusPerms {
		if _, ok := permIDs[key]; !ok {
			continue
		}
		if _, err := d.Pool.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, pengurusID, permIDs[key]); err != nil {
			return err
		}
	}

	hash, err := helper.HashPassword("admin123")
	if err != nil {
		return err
	}

	var exists bool
	err = d.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`,
		"admin@email.com").Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		_, err = d.Pool.Exec(ctx, `
			INSERT INTO users (nama, email, password, jabatan, role_id, status)
			VALUES ('Super Admin', 'admin@email.com', $1, 'Administrator', $2, TRUE)`,
			hash, superAdminID)
		if err != nil {
			return err
		}
	}

	fmt.Println("seeding selesai: role Super Admin & Pengurus, akun admin@email.com")
	return nil
}

func (d *Database) ensureRole(ctx context.Context, nama, deskripsi string) (string, error) {
	var id string
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO roles (nama, deskripsi) VALUES ($1, $2)
		ON CONFLICT (nama) DO UPDATE SET deskripsi = EXCLUDED.deskripsi
		RETURNING id`, nama, deskripsi).Scan(&id)
	return id, err
}
