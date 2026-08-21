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
	maxKKImageSize = 10 << 20 // 10 MB
	geminiTimeout  = 60 * time.Second
)

var allowedKKImageTypes = map[string]string{
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
		model = "gemini-2.5-flash"
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
	NomorKK   string       `json:"nomor_kk"`
	NomorHP   string       `json:"nomor_hp"`
	Alamat    string       `json:"alamat"`
	RT        string       `json:"rt"`
	RW        string       `json:"rw"`
	Desa      string       `json:"desa"`
	Kecamatan string       `json:"kecamatan"`
	Kabupaten string       `json:"kabupaten"`
	Provinsi  string       `json:"provinsi"`
	Kepala    *KKOCRKepala `json:"kepala_keluarga,omitempty"`
}

type KKOCRKepala struct {
	NIK          string `json:"nik"`
	NamaLengkap  string `json:"nama_lengkap"`
	TempatLahir  string `json:"tempat_lahir"`
	TanggalLahir string `json:"tanggal_lahir"`
	JenisKelamin string `json:"jenis_kelamin"`
	Agama        string `json:"agama"`
	Pekerjaan    string `json:"pekerjaan"`
}

type geminiInline struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiPart struct {
	Text       string       `json:"text,omitempty"`
	InlineData *geminiInline `json:"inline_data,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature     float64         `json:"temperature"`
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
- nomor_kk dan nik harus berupa 16 digit angka tanpa spasi atau tanda hubung.
- tanggal_lahir dalam format YYYY-MM-DD.
- jenis_kelamin diisi "L" untuk LAKI-LAKI dan "P" untuk PEREMPUAN.
- agama salah satu dari: Islam, Kristen, Katolik, Hindu, Buddha, Konghucu.
- rt dan rw hanya angkanya saja, tanpa kata "RT"/"RW".
- pekerjaan dan nomor_hp boleh kosong jika tidak tercantum.
- Jika suatu data tidak terbaca dengan jelas atau tidak ada di dokumen, isi string kosong.
- Jangan pernah mengarang atau menebak data yang tidak terlihat.`

func kkOCRSchema() json.RawMessage {
	str := map[string]interface{}{"type": "string"}
	props := func(p map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"type": "object", "properties": p}
	}
	schema := props(map[string]interface{}{
		"nomor_kk":   str,
		"nomor_hp":   str,
		"alamat":     str,
		"rt":         str,
		"rw":         str,
		"desa":       str,
		"kecamatan":  str,
		"kabupaten":  str,
		"provinsi":   str,
		"kepala_keluarga": props(map[string]interface{}{
			"nik":           str,
			"nama_lengkap":  str,
			"tempat_lahir":  str,
			"tanggal_lahir": str,
			"jenis_kelamin": str,
			"agama":         str,
			"pekerjaan":     str,
		}),
	})
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
	if d.Kepala != nil {
		k := d.Kepala
		k.NIK = digitsOnly(k.NIK)
		k.NamaLengkap = strings.TrimSpace(k.NamaLengkap)
		k.TempatLahir = strings.TrimSpace(k.TempatLahir)
		k.TanggalLahir = normalizeTanggal(k.TanggalLahir)
		k.JenisKelamin = normalizeJenisKelamin(k.JenisKelamin)
		k.Agama = normalizeAgama(k.Agama)
		k.Pekerjaan = strings.TrimSpace(k.Pekerjaan)
		if k.NIK == "" && k.NamaLengkap == "" {
			d.Kepala = nil
		}
	}
}

func (h *OCRHandler) OCRKK(w http.ResponseWriter, r *http.Request) {
	if h.apiKey == "" {
		response.Error(w, http.StatusServiceUnavailable, "Fitur scan KK belum dikonfigurasi (GEMINI_API_KEY kosong)")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxKKImageSize+(1<<20))
	if err := r.ParseMultipartForm(maxKKImageSize); err != nil {
		response.Error(w, http.StatusBadRequest, "Foto KK wajib diunggah (maksimal 10 MB)")
		return
	}
	file, header, err := r.FormFile("foto")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Foto KK wajib diunggah")
		return
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, _ := io.ReadFull(file, sniff)
	sniffed := http.DetectContentType(sniff[:n])
	mimeType := allowedKKImageTypes[sniffed]
	if mimeType == "" {
		mimeType = allowedKKImageTypes[header.Header.Get("Content-Type")]
	}
	if mimeType == "" {
		response.Error(w, http.StatusUnsupportedMediaType, "Format foto harus JPG, PNG, WEBP, atau HEIC")
		return
	}

	data, err := io.ReadAll(io.MultiReader(bytes.NewReader(sniff[:n]), file))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Gagal membaca file foto")
		return
	}
	if len(data) > maxKKImageSize {
		response.Error(w, http.StatusBadRequest, "Ukuran foto maksimal 10 MB")
		return
	}

	result, err := h.callGemini(r, mimeType, data)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "Gagal memproses foto KK: "+err.Error())
		return
	}

	normalizeOCRData(result)
	response.Success(w, http.StatusOK, "Data Kartu Keluarga berhasil dibaca", result)
}

func (h *OCRHandler) callGemini(r *http.Request, mimeType string, image []byte) (*KKOCRData, error) {
	payload := geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{
				{InlineData: &geminiInline{MimeType: mimeType, Data: base64.StdEncoding.EncodeToString(image)}},
				{Text: kkOCRPrompt},
			},
		}},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0,
			MaxOutputTokens: 4096,
			ResponseMIME:    "application/json",
			ResponseSchema:  kkOCRSchema(),
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
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("terlalu banyak permintaan scan, coba lagi sebentar")
		}
		return nil, fmt.Errorf("layanan OCR menolak permintaan")
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

	var result KKOCRData
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("hasil pembacaan tidak valid, silakan isi manual")
	}
	return &result, nil
}
