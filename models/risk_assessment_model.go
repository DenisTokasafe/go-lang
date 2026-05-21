package models

import (
	"gorm.io/gorm"
)

type RiskAssessmentCode struct {
	gorm.Model
	Name                string `gorm:"not null" json:"name"`
	Notes               string `gorm:"type:text" json:"notes"`
	ActionDays          int    `gorm:"default:0" json:"action_days"`
	Sequence            int    `gorm:"default:0" json:"sequence"`
	InvestigationReqd   string `gorm:"type:text" json:"investigation_reqd"`
	ReportingObligation string `gorm:"size:50" json:"reporting_obligation"`
	Colour              string `gorm:"size:20" json:"colour"`
}

func GetAllRiskAssessmentCodes(db *gorm.DB) ([]RiskAssessmentCode, error) {
	var assessments []RiskAssessmentCode
	// Diurutkan berdasarkan sequence (1-4) sesuai standar image_ddf941.png
	err := db.Order("sequence ASC").Find(&assessments).Error
	return assessments, err
}

func CreateRiskAssessmentCode(db *gorm.DB, data RiskAssessmentCode) error {
	return db.Create(&data).Error
}

func UpdateRiskAssessmentCode(db *gorm.DB, id string, data map[string]interface{}) error {
	return db.Model(&RiskAssessmentCode{}).Where("id = ?", id).Updates(data).Error
}

func DeleteRiskAssessmentCode(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&RiskAssessmentCode{}, id).Error
}
