package main

import (
	"os"
	"finance-app-be/config"
	"finance-app-be/routes"
)

func main() {
	config.ConnectDB()

	r := routes.SetupRouter()

	// Ambil PORT dari Railway, kalau tidak ada pakai 8080 (lokal)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}