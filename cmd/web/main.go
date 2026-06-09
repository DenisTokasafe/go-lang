package main

import (
	"latihan1/cmd/web/bootstrap"
	"latihan1/cmd/web/config"
	"latihan1/cmd/web/helpers"
	"latihan1/cmd/web/routes"
	"log"
	"net/http/fcgi" // 1. IMPORT PACKAGE FASTCGI INI

	"github.com/joho/godotenv"
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
	// Karena routes Anda terdaftar ke default mux, fcgi akan otomatis membacanya
	routes.RegisterRoutes(db, controllers)

	log.Println("Aplikasi Golang berjalan menggunakan FastCGI di Hostinger...")

	// 2. GANTI LISTENANDSERVE DENGAN INI
	err := fcgi.Serve(nil, nil)
	if err != nil {
		log.Fatalf("Gagal menjalankan FastCGI server: %v\n", err)
	}
}
