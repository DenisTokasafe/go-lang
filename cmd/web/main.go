package main

import (
	"fmt"
	"latihan1/cmd/web/bootstrap"
	"latihan1/cmd/web/config"
	"latihan1/cmd/web/helpers"
	"latihan1/cmd/web/routes"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv" // Tambahkan library godotenv
)

func main() {

	// PENTING: Load file .env di awal sebelum fungsi lain berjalan
	errEnv := godotenv.Load()
	if errEnv != nil {
		log.Println("Peringatan: File .env tidak ditemukan, menggunakan env system")
	}
	// LOAD TERJEMAHAN YAML DI SINI
	errLang := helpers.LoadTranslations()
	if errLang != nil {
		log.Fatalf("Gagal memuat file terjemahan bahasa: %v", errLang)
	}
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
