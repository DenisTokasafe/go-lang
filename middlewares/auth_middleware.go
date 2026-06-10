package middlewares

import (
	"context"
	"latihan1/models"
	"latihan1/utils" // 🟢 Pastikan mengimport package utils tempat ValidateJWT berada
	"net/http"

	"gorm.io/gorm"
)

// Gunakan type custom untuk key context agar lebih aman (Best Practice Go)
type contextKey string

const AuthUserKey contextKey = "authUser"

func AuthMiddleware(db *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_session")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// 🟢 1. VALIDASI JWT TERLEBIH DAHULU
		// Kita cek apakah tokennya asli, palsu, atau sudah expired
		claims, err := utils.ValidateJWT(cookie.Value)
		if err != nil {
			// Jika token bermasalah/dibuat-buat oleh hacker, langsung hapus cookie dan usir ke /login
			http.SetCookie(w, &http.Cookie{Name: "user_session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// AMBIL DATA USER LENGKAP DENGAN ROLE-NYA
		var user models.User

		// 🟢 2. GUNAKAN ID DARI JWT CLAIMS
		// Sekarang kita cari berdasarkan claims.UserID yang sudah terverifikasi aman, bukan cookie.Value lagi
		err = db.Preload("Role").Where("id = ?", claims.UserID).First(&user).Error

		if err != nil {
			// Jika user tiba-tiba dihapus dari DB tapi tokennya masih ada
			http.SetCookie(w, &http.Cookie{Name: "user_session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Simpan seluruh objek user ke context (Ini sudah sangat bagus!)
		ctx := context.WithValue(r.Context(), AuthUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
