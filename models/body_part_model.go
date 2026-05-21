package models

import (
	"gorm.io/gorm"
)

type BodyPart struct {
	gorm.Model         // Menambahkan ID (bigint), CreatedAt, UpdatedAt, DeletedAt
	Code       string  `gorm:"type:varchar(255);unique;not null" json:"code"`
	Name       string  `gorm:"type:varchar(255);not null" json:"name"`
	NameEn     *string `gorm:"type:varchar(255);default:null" json:"name_en"` // Nullable
	Category   string  `gorm:"type:varchar(255);not null" json:"category"`
}

// Menentukan nama tabel secara eksplisit
func (BodyPart) TableName() string {
	return "body_parts"
}

// --- Fungsi Bantu CRUD ---

// GetAllBodyParts mengambil semua data dengan pengurutan numerik pada kode
func GetAllBodyParts(db *gorm.DB) ([]BodyPart, error) {
	var parts []BodyPart
	err := db.Order("LENGTH(code) ASC, code ASC").Find(&parts).Error
	return parts, err
}

// CreateBodyPart menambahkan data body part baru
func CreateBodyPart(db *gorm.DB, code, name, category string, nameEn *string) error {
	newPart := BodyPart{
		Code:     code,
		Name:     name,
		Category: category,
		NameEn:   nameEn,
	}
	return db.Create(&newPart).Error
}

// UpdateBodyPart mengupdate data body part yang sudah ada
func UpdateBodyPart(db *gorm.DB, id, code, name, category string, nameEn *string) error {
	return db.Model(&BodyPart{}).Where("id = ?", id).Updates(map[string]interface{}{
		"code":     code,
		"name":     name,
		"name_en":  nameEn,
		"category": category,
	}).Error
}

// DeleteBodyPart menghapus data secara permanen berdasarkan ID
func DeleteBodyPart(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&BodyPart{}, id).Error
}

// GetBodyPartBySearch melakukan pencarian berdasarkan nama atau kode
func GetBodyPartBySearch(db *gorm.DB, search string) ([]BodyPart, error) {
	var parts []BodyPart
	query := "%" + search + "%"
	err := db.Where("name LIKE ? OR code LIKE ?", query, query).
		Order("LENGTH(code) ASC, code ASC").
		Find(&parts).Error
	return parts, err
}
