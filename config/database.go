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

	// Auto Migration: Pastikan tabel & kolom pendukung wujud di PostgreSQL Cloud
	migrationQuery := `
		ALTER TABLE categories ADD COLUMN IF NOT EXISTS budget_limit NUMERIC DEFAULT 0;
		ALTER TABLE categories ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'expense';
		ALTER TABLE transactions ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'expense';
		ALTER TABLE transactions ADD COLUMN IF NOT EXISTS notes TEXT;

		CREATE TABLE IF NOT EXISTS webauthn_credentials (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			credential_id BYTEA NOT NULL UNIQUE,
			public_key BYTEA NOT NULL,
			attestation_type VARCHAR(100) DEFAULT '',
			transport TEXT DEFAULT '',
			sign_count BIGINT DEFAULT 0,
			user_present BOOLEAN DEFAULT TRUE,
			user_verified BOOLEAN DEFAULT TRUE,
			backup_eligible BOOLEAN DEFAULT FALSE,
			backup_state BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS webauthn_sessions (
			challenge_id VARCHAR(255) PRIMARY KEY,
			user_id INT NOT NULL,
			session_data TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL
		);
	`
	_, _ = DB.Exec(migrationQuery)

	fmt.Println("Berhasil terhubung ke database PostgreSQL!")
}