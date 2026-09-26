package database

import (
	"fmt"

	"rt26/backend/internal/pkg/helper"

	"gorm.io/gorm/clause"
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
	{"anak-kos", "view", "View Anak Kos"},
	{"anak-kos", "create", "Create Anak Kos"},
	{"anak-kos", "update", "Update Anak Kos"},
	{"anak-kos", "delete", "Delete Anak Kos"},
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
	{"pengurus", "view", "View Pengurus"},
	{"pengurus", "create", "Create Pengurus"},
	{"pengurus", "update", "Update Pengurus"},
	{"pengurus", "delete", "Delete Pengurus"},
	{"berita-kegiatan", "view", "View Berita dan Kegiatan"},
	{"berita-kegiatan", "create", "Create Berita dan Kegiatan"},
	{"berita-kegiatan", "update", "Update Berita dan Kegiatan"},
	{"berita-kegiatan", "delete", "Delete Berita dan Kegiatan"},
	{"inventaris", "view", "View Inventaris"},
	{"inventaris", "create", "Create Inventaris"},
	{"inventaris", "update", "Update Inventaris"},
	{"inventaris", "delete", "Delete Inventaris"},
	{"peraturan", "view", "View Peraturan"},
	{"peraturan", "create", "Create Peraturan"},
	{"peraturan", "update", "Update Peraturan"},
	{"peraturan", "delete", "Delete Peraturan"},
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

func (d *Database) Seed() error {
	db := d.DB

	// Seed default website settings
	var settingsCount int64
	db.Model(&WebsiteSettings{}).Count(&settingsCount)
	if settingsCount == 0 {
		settings := WebsiteSettings{
			NamaWebsite: "SIM RT Tegalyoso 26",
			NamaRT:      "RT 26",
			Desa:        "Tegalyoso",
			Kecamatan:   "Klaten Selatan",
			Kabupaten:   "Klaten",
			Provinsi:    "Jawa Tengah",
			Copyright:   "© 2026 RT 26 Kel. Tegalyoso",
			Visi:        "Terwujudnya lingkungan RT 26 yang rukun, aman, peduli, dan sejahtera melalui gotong royong serta pelayanan warga yang transparan.",
			Misi: StringList{
				"Memperkuat kerukunan warga melalui kegiatan gotong royong dan silaturahmi.",
				"Meningkatkan keamanan dan kepedulian terhadap lingkungan.",
				"Menyediakan pelayanan warga yang cepat, terbuka, dan bertanggung jawab.",
				"Mendorong partisipasi warga dalam setiap kegiatan RT.",
			},
			ShowPengumuman: true,
			ShowLaporan:    true,
			ShowPengaduan:  true,
			ShowVisiMisi:   true,
		}
		if err := db.Create(&settings).Error; err != nil {
			return fmt.Errorf("gagal seed website settings: %w", err)
		}
	}

	permIDs := map[string]string{}

	for _, p := range allPermissions {
		perm := Permission{Nama: p.nama, Modul: p.modul, Aksi: p.aksi}
		result := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "modul"}, {Name: "aksi"}},
			DoUpdates: clause.AssignmentColumns([]string{"nama"}),
		}).Create(&perm)
		if result.Error != nil {
			return result.Error
		}
		permIDs[p.modul+":"+p.aksi] = perm.ID
	}

	superAdminID, err := d.ensureRole("Super Admin", "Administrator sistem dengan akses penuh")
	if err != nil {
		return err
	}
	pengurusID, err := d.ensureRole("Pengurus", "Pengurus RT yang mengelola data operasional")
	if err != nil {
		return err
	}

	for _, pid := range permIDs {
		rp := RolePermission{RoleID: superAdminID, PermissionID: pid}
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rp)
	}

	pengurusPerms := []string{
		"dashboard:view",
		"kartu-keluarga:view", "kartu-keluarga:create", "kartu-keluarga:update", "kartu-keluarga:delete", "kartu-keluarga:export",
		"anggota:view", "anggota:create", "anggota:update", "anggota:delete", "anggota:export",
		"anak-kos:view", "anak-kos:create", "anak-kos:update", "anak-kos:delete",
		"kas:view", "kas:create", "kas:update", "kas:delete", "kas:export",
		"iuran:view", "iuran:create", "iuran:update", "iuran:delete", "iuran:export",
		"pengeluaran:view", "pengeluaran:create", "pengeluaran:update", "pengeluaran:delete", "pengeluaran:export",
		"jenis-iuran:view", "jenis-iuran:create", "jenis-iuran:update", "jenis-iuran:delete",
		"kategori-pengeluaran:view", "kategori-pengeluaran:create", "kategori-pengeluaran:update", "kategori-pengeluaran:delete",
		"pengumuman:view", "pengumuman:create", "pengumuman:update", "pengumuman:delete", "pengumuman:export",
		"berita-kegiatan:view", "berita-kegiatan:create", "berita-kegiatan:update", "berita-kegiatan:delete",
		"inventaris:view", "inventaris:create", "inventaris:update", "inventaris:delete",
		"peraturan:view", "peraturan:create", "peraturan:update", "peraturan:delete",
		"pengurus:view", "pengurus:create", "pengurus:update", "pengurus:delete",
		"pengaduan:view", "pengaduan:create", "pengaduan:update", "pengaduan:delete",
		"laporan:view", "laporan:export",
		"website-settings:view",
		"audit-log:view",
	}
	for _, key := range pengurusPerms {
		if _, ok := permIDs[key]; !ok {
			continue
		}
		rp := RolePermission{RoleID: pengurusID, PermissionID: permIDs[key]}
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rp)
	}

	hash, err := helper.HashPassword("admin123")
	if err != nil {
		return err
	}

	var count int64
	db.Model(&User{}).Where("username = ?", "admin").Count(&count)
	if count == 0 {
		admin := User{
			Username: "admin",
			Password: hash,
			Jabatan:  strPtr("Administrator"),
			RoleID:   superAdminID,
			Status:   true,
		}
		if err := db.Create(&admin).Error; err != nil {
			return err
		}
	}

	fmt.Println("seeding selesai: role Super Admin & Pengurus, akun admin")
	return nil
}

func (d *Database) ensureRole(nama, deskripsi string) (string, error) {
	var role Role
	result := d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "nama"}},
		DoUpdates: clause.AssignmentColumns([]string{"deskripsi"}),
	}).Assign(Role{Nama: nama, Deskripsi: deskripsi}).FirstOrCreate(&role, Role{Nama: nama})
	if result.Error != nil {
		return "", result.Error
	}
	return role.ID, nil
}

func strPtr(s string) *string {
	return &s
}
