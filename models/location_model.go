package models

import (
	"gorm.io/gorm"
)

type Location struct {
	Name string `gorm:"unique;not null"`
	gorm.Model
}

// Fungsi bantu untuk CRUD menggunakan GORM
func GetAllLocations(db *gorm.DB) ([]Role, error) {
	var locations []Role
	err := db.Find(&locations).Error
	return locations, err
}

func CreateLocation(db *gorm.DB, name string) error {
	return db.Create(&Location{Name: name}).Error
}

func DeleteLocation(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&Location{}, id).Error
}
