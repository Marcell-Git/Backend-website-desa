package dashboard

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	Pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{Pool: pool}
}

type Summary struct {
	TotalKK              int64    `json:"total_kk"`
	TotalPenduduk        int64    `json:"total_penduduk"`
	SaldoKas             float64  `json:"saldo_kas"`
	PengeluaranBulanIni  float64  `json:"pengeluaran_bulan_ini"`
	PengaduanBaru        int64    `json:"pengaduan_baru"`
	PengaduanDiproses    int64    `json:"pengaduan_diproses"`
	PengumumanAktif      int64    `json:"pengumuman_aktif"`
	AktivitasTerbaru     []Aktivitas `json:"aktivitas_terbaru"`
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

	h.Pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM kartu_keluarga WHERE deleted_at IS NULL`).Scan(&s.TotalKK)
	h.Pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM anggota_keluarga WHERE deleted_at IS NULL`).Scan(&s.TotalPenduduk)

	rows, err := h.Pool.Query(r.Context(), `
		SELECT al.id, al.modul, al.aktivitas, COALESCE(al.deskripsi, ''),
		       COALESCE(u.nama, 'Sistem'), to_char(al.created_at, 'DD-MM-YYYY HH24:MI:SS')
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.user_id
		ORDER BY al.created_at DESC
		LIMIT 10`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var a Aktivitas
		if err := rows.Scan(&a.ID, &a.Modul, &a.Aktivitas, &a.Deskripsi, &a.Nama, &a.Waktu); err != nil {
			response.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		s.AktivitasTerbaru = append(s.AktivitasTerbaru, a)
	}
	if rows.Err() != nil {
		response.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.Success(w, http.StatusOK, "Ringkasan dashboard", s)
}
