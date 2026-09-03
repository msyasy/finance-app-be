package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func ConnectDB() {
	// Ambil dari Environment Variable Render (DATABASE_URL)
	connStr := os.Getenv("DATABASE_URL")

	// Jika DATABASE_URL kosong (saat dijalankan di lokal tanpa .env)
	if connStr == "" {
		connStr = "host=localhost port=5432 user=postgres password=317505 dbname=finance_db sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Gagal membuka koneksi DB:", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("Gagal terhubung ke DB:", err)
	}

	fmt.Println("Berhasil terhubung ke database PostgreSQL!")
}