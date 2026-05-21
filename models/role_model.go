package models

import (
	"gorm.io/gorm"
)

type Role struct {
	gorm.Model        // Taruh di atas agar ID menjadi kolom pertama
	Name       string `gorm:"type:varchar(50);unique;not null"`

	// Relasi HasMany: Satu Role bisa dimiliki banyak User
	Users []User `gorm:"foreignKey:RoleID"`
}

// --- Fungsi Bantu CRUD ---

func GetAllRoles(db *gorm.DB) ([]Role, error) {
	var roles []Role
	// Menambahkan pengurutan berdasarkan nama agar rapi di UI
	err := db.Order("name asc").Find(&roles).Error
	return roles, err
}

func CreateRole(db *gorm.DB, name string) error {
	return db.Create(&Role{Name: name}).Error
}

func DeleteRole(db *gorm.DB, id string) error {
	// Menggunakan Unscoped() seperti permintaan Anda untuk menghapus permanen (Hard Delete)
	return db.Unscoped().Delete(&Role{}, id).Error
}

// Fungsi tambahan: Mencari Role berdasarkan nama (berguna untuk validasi/default role)
func GetRoleByName(db *gorm.DB, name string) (Role, error) {
	var role Role
	err := db.Where("name = ?", name).First(&role).Error
	return role, err
}
