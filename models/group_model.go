package models

import (
	"gorm.io/gorm"
)

type Group struct {
	Name string `gorm:"unique;not null"`
	gorm.Model
}

// Fungsi bantu untuk CRUD menggunakan GORM
func GetAllGroups(db *gorm.DB) ([]Group, error) {
	var groups []Group
	err := db.Find(&groups).Error
	return groups, err
}

func CreateGroup(db *gorm.DB, name string) error {
	return db.Create(&Group{Name: name}).Error
}

func DeleteGroup(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&Group{}, id).Error
}
