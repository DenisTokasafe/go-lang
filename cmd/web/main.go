package main

import (
	"latihan1/cmd/web/bootstrap"
	"latihan1/cmd/web/config"
	"latihan1/cmd/web/helpers"
	"latihan1/cmd/web/routes"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	// 1. IMPORT PACKAGE GORILLA CSRF DI SINI
	"github.com/gorilla/csrf"
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
	// Semua route Anda terdaftar di dalam DefaultServeMux bawaan Go
	routes.RegisterRoutes(db, controllers)

	// =========================================================================
	// 2. KONFIGURASI MIDDLEWARE GORILLA CSRF
	// =========================================================================

	// Ambil CSRF Key dari .env untuk keamanan produksi, atau gunakan fallback default (wajib 32 byte)
	csrfSecret := os.Getenv("CSRF_SECRET_KEY")
	if len(csrfSecret) != 32 {
		log.Println("Peringatan: CSRF_SECRET_KEY di .env tidak ada atau panjangnya tidak 32 byte. Menggunakan kunci fallback otomatis.")
		csrfSecret = "a-very-secret-key-32-characters" // Tepat 32 karakter
	}

	// Ambil status environment (apakah production / development)
	// Ambil status environment (apakah production / development)
	isProd := os.Getenv("APP_ENV") == "production"

	csrfMiddleware := csrf.Protect(
		[]byte(csrfSecret),
		csrf.Secure(isProd),                 // Jika true (di prod), cookie CSRF hanya dikirim lewat HTTPS.
		csrf.HttpOnly(true),                 // Mencegah cookie dibaca oleh JavaScript jahat
		csrf.SameSite(csrf.SameSiteLaxMode), // Standar keamanan browser modern
		csrf.Path("/"),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 Forbidden - Proteksi Keamanan CSRF: Sesi Anda tidak valid atau telah kedaluwarsa."))
		})),
	)

	log.Println("Aplikasi SENTRY berjalan lokal di http://localhost:8080")

	// =========================================================================
	// 3. JALANKAN SERVER DENGAN MIDDLEWARE
	// =========================================================================
	// Default handler menggunakan Mux standar tanpa proteksi CSRF
	var handler http.Handler = http.DefaultServeMux

	// Jika environment production, bungkus handler dengan middleware CSRF
	if isProd {
		handler = csrfMiddleware(http.DefaultServeMux)
		log.Println("🔒 Proteksi CSRF: AKTIF (Production Mode)")
	} else {
		log.Println("⚠️ Proteksi CSRF: NONAKTIF (Development Mode)")
	}

	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatalf("Gagal menjalankan server lokal: %v\n", err)
	}
}
