package report

import (
	"net/http"
	"strconv"
	"time"

	"rt26/backend/internal/pkg/response"
)

type PublicRingkasan struct {
	TotalKas         float64 `json:"total_kas"`
	TotalIuran       float64 `json:"total_iuran"`
	TotalPengeluaran float64 `json:"total_pengeluaran"`
	SaldoAkhir       float64 `json:"saldo_akhir"`
}

type PublicRekapBulanan struct {
	Bulan            int     `json:"bulan"`
	Kas              float64 `json:"kas"`
	Iuran            float64 `json:"iuran"`
	Pengeluaran      float64 `json:"pengeluaran"`
	SaldoAkhir       float64 `json:"saldo_akhir"`
}

type PublicLaporanResponse struct {
	Tahun        string               `json:"tahun"`
	Ringkasan    PublicRingkasan      `json:"ringkasan"`
	RekapBulanan []PublicRekapBulanan `json:"rekap_bulanan"`
}

func waktuIndonesia() time.Time {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Now().In(time.FixedZone("WIB", 7*3600))
	}
	return time.Now().In(loc)
}

func (h *Handler) showLaporanAktif(r *http.Request) bool {
	var show bool
	h.DB.WithContext(r.Context()).
		Table("website_settings").
		Select("show_laporan").
		Scan(&show)
	return show
}

func (h *Handler) sumTahun(r *http.Request, table string, year int) float64 {
	var total float64
	h.DB.WithContext(r.Context()).
		Table(table).
		Select("COALESCE(SUM(nominal), 0)").
		Where("deleted_at IS NULL AND EXTRACT(YEAR FROM tanggal) = ?", year).
		Scan(&total)
	return total
}

func (h *Handler) sumSebelumTahun(r *http.Request, table string, year int) float64 {
	var total float64
	h.DB.WithContext(r.Context()).
		Table(table).
		Select("COALESCE(SUM(nominal), 0)").
		Where("deleted_at IS NULL AND EXTRACT(YEAR FROM tanggal) < ?", year).
		Scan(&total)
	return total
}

func (h *Handler) sumPerBulan(r *http.Request, table string, year int) map[int]float64 {
	type bulanRow struct {
		Bulan int
		Total float64
	}
	var rows []bulanRow
	h.DB.WithContext(r.Context()).
		Table(table).
		Select("EXTRACT(MONTH FROM tanggal) as bulan, COALESCE(SUM(nominal), 0) as total").
		Where("deleted_at IS NULL AND EXTRACT(YEAR FROM tanggal) = ?", year).
		Group("EXTRACT(MONTH FROM tanggal)").
		Scan(&rows)

	result := make(map[int]float64, len(rows))
	for _, row := range rows {
		result[row.Bulan] = row.Total
	}
	return result
}

func (h *Handler) PublicLaporan(w http.ResponseWriter, r *http.Request) {
	if !h.showLaporanAktif(r) {
		response.Error(w, http.StatusForbidden, "Laporan keuangan tidak dipublikasikan")
		return
	}

	tahun := r.URL.Query().Get("tahun")
	if tahun == "" {
		tahun = strconv.Itoa(waktuIndonesia().Year())
	}
	year, err := strconv.Atoi(tahun)
	if err != nil || year < 2000 || year > 2100 {
		response.ValidationError(w, map[string][]string{
			"tahun": {"Tahun harus berupa angka yang valid"},
		})
		return
	}

	totalKas := h.sumTahun(r, "rekap_kas", year)
	totalIuran := h.sumTahun(r, "rekap_iuran", year)
	totalPengeluaran := h.sumTahun(r, "rekap_pengeluaran", year)

	kasPerBulan := h.sumPerBulan(r, "rekap_kas", year)
	iuranPerBulan := h.sumPerBulan(r, "rekap_iuran", year)
	pengeluaranPerBulan := h.sumPerBulan(r, "rekap_pengeluaran", year)

	saldoAwal := (h.sumSebelumTahun(r, "rekap_kas", year) + h.sumSebelumTahun(r, "rekap_iuran", year)) -
		h.sumSebelumTahun(r, "rekap_pengeluaran", year)

	rekap := make([]PublicRekapBulanan, 0, 12)
	saldo := saldoAwal
	for bulan := 1; bulan <= 12; bulan++ {
		kas := kasPerBulan[bulan]
		iuran := iuranPerBulan[bulan]
		pengeluaran := pengeluaranPerBulan[bulan]
		saldo += (kas + iuran) - pengeluaran
		rekap = append(rekap, PublicRekapBulanan{
			Bulan:       bulan,
			Kas:         kas,
			Iuran:       iuran,
			Pengeluaran: pengeluaran,
			SaldoAkhir:  saldo,
		})
	}

	resp := PublicLaporanResponse{
		Tahun: tahun,
		Ringkasan: PublicRingkasan{
			TotalKas:         totalKas,
			TotalIuran:       totalIuran,
			TotalPengeluaran: totalPengeluaran,
			SaldoAkhir:       saldo,
		},
		RekapBulanan: rekap,
	}

	response.Success(w, http.StatusOK, "Laporan keuangan publik berhasil dimuat", resp)
}
