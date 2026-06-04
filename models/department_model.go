package models

import (
	"gorm.io/gorm"
)

type Department struct {
	Name  string `gorm:"unique;not null"`
	Users []User `gorm:"foreignKey:DepartmentID"`
	// Tambahkan 2 baris relasi ini:
	Custodians       []Custodian       `gorm:"foreignKey:DepartmentID"`
	DepartmentGroups []DepartmentGroup `gorm:"foreignKey:DepartmentID"`
	gorm.Model                         // Otomatis menambahkan ID, CreatedAt, UpdatedAt, DeletedAt
}

// Fungsi bantu untuk CRUD menggunakan GORM
func GetAllDepartments(db *gorm.DB) ([]Department, error) {
	var departments []Department
	err := db.Find(&departments).Error
	return departments, err
}

func CreateDepartment(db *gorm.DB, name string) error {
	return db.Create(&Department{Name: name}).Error
}

func DeleteDepartment(db *gorm.DB, id string) error {
	// Cukup satu baris ini untuk menghapus permanen
	return db.Unscoped().Delete(&Department{}, id).Error
}
