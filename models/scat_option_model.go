package models

import (
	"time"

	"gorm.io/gorm"
)

type ScatOption struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Code string `gorm:"type:varchar(255);not null;uniqueIndex:idx_code_type"`
	Name string `gorm:"type:varchar(255);not null"`
	// Definisi Enum di MySQL melalui tag GORM
	Type      string    `gorm:"type:enum('unsafe_act','personal_factor','job_factor','control_system');not null;uniqueIndex:idx_code_type"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// GetAllScatOptions mengambil semua data SCAT
func GetAllScatOptions(db *gorm.DB) ([]ScatOption, error) {
	var options []ScatOption
	err := db.Find(&options).Error
	return options, err
}

// CreateScatOption membuat data baru
func CreateScatOption(db *gorm.DB, code, name, optionType string) error {
	return db.Create(&ScatOption{
		Code: code,
		Name: name,
		Type: optionType,
	}).Error
}

// UpdateScatOption memperbarui data berdasarkan ID
func UpdateScatOption(db *gorm.DB, id string, code, name, optionType string) error {
	return db.Model(&ScatOption{}).Where("id = ?", id).Updates(ScatOption{
		Code: code,
		Name: name,
		Type: optionType,
	}).Error
}

// DeleteScatOption menghapus data secara permanen (Unscoped)
func DeleteScatOption(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&ScatOption{}, id).Error
}
