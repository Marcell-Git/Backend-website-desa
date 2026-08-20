package database

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RecordAudit(ctx context.Context, pool *pgxpool.Pool, userID *string, modul, aktivitas, deskripsi string, before, after interface{}) {
	var beforeJSON, afterJSON []byte

	if before != nil {
		b, err := json.Marshal(before)
		if err != nil {
			b = []byte("null")
		}
		beforeJSON = b
	} else {
		beforeJSON = []byte("null")
	}

	if after != nil {
		a, err := json.Marshal(after)
		if err != nil {
			a = []byte("null")
		}
		afterJSON = a
	} else {
		afterJSON = []byte("null")
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO audit_logs (user_id, modul, aktivitas, deskripsi, data_sebelum, data_sesudah, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, modul, aktivitas, deskripsi, beforeJSON, afterJSON, "", "",
	)
	_ = err
}
