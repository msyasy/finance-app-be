package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func ConnectDB() {
	// Ambil URI database dari Environment Variable
	connStr := os.Getenv("DATABASE_URL")

	// Jika di lokal dan DATABASE_URL tidak ada, gunakan localhost
	if connStr == "" {
		connStr = "host=localhost port=5432 user=postgres password=317505 dbname=finance_db sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Gagal membuka koneksi DB:", err)
	}

	// Konfigurasi Pool Connection untuk stabilitas di Cloud/Railway
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	err = DB.Ping()
	if err != nil {
		log.Fatal("Gagal terhubung ke DB:", err)
	}

	// Auto Migration: Pastikan kolom pendukung wujud di PostgreSQL Cloud
	migrationQuery := `
		ALTER TABLE categories ADD COLUMN IF NOT EXISTS budget_limit NUMERIC DEFAULT 0;
		ALTER TABLE categories ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'expense';
		ALTER TABLE transactions ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'expense';
		ALTER TABLE transactions ADD COLUMN IF NOT EXISTS notes TEXT;
	`
	_, _ = DB.Exec(migrationQuery)

	fmt.Println("Berhasil terhubung ke database PostgreSQL!")
}