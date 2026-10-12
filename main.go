package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"finance-app-be/config"
	"finance-app-be/routes"
)

func main() {
	// Pengecekan JWT_SECRET sebelum koneksi database dan server berjalan
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET wajib di-set")
	}

	config.ConnectDB()

	r := routes.SetupRouter()

	// Ambil PORT dari environment, kalau tidak ada pakai 8080 (lokal)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Konfigurasi Custom HTTP Server dengan Timeout Ketat
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   20 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // Max 1 MB HTTP header size
	}

	log.Printf("Server berjalan secara aman pada port :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}