package models

import (
	"gorm.io/gorm"
)

type Company struct {
	gorm.Model        // Otomatis menambahkan ID, CreatedAt, UpdatedAt, DeletedAt
	Name       string `gorm:"unique;not null"`
}

// Fungsi bantu untuk CRUD menggunakan GORM
func GetAllCompanies(db *gorm.DB) ([]Company, error) {
	var companies []Company
	err := db.Find(&companies).Error
	return companies, err
}

func CreateCompany(db *gorm.DB, name string) error {
	return db.Create(&Company{Name: name}).Error
}

func DeleteCompany(db *gorm.DB, id string) error {
	// Cukup satu baris ini untuk menghapus permanen
	return db.Unscoped().Delete(&Company{}, id).Error
}
