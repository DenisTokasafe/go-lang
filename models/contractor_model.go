package models

import (
	"gorm.io/gorm"
)

type Contractor struct {
	Name       string `gorm:"type:varchar(255);unique;not null"`
	Users      []User `gorm:"foreignKey:ContractorID"`
	gorm.Model        // Taruh di atas agar kolom ID, CreatedAt, dll ada di awal tabel

	// Relasi HasMany: Satu kontraktor bisa memiliki banyak User/Karyawan
	// Ini memudahkan jika nanti Anda ingin: db.Preload("Users").First(&contractor)
}

// --- Fungsi bantu untuk CRUD menggunakan GORM ---

func GetAllContractors(db *gorm.DB) ([]Contractor, error) {
	var contractors []Contractor
	// Ditambahkan Order by Name agar tampilan di dropdown/tabel rapi
	err := db.Order("name asc").Find(&contractors).Error
	return contractors, err
}

func CreateContractor(db *gorm.DB, name string) error {
	return db.Create(&Contractor{Name: name}).Error
}

func DeleteContractor(db *gorm.DB, id string) error {
	// Tetap menggunakan Unscoped() untuk hapus permanen sesuai kebutuhan Anda
	return db.Unscoped().Delete(&Contractor{}, id).Error
}

// Fungsi tambahan yang berguna untuk pengecekan data
func GetContractorByID(db *gorm.DB, id uint) (Contractor, error) {
	var contractor Contractor
	err := db.First(&contractor, id).Error
	return contractor, err
}
