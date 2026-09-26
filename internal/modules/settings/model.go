package settings

import (
	"bytes"
	"encoding/json"
)

// OptionalFloat membedakan tiga kondisi: field tidak dikirim (biarkan),
// dikirim null atau string kosong (kosongkan), dan dikirim angka (simpan).
type OptionalFloat struct {
	Value *float64
}

func (o *OptionalFloat) UnmarshalJSON(data []byte) error {
	raw := string(bytes.TrimSpace(data))
	if raw == "null" || raw == `""` {
		o.Value = nil
		return nil
	}
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	o.Value = &f
	return nil
}

type WebsiteSettingsResponse struct {
	ID             string   `json:"id"`
	NamaWebsite    string   `json:"nama_website"`
	NamaRT         string   `json:"nama_rt"`
	Desa           string   `json:"desa"`
	Kecamatan      string   `json:"kecamatan"`
	Kabupaten      string   `json:"kabupaten"`
	Provinsi       string   `json:"provinsi"`
	Telepon        string   `json:"telepon"`
	Email          string   `json:"email"`
	Alamat         string   `json:"alamat"`
	MapsLat        *float64 `json:"maps_lat"`
	MapsLng        *float64 `json:"maps_lng"`
	Facebook       string   `json:"facebook"`
	Instagram      string   `json:"instagram"`
	Whatsapp       string   `json:"whatsapp"`
	Youtube        string   `json:"youtube"`
	Tiktok         string   `json:"tiktok"`
	LogoPath       string   `json:"logo_path"`
	Copyright      string   `json:"copyright"`
	Visi           string   `json:"visi"`
	Misi           []string `json:"misi"`
	ShowPengumuman bool     `json:"show_pengumuman"`
	ShowLaporan    bool     `json:"show_laporan"`
	ShowPengaduan  bool     `json:"show_pengaduan"`
	ShowVisiMisi   bool     `json:"show_visi_misi"`
	ShowPeraturan  bool     `json:"show_peraturan"`
}

type UpdateSettingsRequest struct {
	NamaWebsite    *string        `json:"nama_website,omitempty"`
	NamaRT         *string        `json:"nama_rt,omitempty"`
	Desa           *string        `json:"desa,omitempty"`
	Kecamatan      *string        `json:"kecamatan,omitempty"`
	Kabupaten      *string        `json:"kabupaten,omitempty"`
	Provinsi       *string        `json:"provinsi,omitempty"`
	Telepon        *string        `json:"telepon,omitempty"`
	Email          *string        `json:"email,omitempty"`
	Alamat         *string        `json:"alamat,omitempty"`
	MapsLat        *OptionalFloat `json:"maps_lat,omitempty"`
	MapsLng        *OptionalFloat `json:"maps_lng,omitempty"`
	Facebook       *string        `json:"facebook,omitempty"`
	Instagram      *string        `json:"instagram,omitempty"`
	Whatsapp       *string        `json:"whatsapp,omitempty"`
	Youtube        *string        `json:"youtube,omitempty"`
	Tiktok         *string        `json:"tiktok,omitempty"`
	Copyright      *string        `json:"copyright,omitempty"`
	Visi           *string        `json:"visi,omitempty"`
	Misi           *[]string      `json:"misi,omitempty"`
	ShowPengumuman *bool          `json:"show_pengumuman,omitempty"`
	ShowLaporan    *bool          `json:"show_laporan,omitempty"`
	ShowPengaduan  *bool          `json:"show_pengaduan,omitempty"`
	ShowVisiMisi   *bool          `json:"show_visi_misi,omitempty"`
	ShowPeraturan  *bool          `json:"show_peraturan,omitempty"`
}
