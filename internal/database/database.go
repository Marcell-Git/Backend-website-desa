package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB *gorm.DB
}

func Connect(databaseURL string) (*Database, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan sql.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}

	return &Database{DB: db}, nil
}

func (d *Database) RunMigrations() error {
	hadVisi := d.DB.Migrator().HasColumn(&WebsiteSettings{}, "visi")
	hadMisi := d.DB.Migrator().HasColumn(&WebsiteSettings{}, "misi")
	hadShowVisiMisi := d.DB.Migrator().HasColumn(&WebsiteSettings{}, "show_visi_misi")
	err := d.DB.AutoMigrate(
		&Role{},
		&Permission{},
		&RolePermission{},
		&User{},
		&RefreshToken{},
		&KartuKeluarga{},
		&AnggotaKeluarga{},
		&JenisIuran{},
		&KategoriPengeluaran{},
		&RekapKas{},
		&RekapIuran{},
		&RekapPengeluaran{},
		&Pengumuman{},
		&AnakKos{},
		&AuditLog{},
		&Pengaduan{},
		&PengaduanRiwayat{},
		&WebsiteSettings{},
		&Backup{},
	)
	if err != nil {
		return fmt.Errorf("gagal menjalankan migrasi: %w", err)
	}
	if !hadVisi || !hadMisi || !hadShowVisiMisi {
		updates := map[string]interface{}{}
		if !hadVisi {
			updates["visi"] = "Terwujudnya lingkungan RT 26 yang rukun, aman, peduli, dan sejahtera melalui gotong royong serta pelayanan warga yang transparan."
		}
		if !hadMisi {
			updates["misi"] = StringList{
				"Memperkuat kerukunan warga melalui kegiatan gotong royong dan silaturahmi.",
				"Meningkatkan keamanan dan kepedulian terhadap lingkungan.",
				"Menyediakan pelayanan warga yang cepat, terbuka, dan bertanggung jawab.",
				"Mendorong partisipasi warga dalam setiap kegiatan RT.",
			}
		}
		if !hadShowVisiMisi {
			updates["show_visi_misi"] = true
		}
		if len(updates) > 0 {
			if err := d.DB.Model(&WebsiteSettings{}).Where("1 = 1").Updates(updates).Error; err != nil {
				return fmt.Errorf("gagal mengisi pengaturan visi dan misi: %w", err)
			}
		}
	}

	if err := d.DB.Exec(`ALTER TABLE users DROP COLUMN IF EXISTS email`).Error; err != nil {
		return fmt.Errorf("gagal menghapus kolom email: %w", err)
	}
	if err := d.DB.Exec(`DROP INDEX IF EXISTS idx_users_email`).Error; err != nil {
		return fmt.Errorf("gagal menghapus index email: %w", err)
	}
	if err := d.DB.Exec(`ALTER TABLE users DROP COLUMN IF EXISTS nama`).Error; err != nil {
		return fmt.Errorf("gagal menghapus kolom nama: %w", err)
	}

	log.Println("migrasi diterapkan via AutoMigrate")
	return nil
}

func (d *Database) Close() {
	sqlDB, err := d.DB.DB()
	if err == nil {
		sqlDB.Close()
	}
}
