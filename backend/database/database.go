package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xenvoid404/neko-tunneling/config"
)

type Database struct {
	*sql.DB
}

func Setup(ctx context.Context, cfg *config.Config) *Database {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	dsn := fmt.Sprintf("file:%s?_fk=1&_journal_mode=WAL&_synchronous=NORMAL&_temp_store=MEMORY", cfg.DBPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		slog.Error("Gagal membuka database",
			slog.Any("error", err))
		os.Exit(1)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.PingContext(dbCtx); err != nil {
		slog.Error("Database tidak merespon",
			slog.Any("error", err))
		os.Exit(1)
	}

	appDB := &Database{db}

	if err := appDB.Migrate(dbCtx); err != nil {
		slog.Error("Gagal melakukan migrasi database",
			slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Inisiasi dan migrasi database berhasil",
		slog.String("path", cfg.DBPath))
	return appDB
}

func (d *Database) Close() error {
	if d == nil || d.DB == nil {
		return nil
	}
	return d.DB.Close()
}

func (d *Database) Migrate(ctx context.Context) error {
	query := `
  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    protocol TEXT NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    limit_ip INTEGER DEFAULT 0 NOT NULL,
    limit_quota INTEGER DEFAULT 0 NOT NULL,
    status TEXT DEFAULT 'active' NOT NULL,
    expired_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );
  CREATE TRIGGER IF NOT EXISTS trg_users_updated_at
  AFTER UPDATE ON users FOR EACH ROW
  BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
  END;
  `

	if _, err := d.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("gagal memuat tabel database: %w", err)
	}

	slog.Info("Migrasi tabel database berhasil")
	return nil
}
