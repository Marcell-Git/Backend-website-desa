package settings

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
	LogoPath       string   `json:"logo_path"`
	Copyright      string   `json:"copyright"`
	Visi           string   `json:"visi"`
	Misi           []string `json:"misi"`
	ShowPengumuman bool     `json:"show_pengumuman"`
	ShowLaporan    bool     `json:"show_laporan"`
	ShowPengaduan  bool     `json:"show_pengaduan"`
	ShowVisiMisi   bool     `json:"show_visi_misi"`
}

type UpdateSettingsRequest struct {
	NamaWebsite    *string   `json:"nama_website,omitempty"`
	NamaRT         *string   `json:"nama_rt,omitempty"`
	Desa           *string   `json:"desa,omitempty"`
	Kecamatan      *string   `json:"kecamatan,omitempty"`
	Kabupaten      *string   `json:"kabupaten,omitempty"`
	Provinsi       *string   `json:"provinsi,omitempty"`
	Telepon        *string   `json:"telepon,omitempty"`
	Email          *string   `json:"email,omitempty"`
	Alamat         *string   `json:"alamat,omitempty"`
	Copyright      *string   `json:"copyright,omitempty"`
	Visi           *string   `json:"visi,omitempty"`
	Misi           *[]string `json:"misi,omitempty"`
	ShowPengumuman *bool     `json:"show_pengumuman,omitempty"`
	ShowLaporan    *bool     `json:"show_laporan,omitempty"`
	ShowPengaduan  *bool     `json:"show_pengaduan,omitempty"`
	ShowVisiMisi   *bool     `json:"show_visi_misi,omitempty"`
}
