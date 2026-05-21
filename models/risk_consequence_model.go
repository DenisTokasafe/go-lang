package models

import (
	"gorm.io/gorm"
)

type RiskConsequence struct {
	gorm.Model
	Name          string `gorm:"not null" json:"name"`            // Contoh: 5 - Catastrophic
	Description   string `gorm:"type:text" json:"description"`    // Deskripsi detail HSE/ENV/FIN
	Reportable    string `gorm:"size:5" json:"reportable"`        // Yes / No
	Sequence      int    `gorm:"default:0" json:"sequence"`       // Urutan tampilan
	SeverityLevel int    `gorm:"default:0" json:"severity_level"` // Level keparahan (1-5)
}

// Fungsi bantu untuk CRUD

func GetAllRiskConsequences(db *gorm.DB) ([]RiskConsequence, error) {
	var consequences []RiskConsequence
	// Diurutkan berdasarkan severity_level atau sequence sesuai gambar
	err := db.Order("severity_level ASC").Find(&consequences).Error
	return consequences, err
}

func CreateRiskConsequence(db *gorm.DB, name string, desc string, report string, seq int, sev int) error {
	return db.Create(&RiskConsequence{
		Name:          name,
		Description:   desc,
		Reportable:    report,
		Sequence:      seq,
		SeverityLevel: sev,
	}).Error
}

func DeleteRiskConsequence(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&RiskConsequence{}, id).Error
}
