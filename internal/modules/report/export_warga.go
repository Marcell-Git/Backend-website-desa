package report

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/database"
	"rt26/backend/internal/middleware"
	"rt26/backend/internal/pkg/response"
)

type kkCetakRow struct {
	ID         string  `json:"id"`
	NomorKK    string  `json:"nomor_kk"`
	NamaKepala *string `json:"nama_kepala"`
	NIKKepala  *string `json:"nik_kepala"`
	Alamat     string  `json:"alamat"`
	RT         string  `json:"rt"`
	RW         string  `json:"rw"`
	NomorHP    *string `json:"nomor_hp"`
}

type anggotaCetakRow struct {
	KartuKeluargaID  string  `json:"kartu_keluarga_id"`
	NIK              string  `json:"nik"`
	NamaLengkap      string  `json:"nama_lengkap"`
	JenisKelamin     string  `json:"jenis_kelamin"`
	TempatLahir      string  `json:"tempat_lahir"`
	TanggalLahir     string  `json:"tanggal_lahir"`
	Agama            string  `json:"agama"`
	Pekerjaan        *string `json:"pekerjaan"`
	HubunganKeluarga string  `json:"hubungan_keluarga"`
	Status           bool    `json:"status"`
}

// toCells mengubah anggota menjadi sel tabel sesuai urutan anggotaCetakHeaders.
// Kolom No dikosongkan karena penomorannya diisi setelah pengelompokan per KK.
func (a anggotaCetakRow) toCells() []string {
	return []string{
		"",
		orDash(a.NamaLengkap),
		orDash(a.NIK),
		labelJenisKelamin(a.JenisKelamin),
		orDash(a.TempatLahir),
		orDash(a.TanggalLahir),
		orDash(a.Agama),
		derefStr(a.Pekerjaan),
		orDash(a.HubunganKeluarga),
		labelStatus(a.Status),
	}
}

type anakKosCetakRow struct {
	NIK          string  `json:"nik"`
	NamaLengkap  string  `json:"nama_lengkap"`
	JenisKelamin string  `json:"jenis_kelamin"`
	TanggalLahir string  `json:"tanggal_lahir"`
	Agama        string  `json:"agama"`
	Pekerjaan    *string `json:"pekerjaan"`
	NomorHP      *string `json:"nomor_hp"`
	AlamatAsal   string  `json:"alamat_asal"`
	Status       bool    `json:"status"`
}

func writePDF(w http.ResponseWriter, baseName string, table PDFTable) {
	pdfBytes, err := generatePDFLandscape(table)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-%s.pdf\"", baseName, time.Now().Format("20060102")))
	w.Write(pdfBytes)
}

func auditExport(r *http.Request, db *gorm.DB, aktivitas, deskripsi string, jumlah int) {
	uid := middleware.GetUserID(r)
	database.RecordAudit(r.Context(), db, &uid, "Master Data", aktivitas, deskripsi, nil, map[string]any{"jumlah_data": jumlah})
}

func labelJenisKelamin(jk string) string {
	if jk == "P" {
		return "Perempuan"
	}
	return "Laki-laki"
}

func labelStatus(status bool) string {
	if status {
		return "Aktif"
	}
	return "Nonaktif"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// ExportKKPDF mencetak seluruh Kartu Keluarga RT 26 beserta daftar anggotanya.
// Satu KK menjadi satu blok: informasi KK lalu tabel anggota di bawahnya.
func (h *ExportHandler) ExportKKPDF(w http.ResponseWriter, r *http.Request) {
	db := h.DB.WithContext(r.Context())

	var kks []kkCetakRow
	if err := db.
		Table("kartu_keluarga kk").
		Select(`kk.id, kk.nomor_kk, a.nama_lengkap as nama_kepala, a.nik as nik_kepala, kk.alamat, kk.rt, kk.rw, kk.nomor_hp`).
		Joins("LEFT JOIN anggota_keluarga a ON a.id = kk.kepala_keluarga_id AND a.deleted_at IS NULL").
		Where("kk.deleted_at IS NULL").
		Order("kk.nomor_kk ASC").
		Scan(&kks).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data Kartu Keluarga")
		return
	}

	var anggota []anggotaCetakRow
	if err := db.
		Table("anggota_keluarga ag").
		Select(`ag.kartu_keluarga_id, ag.nik, ag.nama_lengkap, ag.jenis_kelamin, ag.tempat_lahir,
			to_char(ag.tanggal_lahir, 'DD/MM/YYYY') as tanggal_lahir, ag.agama, ag.pekerjaan, ag.hubungan_keluarga, ag.status`).
		Where("ag.deleted_at IS NULL").
		Order("ag.kartu_keluarga_id ASC, (ag.hubungan_keluarga = 'Kepala Keluarga') DESC, ag.nama_lengkap ASC").
		Scan(&anggota).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data anggota keluarga")
		return
	}

	anggotaPerKK := make(map[string][][]string, len(kks))
	totalAnggota := 0
	for _, ag := range anggota {
		baris := ag.toCells()
		anggotaPerKK[ag.KartuKeluargaID] = append(anggotaPerKK[ag.KartuKeluargaID], baris)
		totalAnggota++
	}

	blocks := make([]KKBlock, 0, len(kks))
	for _, kk := range kks {
		rows := anggotaPerKK[kk.ID]
		for i := range rows {
			rows[i][0] = fmt.Sprintf("%d", i+1)
		}
		blocks = append(blocks, KKBlock{
			NomorKK:    kk.NomorKK,
			NamaKepala: derefStr(kk.NamaKepala),
			NIKKepala:  derefStr(kk.NIKKepala),
			Alamat:     orDash(kk.Alamat),
			RT:         kk.RT,
			RW:         kk.RW,
			NomorHP:    derefStr(kk.NomorHP),
			Anggota:    rows,
		})
	}

	pdfBytes, err := generatePDFKKBlocks(
		"DAFTAR KARTU KELUARGA BESERTA ANGGOTANYA",
		"Warga RT 26 Kelurahan Tegalyoso, Kecamatan Klaten Selatan, Kabupaten Klaten",
		blocks,
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal generate PDF")
		return
	}

	auditExport(r, h.DB, "Export KK", fmt.Sprintf("Mencetak %d Kartu Keluarga dengan %d anggota ke PDF", len(blocks), totalAnggota), len(blocks))

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"data-kartu-keluarga-%s.pdf\"", time.Now().Format("20060102")))
	w.Write(pdfBytes)
}

// ExportAnakKosPDF mencetak seluruh data Anak Kos / Warga Kontrak dalam format A4 landscape.
func (h *ExportHandler) ExportAnakKosPDF(w http.ResponseWriter, r *http.Request) {
	var items []anakKosCetakRow
	if err := h.DB.WithContext(r.Context()).
		Table("anak_kos ak").
		Select("ak.nik, ak.nama_lengkap, ak.jenis_kelamin, to_char(ak.tanggal_lahir, 'DD/MM/YYYY') as tanggal_lahir, ak.agama, ak.pekerjaan, ak.nomor_hp, ak.alamat_asal, ak.status").
		Where("ak.deleted_at IS NULL").
		Order("ak.nama_lengkap ASC").
		Scan(&items).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat data Anak Kos")
		return
	}

	headers := []string{"No", "NIK", "Nama Lengkap", "Jenis Kelamin", "Tgl Lahir", "Agama", "Pekerjaan", "No. HP", "Alamat Asal", "Status"}
	widths := []float64{10, 30, 40, 24, 24, 22, 30, 28, 51, 18}
	aligns := []string{"C", "L", "L", "C", "C", "L", "L", "L", "L", "C"}

	rows := make([][]string, 0, len(items))
	var aktif, nonaktif int
	for i, it := range items {
		if it.Status {
			aktif++
		} else {
			nonaktif++
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			it.NIK,
			it.NamaLengkap,
			labelJenisKelamin(it.JenisKelamin),
			orDash(it.TanggalLahir),
			it.Agama,
			derefStr(it.Pekerjaan),
			derefStr(it.NomorHP),
			orDash(it.AlamatAsal),
			labelStatus(it.Status),
		})
	}

	auditExport(r, h.DB, "Export Anak Kos", fmt.Sprintf("Mencetak %d data anak kos ke PDF", len(items)), len(items))

	writePDF(w, "data-anak-kos", PDFTable{
		Title:     "DAFTAR ANAK KOS / WARGA KONTRAK",
		Subtitle:  "Warga RT 26 Kelurahan Tegalyoso, Kecamatan Klaten Selatan, Kabupaten Klaten",
		Headers:   headers,
		Rows:      rows,
		Widths:    widths,
		Aligns:    aligns,
		MaxLines:  2,
		TotalText: fmt.Sprintf("Total: %d anak kos (%d aktif, %d nonaktif)", len(items), aktif, nonaktif),
	})
}
