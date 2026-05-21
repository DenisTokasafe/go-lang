package models

import (
	"gorm.io/gorm"
)

type Documentation struct {
	gorm.Model

	// FILE DATA
	FileURL  string `gorm:"type:varchar(255);not null" json:"file_url"`
	FileName string `gorm:"type:varchar(255)" json:"file_name"`
	FileType string `gorm:"type:varchar(50)" json:"file_type"`
	FileSize int64  `json:"file_size"`

	// RELASI KE PIVOT
	HazardLinks []HazardDocumentation `gorm:"foreignKey:DocumentationID" json:"hazard_links"`

	// OPTIONAL: siapa upload
	UploadedByID *uint `json:"uploaded_by_id"`
	UploadedBy   *User `gorm:"foreignKey:UploadedByID"`
}
