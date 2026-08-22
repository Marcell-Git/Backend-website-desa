package masterdata

import (
	"encoding/json"
	"net/http"
	"strings"

	"rt26/backend/internal/pkg/response"
)

// KTPOCRData adalah hasil pembacaan foto KTP yang dikirim ke frontend
// untuk mengisi form anak kos secara otomatis.
type KTPOCRData struct {
	NIK          string `json:"nik"`
	NamaLengkap  string `json:"nama_lengkap"`
	JenisKelamin string `json:"jenis_kelamin"`
	TanggalLahir string `json:"tanggal_lahir"`
	Agama        string `json:"agama"`
	Pekerjaan    string `json:"pekerjaan"`
	AlamatAsal   string `json:"alamat_asal"`
}

// ktpOCRRaw menampung hasil mentah dari Gemini sebelum alamat dirangkai.
type ktpOCRRaw struct {
	NIK          string `json:"nik"`
	NamaLengkap  string `json:"nama_lengkap"`
	TanggalLahir string `json:"tanggal_lahir"`
	JenisKelamin string `json:"jenis_kelamin"`
	Agama        string `json:"agama"`
	Pekerjaan    string `json:"pekerjaan"`
	Alamat       string `json:"alamat"`
	RT           string `json:"rt"`
	RW           string `json:"rw"`
	Desa         string `json:"desa"`
	Kecamatan    string `json:"kecamatan"`
	Kabupaten    string `json:"kabupaten"`
	Provinsi     string `json:"provinsi"`
}

const ktpOCRPrompt = `Anda adalah asisten yang mengekstrak data dari foto Kartu Tanda Penduduk (KTP) Republik Indonesia.
Baca seluruh teks pada gambar dan ekstrak datanya ke dalam JSON sesuai skema yang diberikan.

Aturan:
- nik harus berupa 16 digit angka tanpa spasi atau tanda hubung.
- Baris "Tempat/Tgl Lahir" berisi tempat dan tanggal lahir sekaligus (contoh: "KLATEN, 25-09-1978").
  Isi hanya bagian tangganya ke "tanggal_lahir" dalam format YYYY-MM-DD; abaikan tempatnya.
- jenis_kelamin diisi "L" untuk LAKI-LAKI dan "P" untuk PEREMPUAN.
- agama salah satu dari: Islam, Kristen, Katolik, Hindu, Buddha, Konghucu.
- rt dan rw hanya angkanya saja, tanpa kata "RT"/"RW".
- alamat, desa, kecamatan, kabupaten, dan provinsi berasal dari blok alamat pada KTP.
- pekerjaan boleh kosong jika tidak tercantum.
- Jika suatu data tidak terbaca dengan jelas atau tidak ada di dokumen, isi string kosong.
- Jangan pernah mengarang atau menebak data yang tidak terlihat.`

func ktpOCRSchema() json.RawMessage {
	str := map[string]interface{}{"type": "string"}
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"nik":           str,
			"nama_lengkap":  str,
			"tanggal_lahir": str,
			"jenis_kelamin": str,
			"agama":         str,
			"pekerjaan":     str,
			"alamat":        str,
			"rt":            str,
			"rw":            str,
			"desa":          str,
			"kecamatan":     str,
			"kabupaten":     str,
			"provinsi":      str,
		},
	}
	b, _ := json.Marshal(schema)
	return b
}

func composeAlamatAsal(r *ktpOCRRaw) string {
	parts := []string{}
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			parts = append(parts, s)
		}
	}
	add(r.Alamat)
	if r.RT != "" || r.RW != "" {
		rt := r.RT
		if rt == "" {
			rt = "-"
		}
		rw := r.RW
		if rw == "" {
			rw = "-"
		}
		parts = append(parts, "RT "+rt+"/RW "+rw)
	}
	add(r.Desa)
	add(r.Kecamatan)
	add(r.Kabupaten)
	add(r.Provinsi)
	return strings.Join(parts, ", ")
}

func normalizeKTPData(raw *ktpOCRRaw) *KTPOCRData {
	d := &KTPOCRData{
		NIK:          digitsOnly(raw.NIK),
		NamaLengkap:  strings.TrimSpace(raw.NamaLengkap),
		JenisKelamin: normalizeJenisKelamin(raw.JenisKelamin),
		TanggalLahir: normalizeTanggal(raw.TanggalLahir),
		Agama:        normalizeAgama(raw.Agama),
		Pekerjaan:    strings.TrimSpace(raw.Pekerjaan),
		AlamatAsal:   composeAlamatAsal(raw),
	}
	return d
}

func (h *OCRHandler) OCRKTP(w http.ResponseWriter, r *http.Request) {
	data, mimeType, ok := h.beginScan(w, r, "KTP")
	if !ok {
		return
	}

	rawJSON, err := h.callGeminiJSON(r, mimeType, data, ktpOCRPrompt, ktpOCRSchema())
	if err != nil {
		response.Error(w, http.StatusBadGateway, "Gagal memproses foto KTP: "+err.Error())
		return
	}
	var raw ktpOCRRaw
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		response.Error(w, http.StatusBadGateway, "Hasil pembacaan tidak valid, silakan isi manual")
		return
	}

	result := normalizeKTPData(&raw)
	response.Success(w, http.StatusOK, "Data KTP berhasil dibaca", result)
}
