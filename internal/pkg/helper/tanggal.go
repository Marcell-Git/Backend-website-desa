package helper

import (
	"fmt"
	"time"
)

var namaBulanSingkat = [12]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// FormatTanggal mengubah waktu menjadi format tanggal standar aplikasi, contoh: 12-Sep-2004.
func FormatTanggal(t time.Time) string {
	return fmt.Sprintf("%02d-%s-%d", t.Day(), namaBulanSingkat[t.Month()-1], t.Year())
}

// FormatTanggalWaktu sama seperti FormatTanggal ditambah jam, contoh: 12-Sep-2004 14:30.
func FormatTanggalWaktu(t time.Time) string {
	return FormatTanggal(t) + " " + t.Format("15:04")
}

// FormatTanggalISO memformat string tanggal YYYY-MM-DD. String kosong atau tidak valid
// dikembalikan apa adanya.
func FormatTanggalISO(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return FormatTanggal(t)
}
