package services

import (
	"latihan1/models"

	"gorm.io/gorm"
)

type DocumentationService struct {
	DB *gorm.DB
}

// =========================
// SAVE SINGLE DOCUMENTATION
// =========================
func (s *DocumentationService) Save(
	hazardID uint,
	fileURL string,
	docType string,
) error {

	// 1. insert documentations
	doc := models.Documentation{
		FileURL: fileURL,
	}

	if err := s.DB.Create(&doc).Error; err != nil {
		return err
	}

	// 2. insert pivot
	pivot := models.HazardDocumentation{
		HazardID:        hazardID,
		DocumentationID: doc.ID,
		DocType:         docType,
	}

	if err := s.DB.Create(&pivot).Error; err != nil {
		return err
	}

	return nil
}

// =========================
// SAVE MULTIPLE DOCUMENTS
// =========================
func (s *DocumentationService) SaveBulk(
	hazardID uint,
	fileURLs []string,
	docType string,
) error {

	for _, fileURL := range fileURLs {

		// 1. insert documentations
		doc := models.Documentation{
			FileURL: fileURL,
		}

		if err := s.DB.Create(&doc).Error; err != nil {
			return err
		}

		// 2. insert pivot
		pivot := models.HazardDocumentation{
			HazardID:        hazardID,
			DocumentationID: doc.ID,
			DocType:         docType,
		}

		if err := s.DB.Create(&pivot).Error; err != nil {
			return err
		}
	}

	return nil
}
