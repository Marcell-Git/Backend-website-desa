package peraturan

import "time"

type Item struct {
	ID         string    `json:"id"`
	Judul      string    `json:"judul"`
	FilePDF    string    `json:"file_pdf"`
	DibuatOleh string    `json:"dibuat_oleh"`
	NamaUser   string    `json:"nama_user,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PublicItem struct {
	ID        string    `json:"id"`
	Judul     string    `json:"judul"`
	FilePDF   string    `json:"file_pdf"`
	CreatedAt time.Time `json:"created_at"`
}
