package pengurus

import "time"

type Item struct {
	ID          string    `json:"id"`
	NamaLengkap string    `json:"nama_lengkap"`
	Jabatan     string    `json:"jabatan"`
	Foto        *string   `json:"foto,omitempty"`
	Urutan      int       `json:"urutan"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
