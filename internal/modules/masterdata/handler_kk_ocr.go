package masterdata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"rt26/backend/internal/pkg/response"
)

const (
	maxOCRImageSize = 10 << 20 // 10 MB
	geminiTimeout   = 60 * time.Second
)

var allowedOCRImageTypes = map[string]string{
	"image/jpeg": "image/jpeg",
	"image/png":  "image/png",
	"image/webp": "image/webp",
	"image/heic": "image/heic",
	"image/heif": "image/heif",
}

type OCRHandler struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOCRHandler(apiKey, model string) *OCRHandler {
	if model == "" {
		model = "gemini-3.6-flash"
	}
	return &OCRHandler{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: geminiTimeout},
	}
}

// KKOCRData adalah hasil pembacaan foto KK yang dikirim ke frontend
// untuk mengisi form secara otomatis.
type KKOCRData struct {
	NomorKK   string         `json:"nomor_kk"`
	NomorHP   string         `json:"nomor_hp"`
	Alamat    string         `json:"alamat"`
	RT        string         `json:"rt"`
	RW        string         `json:"rw"`
	Desa      string         `json:"desa"`
	Kecamatan string         `json:"kecamatan"`
	Kabupaten string         `json:"kabupaten"`
	Provinsi  string         `json:"provinsi"`
	Anggota   []KKOCRAnggota `json:"anggota,omitempty"`
}

type KKOCRAnggota struct {
	NIK              string `json:"nik"`
	NamaLengkap      string `json:"nama_lengkap"`
	TempatLahir      string `json:"tempat_lahir"`
	TanggalLahir     string `json:"tanggal_lahir"`
	JenisKelamin     string `json:"jenis_kelamin"`
	Agama            string `json:"agama"`
	Pekerjaan        string `json:"pekerjaan"`
	HubunganKeluarga string `json:"hubungan_keluarga"`
}

type geminiInline struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiPart struct {
	Text       string        `json:"text,omitempty"`
	InlineData *geminiInline `json:"inline_data,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int32           `json:"maxOutputTokens"`
	ResponseMIME    string          `json:"responseMimeType"`
	ResponseSchema  json.RawMessage `json:"responseSchema"`
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const kkOCRPrompt = `Anda adalah asisten yang mengekstrak data dari foto Kartu Keluarga (KK) Republik Indonesia.
Baca seluruh teks pada gambar dan ekstrak datanya ke dalam JSON sesuai skema yang diberikan.

Aturan:
- Ekstrak SEMUA orang yang tercantum di dokumen ke dalam array "anggota", satu objek per baris nama.
- hubungan_keluarga untuk kepala keluarga diisi tepat "Kepala Keluarga". Untuk anggota lain gunakan
  salah satu dari: Suami, Istri, Anak, Menantu, Cucu, Orang Tua, Mertua, Famili Lain, Pembantu, Lainnya.
- nomor_kk dan nik harus berupa 16 digit angka tanpa spasi atau tanda hubung.
- tanggal_lahir dalam format YYYY-MM-DD.
- jenis_kelamin diisi "L" untuk LAKI-LAKI dan "P" untuk PEREMPUAN.
- agama salah satu dari: Islam, Kristen, Katolik, Hindu, Buddha, Konghucu.
- rt dan rw hanya angkanya saja, tanpa kata "RT"/"RW".
- pekerjaan boleh kosong jika tidak tercantum.
- Jika suatu data tidak terbaca dengan jelas atau tidak ada di dokumen, isi string kosong.
- Jangan pernah mengarang atau menebak data yang tidak terlihat.`

func kkOCRSchema() json.RawMessage {
	str := map[string]interface{}{"type": "string"}
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"nomor_kk":  str,
			"nomor_hp":  str,
			"alamat":    str,
			"rt":        str,
			"rw":        str,
			"desa":      str,
			"kecamatan": str,
			"kabupaten": str,
			"provinsi":  str,
			"anggota": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"nik":               str,
						"nama_lengkap":      str,
						"tempat_lahir":      str,
						"tanggal_lahir":     str,
						"jenis_kelamin":     str,
						"agama":             str,
						"pekerjaan":         str,
						"hubungan_keluarga": str,
					},
					"required": []string{"nama_lengkap"},
				},
			},
		},
	}
	b, _ := json.Marshal(schema)
	return b
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func truncateReason(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "..."
}

func normalizeJenisKelamin(s string) string {
	u := strings.ToUpper(strings.TrimSpace(s))
	switch {
	case u == "P" || strings.Contains(u, "PEREMPUAN") || strings.Contains(u, "WANITA"):
		return "P"
	case u == "L" || strings.Contains(u, "LAKI") || strings.Contains(u, "PRIA"):
		return "L"
	default:
		return ""
	}
}

var agamaList = []string{"Islam", "Kristen", "Katolik", "Hindu", "Buddha", "Konghucu"}

func normalizeAgama(s string) string {
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "" {
		return ""
	}
	for _, a := range agamaList {
		if low == strings.ToLower(a) || strings.Contains(low, strings.ToLower(a)) {
			return a
		}
	}
	return ""
}

func normalizeTanggal(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	layouts := []string{"2006-01-02", "02-01-2006", "02/01/2006", "2006/01/02", "2 January 2006", "02 January 2006"}
	for _, l := range layouts {
		if parsed, err := time.Parse(l, t); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return ""
}

var hubunganList = []string{
	"Kepala Keluarga", "Suami", "Istri", "Anak", "Menantu",
	"Cucu", "Orang Tua", "Mertua", "Famili Lain", "Pembantu", "Lainnya",
}

func normalizeHubungan(s string) string {
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "" {
		return ""
	}
	for _, h := range hubunganList {
		hl := strings.ToLower(h)
		if low == hl || strings.Contains(low, hl) || strings.Contains(hl, low) {
			return h
		}
	}
	// KK sering menulis "KEPALA KELUARGA" atau "KEPALA" — sudah tercakup di atas.
	return "Lainnya"
}

func normalizeOCRData(d *KKOCRData) {
	d.NomorKK = digitsOnly(d.NomorKK)
	d.NomorHP = strings.TrimSpace(d.NomorHP)
	d.Alamat = strings.TrimSpace(d.Alamat)
	d.RT = digitsOnly(d.RT)
	d.RW = digitsOnly(d.RW)
	d.Desa = strings.TrimSpace(d.Desa)
	d.Kecamatan = strings.TrimSpace(d.Kecamatan)
	d.Kabupaten = strings.TrimSpace(d.Kabupaten)
	d.Provinsi = strings.TrimSpace(d.Provinsi)

	cleaned := make([]KKOCRAnggota, 0, len(d.Anggota))
	seenNIK := map[string]bool{}
	for _, a := range d.Anggota {
		a.NIK = digitsOnly(a.NIK)
		a.NamaLengkap = strings.TrimSpace(a.NamaLengkap)
		a.TempatLahir = strings.TrimSpace(a.TempatLahir)
		a.TanggalLahir = normalizeTanggal(a.TanggalLahir)
		a.JenisKelamin = normalizeJenisKelamin(a.JenisKelamin)
		a.Agama = normalizeAgama(a.Agama)
		a.Pekerjaan = strings.TrimSpace(a.Pekerjaan)
		a.HubunganKeluarga = normalizeHubungan(a.HubunganKeluarga)
		if a.NamaLengkap == "" {
			continue
		}
		key := a.NIK + "|" + strings.ToLower(a.NamaLengkap)
		if seenNIK[key] {
			continue
		}
		seenNIK[key] = true
		cleaned = append(cleaned, a)
	}
	d.Anggota = cleaned
}

// readImage memvalidasi & membaca file foto dari multipart form.
// Mengembalikan (data, mimeType, true) atau menuliskan response error lalu (nil, "", false).
func readImage(w http.ResponseWriter, r *http.Request, label string) ([]byte, string, bool) {
	if err := r.ParseMultipartForm(maxOCRImageSize); err != nil {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("Foto %s wajib diunggah (maksimal 10 MB)", label))
		return nil, "", false
	}
	file, header, err := r.FormFile("foto")
	if err != nil {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("Foto %s wajib diunggah", label))
		return nil, "", false
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, _ := io.ReadFull(file, sniff)
	sniffed := http.DetectContentType(sniff[:n])
	mimeType := allowedOCRImageTypes[sniffed]
	if mimeType == "" {
		mimeType = allowedOCRImageTypes[header.Header.Get("Content-Type")]
	}
	if mimeType == "" {
		response.Error(w, http.StatusUnsupportedMediaType, "Format foto harus JPG, PNG, WEBP, atau HEIC")
		return nil, "", false
	}

	data, err := io.ReadAll(io.MultiReader(bytes.NewReader(sniff[:n]), file))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Gagal membaca file foto")
		return nil, "", false
	}
	if len(data) > maxOCRImageSize {
		response.Error(w, http.StatusBadRequest, "Ukuran foto maksimal 10 MB")
		return nil, "", false
	}
	return data, mimeType, true
}

func (h *OCRHandler) OCRKK(w http.ResponseWriter, r *http.Request) {
	data, mimeType, ok := h.beginScan(w, r, "KK")
	if !ok {
		return
	}

	raw, err := h.callGeminiJSON(r, mimeType, data, kkOCRPrompt, kkOCRSchema())
	if err != nil {
		response.Error(w, http.StatusBadGateway, "Gagal memproses foto KK: "+err.Error())
		return
	}
	var result KKOCRData
	if err := json.Unmarshal(raw, &result); err != nil {
		response.Error(w, http.StatusBadGateway, "Hasil pembacaan tidak valid, silakan isi manual")
		return
	}

	normalizeOCRData(&result)
	response.Success(w, http.StatusOK, "Data Kartu Keluarga berhasil dibaca", result)
}

// beginScan memeriksa konfigurasi OCR lalu membaca foto dari request.
func (h *OCRHandler) beginScan(w http.ResponseWriter, r *http.Request, label string) ([]byte, string, bool) {
	if h.apiKey == "" {
		response.Error(w, http.StatusServiceUnavailable, fmt.Sprintf("Fitur scan %s belum dikonfigurasi (GEMINI_API_KEY kosong)", label))
		return nil, "", false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxOCRImageSize+(1<<20))
	return readImage(w, r, label)
}

// callGeminiJSON mengirim foto + prompt ke Gemini dengan structured output,
// mengembalikan teks JSON mentah dari model.
func (h *OCRHandler) callGeminiJSON(r *http.Request, mimeType string, image []byte, prompt string, schema json.RawMessage) ([]byte, error) {
	payload := geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{
				{InlineData: &geminiInline{MimeType: mimeType, Data: base64.StdEncoding.EncodeToString(image)}},
				{Text: prompt},
			},
		}},
		GenerationConfig: geminiGenerationConfig{
			MaxOutputTokens: 8192,
			ResponseMIME:    "application/json",
			ResponseSchema:  schema,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("gagal menyiapkan request")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", h.model)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gagal menyiapkan request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tidak dapat menghubungi layanan OCR")
	}
	defer resp.Body.Close()

	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("respon OCR tidak valid")
	}
	if gr.Error != nil {
		reason := truncateReason(gr.Error.Message, 160)
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("terlalu banyak permintaan scan, coba lagi sebentar")
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, fmt.Errorf("API key OCR tidak valid atau tidak memiliki akses: %s", reason)
		default:
			if reason != "" {
				return nil, fmt.Errorf("layanan OCR menolak permintaan: %s", reason)
			}
			return nil, fmt.Errorf("layanan OCR menolak permintaan")
		}
	}
	if len(gr.Candidates) == 0 {
		return nil, fmt.Errorf("foto tidak dapat dikenali, coba ambil foto yang lebih jelas")
	}

	var text strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	raw := strings.TrimSpace(text.String())
	if raw == "" {
		return nil, fmt.Errorf("data tidak terbaca dari foto, coba ambil foto yang lebih jelas")
	}
	return []byte(raw), nil
}
