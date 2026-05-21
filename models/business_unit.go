package models

import (
	"gorm.io/gorm"
)

type BusinessUnit struct {
	Name      string `gorm:"type:varchar(255);not null" json:"name"`
	CompanyID uint   `gorm:"not null" json:"company_id"`
	// Relasi Belongs To ke Company
	Company Company `gorm:"foreignKey:CompanyID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"company"`
	gorm.Model
}

// Mendapatkan semua Business Unit (beserta data Company-nya jika perlu)
func GetAllBusinessUnits(db *gorm.DB) ([]BusinessUnit, error) {
	var units []BusinessUnit
	// Kita gunakan Preload("Company") agar data nama perusahaannya ikut terbawa
	err := db.Preload("Company").Find(&units).Error
	return units, err
}

// Membuat Business Unit baru
func CreateBusinessUnit(db *gorm.DB, name string, companyID uint) error {
	return db.Create(&BusinessUnit{
		Name:      name,
		CompanyID: companyID,
	}).Error
}

// Menghapus Business Unit secara permanen
func DeleteBusinessUnit(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&BusinessUnit{}, id).Error
}

// Tambahan: Update Business Unit
func UpdateBusinessUnit(db *gorm.DB, id string, name string, companyID uint) error {
	return db.Model(&BusinessUnit{}).Where("id = ?", id).Updates(BusinessUnit{
		Name:      name,
		CompanyID: companyID,
	}).Error
}
