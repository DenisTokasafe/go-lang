package models

import (
	"gorm.io/gorm"
)

type RiskLikelihood struct {
	gorm.Model
	Name     string `gorm:"not null" json:"name"`
	Notes    string `json:"notes"`
	Sequence int    `gorm:"default:0" json:"sequence"`
}

// Fungsi bantu untuk CRUD menggunakan GORM

func GetAllRiskLikelihoods(db *gorm.DB) ([]RiskLikelihood, error) {
	var likelihoods []RiskLikelihood
	// Diurutkan berdasarkan sequence agar muncul A, B, C, D, E secara berurutan
	err := db.Order("sequence ASC").Find(&likelihoods).Error
	return likelihoods, err
}

func CreateRiskLikelihood(db *gorm.DB, name string, notes string, sequence int) error {
	return db.Create(&RiskLikelihood{
		Name:     name,
		Notes:    notes,
		Sequence: sequence,
	}).Error
}

func UpdateRiskLikelihood(db *gorm.DB, id string, name string, notes string, sequence int) error {
	return db.Model(&RiskLikelihood{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":     name,
		"notes":    notes,
		"sequence": sequence,
	}).Error
}

func DeleteRiskLikelihood(db *gorm.DB, id string) error {
	// Menggunakan Unscoped() sesuai permintaan Anda untuk hapus permanen
	return db.Unscoped().Delete(&RiskLikelihood{}, id).Error
}
