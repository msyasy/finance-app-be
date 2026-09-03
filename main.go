package main

import (
	"finance-app-be/config"
	"finance-app-be/routes"
)

func main() {
	config.ConnectDB()

	r := routes.SetupRouter()

	r.Run(":8080")
}