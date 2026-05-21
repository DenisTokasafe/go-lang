package middlewares

import (
	"latihan1/models"
	"net/http"
)

// RoleMiddleware membatasi akses berdasarkan nama role
func RoleMiddleware(targetRole string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ambil user dari context (yang sudah diisi oleh AuthMiddleware)
		userVal := r.Context().Value(AuthUserKey)
		if userVal == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user := userVal.(models.User)

		// Cek apakah nama role sesuai
		// Kita gunakan LOWER agar lebih aman dari kesalahan penulisan kapital
		if user.Role.Name != targetRole {
			// Jika bukan admin, lempar ke dashboard atau tampilkan 403 Forbidden
			http.Error(w, "403 Forbidden: Anda tidak memiliki akses ke halaman ini.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}
