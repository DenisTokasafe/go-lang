package middlewares

import (
	"context"
	"latihan1/models"
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

		// AMBIL DATA USER LENGKAP DENGAN ROLE-NYA
		var user models.User
		// Preload("Role") adalah kunci agar Role.Name tidak kosong
		err = db.Preload("Role").Where("id = ?", cookie.Value).First(&user).Error

		if err != nil {
			// Jika user tidak ditemukan di DB, hapus cookie dan redirect
			http.SetCookie(w, &http.Cookie{Name: "user_session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Simpan seluruh objek user ke context
		ctx := context.WithValue(r.Context(), AuthUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
