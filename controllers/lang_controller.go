package controllers

import (
	"net/http"
	"time"
)

// SetLanguage menyimpan preferensi bahasa user ke dalam Cookie
func SetLanguage(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter lang dari URL (contoh: /set-lang?lang=en)
	lang := r.URL.Query().Get("lang")

	// Validasi agar hanya 'id' atau 'en' yang diterima
	if lang != "en" && lang != "id" {
		lang = "id" // Fallback ke default
	}

	// Buat cookie dengan masa berlaku 30 hari
	http.SetCookie(w, &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Path:     "/",  // Berlaku untuk semua rute
		HttpOnly: true, // Keamanan ekstra
	})

	// Redirect kembali ke halaman sebelumnya (Referer)
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	http.Redirect(w, r, referer, http.StatusFound)
}
