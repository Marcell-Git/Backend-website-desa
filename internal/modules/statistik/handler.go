package statistik

import (
	"math"
	"net/http"
	"sort"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/pkg/response"
)

const (
	maksPekerjaan   = 5
	labelKos        = "Anak Kos / Warga Kontrak"
	labelBelumDiisi = "Belum diisi"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

type Label struct {
	Label  string  `json:"label"`
	Jumlah int64   `json:"jumlah"`
	Persen float64 `json:"persen"`
}

type Statistik struct {
	TotalWarga   int64   `json:"total_warga"`
	TotalKK      int64   `json:"total_kk"`
	TotalAnakKos int64   `json:"total_anak_kos"`
	Gender       []Label `json:"gender"`
	StatusKK     []Label `json:"status_kk"`
	Pekerjaan    []Label `json:"pekerjaan"`
	Umur         []Label `json:"umur"`
}

type jumlahRow struct {
	Key    string
	Jumlah int64
}

type bucketPekerjaan struct {
	Label  string
	Jumlah int64
	varian int64
}

var (
	urutanGender = []string{"L", "P"}
	labelGender  = map[string]string{"L": "Laki-laki", "P": "Perempuan"}
	urutanStatus = []string{"Kepala Keluarga", "Anggota Keluarga", labelKos}
	urutanUmur   = []string{
		"Balita (0-4 tahun)",
		"Anak (5-14 tahun)",
		"Remaja (15-24 tahun)",
		"Dewasa (25-54 tahun)",
		"Lansia (55 tahun ke atas)",
		labelBelumDiisi,
	}
)

func (h *Handler) showStatistikAktif(r *http.Request) bool {
	var show bool
	h.DB.WithContext(r.Context()).
		Table("website_settings").
		Select("show_statistik").
		Scan(&show)
	return show
}

func mulaiHurufBesar(s string) bool {
	huruf := []rune(s)
	return len(huruf) > 0 && unicode.IsUpper(huruf[0])
}

func hitungPersen(jumlah, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(jumlah)/float64(total)*1000) / 10
}

func urutLabels(keys []string, counts map[string]int64, total int64) []Label {
	out := make([]Label, 0, len(keys))
	for _, k := range keys {
		jumlah := counts[k]
		out = append(out, Label{Label: k, Jumlah: jumlah, Persen: hitungPersen(jumlah, total)})
	}
	return out
}

func hanyaYangAda(keys []string, counts map[string]int64) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if counts[k] > 0 {
			out = append(out, k)
		}
	}
	return out
}

func (h *Handler) PublicStatistik(w http.ResponseWriter, r *http.Request) {
	if !h.showStatistikAktif(r) {
		response.Error(w, http.StatusForbidden, "Statistik warga tidak dipublikasikan")
		return
	}

	db := h.DB.WithContext(r.Context())

	var totalAnggota, totalAnakKos, totalKK int64
	if err := db.Model(&database.AnggotaKeluarga{}).Where("deleted_at IS NULL").Count(&totalAnggota).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat statistik warga")
		return
	}
	if err := db.Model(&database.AnakKos{}).Where("deleted_at IS NULL").Count(&totalAnakKos).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat statistik warga")
		return
	}
	if err := db.Model(&database.KartuKeluarga{}).Where("deleted_at IS NULL").Count(&totalKK).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat statistik warga")
		return
	}

	totalWarga := totalAnggota + totalAnakKos
	resp := Statistik{
		TotalWarga:   totalWarga,
		TotalKK:      totalKK,
		TotalAnakKos: totalAnakKos,
		Gender:       []Label{},
		StatusKK:     []Label{},
		Pekerjaan:    []Label{},
		Umur:         []Label{},
	}

	if totalWarga == 0 {
		response.Success(w, http.StatusOK, "Statistik warga berhasil dimuat", resp)
		return
	}

	var genderRows []jumlahRow
	if err := db.Raw(`
		SELECT COALESCE(jenis_kelamin, '') AS key, COUNT(*) AS jumlah
		FROM (
			SELECT jenis_kelamin FROM anggota_keluarga WHERE deleted_at IS NULL
			UNION ALL
			SELECT jenis_kelamin FROM anak_kos WHERE deleted_at IS NULL
		) t
		GROUP BY COALESCE(jenis_kelamin, '')`).Scan(&genderRows).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat statistik warga")
		return
	}
	countsGender := make(map[string]int64)
	for _, row := range genderRows {
		if _, ok := labelGender[row.Key]; ok {
			countsGender[row.Key] += row.Jumlah
		}
	}
	for _, k := range hanyaYangAda(urutanGender, countsGender) {
		resp.Gender = append(resp.Gender, Label{
			Label:  labelGender[k],
			Jumlah: countsGender[k],
			Persen: hitungPersen(countsGender[k], totalWarga),
		})
	}

	var statusRows []jumlahRow
	if err := db.Raw(`
		SELECT status AS key, COUNT(*) AS jumlah
		FROM (
			SELECT CASE
				WHEN LOWER(TRIM(a.hubungan_keluarga)) = 'kepala keluarga'
					OR (k.kepala_keluarga_id IS NOT NULL AND k.kepala_keluarga_id = a.id)
				THEN 'Kepala Keluarga'
				ELSE 'Anggota Keluarga'
			END AS status
			FROM anggota_keluarga a
			LEFT JOIN kartu_keluarga k ON k.id = a.kartu_keluarga_id AND k.deleted_at IS NULL
			WHERE a.deleted_at IS NULL
			UNION ALL
			SELECT '` + labelKos + `' FROM anak_kos WHERE deleted_at IS NULL
		) t
		GROUP BY status`).Scan(&statusRows).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat statistik warga")
		return
	}
	countsStatus := make(map[string]int64)
	for _, row := range statusRows {
		countsStatus[row.Key] += row.Jumlah
	}
	resp.StatusKK = urutLabels(hanyaYangAda(urutanStatus, countsStatus), countsStatus, totalWarga)

	// Umur dihitung dari tanggal_lahir, tidak disimpan sebagai kolom. Daftar
	// gabungan dibuat sekali lewat CTE supaya CASE batas usianya tidak perlu
	// ditulis ulang untuk tiap tabel.
	//
	// Guard tanggal_lahir IS NULL wajib ada: age(CURRENT_DATE, NULL) bernilai
	// NULL, sehingga seluruh WHEN tidak cocok dan record tanpa tanggal lahir
	// akan ikut terhitung sebagai kelompok terakhir (Lansia).
	var umurRows []jumlahRow
	if err := db.Raw(`
		WITH gabungan AS (
			SELECT tanggal_lahir FROM anggota_keluarga WHERE deleted_at IS NULL
			UNION ALL
			SELECT tanggal_lahir FROM anak_kos WHERE deleted_at IS NULL
		)
		SELECT kelompok AS key, COUNT(*) AS jumlah
		FROM (
			SELECT CASE
				WHEN tanggal_lahir IS NULL THEN '` + labelBelumDiisi + `'
				WHEN EXTRACT(YEAR FROM age(CURRENT_DATE, tanggal_lahir)) < 5 THEN '` + urutanUmur[0] + `'
				WHEN EXTRACT(YEAR FROM age(CURRENT_DATE, tanggal_lahir)) < 15 THEN '` + urutanUmur[1] + `'
				WHEN EXTRACT(YEAR FROM age(CURRENT_DATE, tanggal_lahir)) < 25 THEN '` + urutanUmur[2] + `'
				WHEN EXTRACT(YEAR FROM age(CURRENT_DATE, tanggal_lahir)) < 55 THEN '` + urutanUmur[3] + `'
				ELSE '` + urutanUmur[4] + `'
			END AS kelompok
			FROM gabungan
		) t
		GROUP BY kelompok`).Scan(&umurRows).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat statistik warga")
		return
	}
	countsUmur := make(map[string]int64)
	for _, row := range umurRows {
		countsUmur[row.Key] += row.Jumlah
	}
	resp.Umur = urutLabels(hanyaYangAda(urutanUmur, countsUmur), countsUmur, totalWarga)

	var pekerjaanRows []jumlahRow
	if err := db.Raw(`
		SELECT pekerjaan AS key, COUNT(*) AS jumlah
		FROM (
			SELECT COALESCE(NULLIF(TRIM(pekerjaan), ''), '` + labelBelumDiisi + `') AS pekerjaan
			FROM anggota_keluarga WHERE deleted_at IS NULL
			UNION ALL
			SELECT COALESCE(NULLIF(TRIM(pekerjaan), ''), '` + labelBelumDiisi + `')
			FROM anak_kos WHERE deleted_at IS NULL
		) t
		GROUP BY pekerjaan
		ORDER BY COUNT(*) DESC`).Scan(&pekerjaanRows).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat statistik warga")
		return
	}

	// Varian huruf besar-kecil digabung agar satu pekerjaan tidak terpecah jadi
	// beberapa baris. Penulisannya diambil dari varian yang paling banyak
	// dipakai, dan diurutkan ulang karena urutan dari SQL mengikuti jumlah per
	// varian, bukan jumlah gabungan.
	var buckets []bucketPekerjaan
	idxBucket := make(map[string]int)
	for _, row := range pekerjaanRows {
		key := strings.TrimSpace(row.Key)
		if key == "" {
			key = labelBelumDiisi
		}
		lower := strings.ToLower(key)
		if i, seen := idxBucket[lower]; seen {
			b := &buckets[i]
			if row.Jumlah > b.varian ||
				(row.Jumlah == b.varian && mulaiHurufBesar(key) && !mulaiHurufBesar(b.Label)) {
				b.Label = key
				b.varian = row.Jumlah
			}
			b.Jumlah += row.Jumlah
			continue
		}
		idxBucket[lower] = len(buckets)
		buckets = append(buckets, bucketPekerjaan{Label: key, Jumlah: row.Jumlah, varian: row.Jumlah})
	}
	// Urutan diikat dengan nama saat jumlah sama. Tanpa itu urutan lima besar
	// bisa berubah-ubah antar-request, karena PostgreSQL tidak menjamin urutan
	// baris yang jumlah-nya sama, dan hanya lima teratas yang ditampilkan --
	// jadi satu pekerjaan bisa keluar-masuk tanpa ada datanya yang berubah.
	sort.SliceStable(buckets, func(i, j int) bool {
		if buckets[i].Jumlah != buckets[j].Jumlah {
			return buckets[i].Jumlah > buckets[j].Jumlah
		}
		return buckets[i].Label < buckets[j].Label
	})

	keysPekerjaan := make([]string, 0, len(buckets))
	countsPekerjaan := make(map[string]int64, len(buckets))
	for _, b := range buckets {
		keysPekerjaan = append(keysPekerjaan, b.Label)
		countsPekerjaan[b.Label] = b.Jumlah
	}
	if len(keysPekerjaan) > maksPekerjaan {
		keysPekerjaan = keysPekerjaan[:maksPekerjaan]
	}

	resp.Pekerjaan = urutLabels(keysPekerjaan, countsPekerjaan, totalWarga)

	response.Success(w, http.StatusOK, "Statistik warga berhasil dimuat", resp)
}
