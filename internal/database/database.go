package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

type Database struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*Database, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat koneksi database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}
	return &Database{Pool: pool}, nil
}

func (d *Database) RunMigrations(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("gagal membuat tabel schema_migrations: %w", err)
	}

	files, err := fs.Glob(embeddedMigrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		match := regexp.MustCompile(`migrations/(\d+)_.*\.sql`).FindStringSubmatch(file)
		if match == nil {
			continue
		}
		version := 0
		fmt.Sscanf(match[1], "%d", &version)

		var exists bool
		err := d.Pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version,
		).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		content, err := embeddedMigrations.ReadFile(file)
		if err != nil {
			return err
		}

		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migrasi %s gagal: %w", file, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}
		fmt.Printf("migrasi diterapkan: %s\n", file)
	}
	return nil
}

func (d *Database) Close() {
	d.Pool.Close()
}
