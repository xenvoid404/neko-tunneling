package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/pkg/logger"
)

var log = logger.CreateLogger()
var DB *sql.DB

func Connect(ctx context.Context, cfg *config.Config) error {
	dsn := fmt.Sprintf("file:%s?_fk=1&_journal_mode=WAL&_synchronous=NORMAL&_temp_store=MEMORY", cfg.DBPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("gagal membuka database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database tidak merespon: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		return err
	}

	DB = db
	log.Info("Inisiasi database berhasil",
		slog.String("path", cfg.DBPath))
	return nil
}

func Close() error {
	if DB != nil {
		log.Info("Menutup koneksi database...")
		if err := DB.Close(); err != nil {
			log.Error("Gagal menutup koneksi database",
				slog.Any("error", err))
			return
		}
		log.Info("Koneksi database berhasil ditutup")
	}
}

func migrate(ctx context.Context, db *sql.DB) error {
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

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("gagal memuat tabel database: %w", err)
	}

	log.Info("Migrasi tabel database berhasil")
	return nil
}
