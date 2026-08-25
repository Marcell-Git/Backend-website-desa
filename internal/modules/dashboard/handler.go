package dashboard

import (
	"fmt"
	"net/http"
	"strconv"
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
	TotalKK             int64       `json:"total_kk"`
	TotalPenduduk       int64       `json:"total_penduduk"`
	SaldoKas            float64     `json:"saldo_kas"`
	TotalIuran          float64     `json:"total_iuran"`
	PengeluaranBulanIni float64     `json:"pengeluaran_bulan_ini"`
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

func sumColumn(db *gorm.DB, table, column string, where string, args ...interface{}) float64 {
	var result float64
	query := "SELECT COALESCE(SUM(" + column + "), 0) FROM " + table + " WHERE deleted_at IS NULL"
	if where != "" {
		query += " AND " + where
	}
	db.Raw(query, args...).Scan(&result)
	return result
}

type GrafikItem struct {
	BulanLabel  string  `json:"bulan_label"`
	Kas         float64 `json:"kas"`
	Iuran       float64 `json:"iuran"`
	Pengeluaran float64 `json:"pengeluaran"`
}

var bulanIndo = []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

func (h *Handler) GrafikBulanan(w http.ResponseWriter, r *http.Request) {
	jumlahBulan := 12
	if p := r.URL.Query().Get("bulan"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 6 || v > 12 {
			response.ValidationError(w, map[string][]string{
				"bulan": {"Bulan harus berupa angka antara 6 sampai 12"},
			})
			return
		}
		jumlahBulan = v
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	now := time.Now().In(loc)
	awal := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -(jumlahBulan - 1), 0)

	type bulananRow struct {
		Tahun int
		Bulan int
		Total float64
	}

	sumPerBulan := func(table string) map[int]float64 {
		var rows []bulananRow
		h.DB.WithContext(r.Context()).
			Table(table).
			Select("EXTRACT(YEAR FROM tanggal) as tahun, EXTRACT(MONTH FROM tanggal) as bulan, COALESCE(SUM(nominal), 0) as total").
			Where("deleted_at IS NULL AND tanggal >= ?", awal.Format("2006-01-02")).
			Group("EXTRACT(YEAR FROM tanggal), EXTRACT(MONTH FROM tanggal)").
			Scan(&rows)

		result := make(map[int]float64, len(rows))
		for _, row := range rows {
			result[row.Tahun*100+row.Bulan] = row.Total
		}
		return result
	}

	kas := sumPerBulan("rekap_kas")
	iuran := sumPerBulan("rekap_iuran")
	pengeluaran := sumPerBulan("rekap_pengeluaran")

	data := make([]GrafikItem, 0, jumlahBulan)
	for i := jumlahBulan - 1; i >= 0; i-- {
		t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -i, 0)
		key := t.Year()*100 + int(t.Month())
		data = append(data, GrafikItem{
			BulanLabel:  fmt.Sprintf("%s %d", bulanIndo[t.Month()-1], t.Year()%100),
			Kas:         kas[key],
			Iuran:       iuran[key],
			Pengeluaran: pengeluaran[key],
		})
	}

	response.Success(w, http.StatusOK, "Grafik keuangan berhasil dimuat", data)
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	var s Summary

	h.DB.WithContext(r.Context()).Model(&database.KartuKeluarga{}).
		Where("deleted_at IS NULL").Count(&s.TotalKK)
	h.DB.WithContext(r.Context()).Model(&database.AnggotaKeluarga{}).
		Where("deleted_at IS NULL").Count(&s.TotalPenduduk)

	s.SaldoKas = sumColumn(h.DB, "rekap_kas", "nominal", "") +
		sumColumn(h.DB, "rekap_iuran", "nominal", "") -
		sumColumn(h.DB, "rekap_pengeluaran", "nominal", "")

	s.TotalIuran = sumColumn(h.DB, "rekap_iuran", "nominal", "")

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	s.PengeluaranBulanIni = sumColumn(h.DB, "rekap_pengeluaran", "nominal", "tanggal >= ?", startOfMonth)

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
		Select("al.id, al.modul, al.aktivitas, COALESCE(al.deskripsi, '') as deskripsi, COALESCE(u.username, 'Sistem') as nama, al.created_at as waktu").
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
