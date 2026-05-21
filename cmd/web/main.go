package main

import (
	"fmt"
	"latihan1/cmd/web/bootstrap"
	"latihan1/cmd/web/config"
	"latihan1/cmd/web/routes"
	"net/http"
	"os"
)

func main() {

	// Init Database
	db := config.InitDB()

	// Init Controllers
	controllers := bootstrap.InitControllers(db)

	// Register Routes
	routes.RegisterRoutes(db, controllers)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	fmt.Printf("Server is running on http://localhost:%s\n", port)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
