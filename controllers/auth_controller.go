package controllers

import (
	"fmt"
	"latihan1/models"
	"latihan1/utils" // Pastikan fungsi HashPassword & CheckPasswordHash ada di sini
	"net/http"
	"time"

	"github.com/gorilla/csrf"
	"gorm.io/gorm"
)

type AuthController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

// ShowLogin menampilkan halaman login
func (ac *AuthController) ShowLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ISI TOKEN:", csrf.TemplateField(r))
	data := map[string]interface{}{
		// WAJIB DITAMBAHKAN: Kirim field CSRF ke template login
		"csrfField": csrf.TemplateField(r),
	}
	ac.Render(w, r, "auth/login.html", data)
}

// Login memproses autentikasi user
// Login memproses autentikasi user
func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user models.User
	if err := ac.DB.Where("username = ?", username).First(&user).Error; err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Verifikasi Password menggunakan bcrypt
	if !utils.CheckPasswordHash(password, user.Password) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 🟢 1. Generate JWT Token (Menggantikan ID mentah)
	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		http.Error(w, "Gagal membuat session aman", http.StatusInternalServerError)
		return
	}

	// 🟢 2. Set Session via Cookie dengan pengamanan ekstra
	expiration := time.Now().Add(24 * time.Hour)
	cookie := http.Cookie{
		Name:     "user_session",
		Value:    token, // Sekarang nilainya berupa token acak terenkripsi JWT
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,                 // Mencegah XSS (JavaScript tidak bisa mencuri cookie)
		Secure:   false,                // Set ke 'true' jika kamu sudah pakai HTTPS
		SameSite: http.SameSiteLaxMode, // Melindungi dari serangan CSRF
	}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ShowRegister menampilkan halaman registrasi
func (ac *AuthController) ShowRegister(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"csrfField": csrf.TemplateField(r),
	}
	ac.Render(w, r, "auth/register.html", data)
}

// Register memproses pendaftaran user baru
func (ac *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if password != confirm {
		// Logika handling password tidak cocok
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	// Hash password sebelum simpan
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		http.Error(w, "Gagal memproses password", http.StatusInternalServerError)
		return
	}

	user := models.User{
		Username: username,
		Password: hashedPassword,
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Logout menghapus session
func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "user_session",
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
