package controllers

import (
	"latihan1/models"
	"latihan1/utils" // Pastikan fungsi HashPassword & CheckPasswordHash ada di sini
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type AuthController struct {
	DB     *gorm.DB
	Render func(w http.ResponseWriter, r *http.Request, tmpl string, data interface{})
}

// ShowLogin menampilkan halaman login
func (ac *AuthController) ShowLogin(w http.ResponseWriter, r *http.Request) {
	ac.Render(w, r, "auth/login.html", nil)
}

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
		// Set flash message jika user tidak ditemukan (Implementasikan fungsi setCookie flash jika perlu)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Verifikasi Password menggunakan bcrypt
	if !utils.CheckPasswordHash(password, user.Password) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Set Session via Cookie
	expiration := time.Now().Add(24 * time.Hour)
	cookie := http.Cookie{
		Name:     "user_session",
		Value:    strconv.FormatUint(uint64(user.ID), 10),
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true, // Keamanan ekstra
	}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ShowRegister menampilkan halaman registrasi
func (ac *AuthController) ShowRegister(w http.ResponseWriter, r *http.Request) {
	ac.Render(w, r, "auth/register.html", nil)
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
