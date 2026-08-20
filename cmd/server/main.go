package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"rt26/backend/internal/config"
	"rt26/backend/internal/database"
	"rt26/backend/internal/router"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.RunMigrations(ctx); err != nil {
		log.Fatal(err)
	}
	if err := db.Seed(ctx); err != nil {
		log.Fatal(err)
	}

	handler := router.New(db.Pool, cfg)

	addr := ":" + cfg.AppPort
	fmt.Fprintf(os.Stdout, "Sistem Informasi RT Tegalyoso 26 — backend berjalan di http://localhost%s\n", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
