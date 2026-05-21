package models

import (
	"gorm.io/gorm"
)

type RiskMatrix struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	RiskConsequenceID uint            `gorm:"not null;uniqueIndex:idx_matrix" json:"risk_consequence_id"`
	RiskConsequence   RiskConsequence `gorm:"foreignKey:RiskConsequenceID"`

	RiskLikelihoodID uint           `gorm:"not null;uniqueIndex:idx_matrix" json:"risk_likelihood_id"`
	RiskLikelihood   RiskLikelihood `gorm:"foreignKey:RiskLikelihoodID"`

	RiskAssessmentID uint               `gorm:"not null" json:"risk_assessment_id"`
	RiskAssessment   RiskAssessmentCode `gorm:"foreignKey:RiskAssessmentID"`
}

// RiskMatrixResponse digunakan jika Anda ingin mem-parsing data ke JSON yang lebih flat
type RiskMatrixResponse struct {
	ID              uint   `json:"id"`
	ConsequenceName string `json:"consequence_name"`
	LikelihoodName  string `json:"likelihood_name"`
	AssessmentName  string `json:"assessment_name"`
	AssessmentColor string `json:"assessment_color"`
}

// --- Fungsi Helper untuk CRUD (Disesuaikan untuk Controller) ---

// CreateRiskMatrix membuat data baru
func CreateRiskMatrix(db *gorm.DB, consequenceID, likelihoodID, assessmentID uint) error {
	return db.Create(&RiskMatrix{
		RiskConsequenceID: consequenceID,
		RiskLikelihoodID:  likelihoodID,
		RiskAssessmentID:  assessmentID,
	}).Error
}

// UpdateRiskMatrix hanya mengupdate hasil assessment-nya saja berdasarkan ID matrix
func UpdateRiskMatrix(db *gorm.DB, id string, assessmentID uint) error {
	return db.Model(&RiskMatrix{}).Where("id = ?", id).Update("risk_assessment_id", assessmentID).Error
}

// DeleteRiskMatrix menghapus data secara permanen (Unscoped)
func DeleteRiskMatrix(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&RiskMatrix{}, id).Error
}

// GetMatrixByCoordinate mencari apakah sudah ada aturan untuk koordinat tertentu (berguna untuk Import Excel)
func GetMatrixByCoordinate(db *gorm.DB, consID, likeID uint) (*RiskMatrix, error) {
	var matrix RiskMatrix
	err := db.Where("risk_consequence_id = ? AND risk_likelihood_id = ?", consID, likeID).First(&matrix).Error
	if err != nil {
		return nil, err
	}
	return &matrix, nil
}
