package seeders

import (
	"fmt"
	"latihan1/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedAdminUser(db *gorm.DB) error {

	// =========================
	// GET ROLE
	// =========================
	var role models.Role

	if err := db.Where("name = ?", "administrator").First(&role).Error; err != nil {
		return fmt.Errorf("role administrator tidak ditemukan: %w", err)
	}

	// =========================
	// GET DEPARTMENT
	// =========================
	var department models.Department

	if err := db.Where("name = ?", "HSE & Formalities").First(&department).Error; err != nil {
		return fmt.Errorf("department HSE & Formalities tidak ditemukan: %w", err)
	}

	// =========================
	// CHECK USER EXIST
	// =========================
	var existingUser models.User

	err := db.Where("email = ?", "yoman.banea@archimining.com").
		First(&existingUser).Error

	if err == nil {
		fmt.Println("User admin sudah ada")
		return nil
	}

	// =========================
	// HASH PASSWORD
	// =========================
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte("Denis28%%"),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return fmt.Errorf("gagal hash password: %w", err)
	}

	// =========================
	// CREATE USER
	// =========================
	user := models.User{
		Name:         "Banea, Yoman Denis",
		Username:     "yoman.banea",
		Email:        "yoman.banea@archimining.com",
		Password:     string(hashedPassword),
		RoleID:       role.ID,
		DepartmentID: &department.ID,
		IsPIC:        false,
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("gagal membuat admin user: %w", err)
	}

	fmt.Println("Seeder admin user berhasil dijalankan")

	return nil
}
