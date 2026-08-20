package dashboard

import (
	"net/http"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

type Summary struct {
	TotalKK             int64      `json:"total_kk"`
	TotalPenduduk       int64      `json:"total_penduduk"`
	SaldoKas            float64    `json:"saldo_kas"`
	PengeluaranBulanIni float64    `json:"pengeluaran_bulan_ini"`
	PengaduanBaru       int64      `json:"pengaduan_baru"`
	PengaduanDiproses   int64      `json:"pengaduan_diproses"`
	PengumumanAktif     int64      `json:"pengumuman_aktif"`
	AktivitasTerbaru    []Aktivitas `json:"aktivitas_terbaru"`
}

type Aktivitas struct {
	ID        string `json:"id"`
	Modul     string `json:"modul"`
	Aktivitas string `json:"aktivitas"`
	Deskripsi string `json:"deskripsi"`
	Nama      string `json:"nama_pengguna"`
	Waktu     string `json:"waktu"`
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	var s Summary

	h.DB.WithContext(r.Context()).Model(&database.KartuKeluarga{}).
		Where("deleted_at IS NULL").Count(&s.TotalKK)
	h.DB.WithContext(r.Context()).Model(&database.AnggotaKeluarga{}).
		Where("deleted_at IS NULL").Count(&s.TotalPenduduk)

	type auditRow struct {
		ID        string
		Modul     string
		Aktivitas string
		Deskripsi string
		Nama      string
		Waktu     time.Time
	}

	var rows []auditRow
	h.DB.WithContext(r.Context()).
		Table("audit_logs al").
		Select("al.id, al.modul, al.aktivitas, COALESCE(al.deskripsi, '') as deskripsi, COALESCE(u.nama, 'Sistem') as nama, al.created_at as waktu").
		Joins("LEFT JOIN users u ON u.id = al.user_id").
		Order("al.created_at DESC").
		Limit(10).
		Scan(&rows)

	for _, row := range rows {
		s.AktivitasTerbaru = append(s.AktivitasTerbaru, Aktivitas{
			ID:        row.ID,
			Modul:     row.Modul,
			Aktivitas: row.Aktivitas,
			Deskripsi: row.Deskripsi,
			Nama:      row.Nama,
			Waktu:     row.Waktu.Format("02-01-2006 15:04:05"),
		})
	}

	response.Success(w, http.StatusOK, "Ringkasan dashboard", s)
}
