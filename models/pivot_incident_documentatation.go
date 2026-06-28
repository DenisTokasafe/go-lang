package models

import "gorm.io/gorm"

type IncidentDocumentation struct {
	gorm.Model
	// Menghubungkan ke tabel IncidentReport
	IncidentReportID uint
	DocumentationID  uint
	DocType          string         `gorm:"type:varchar(20)" json:"doc_type"`
	IncidentReport   IncidentReport `gorm:"foreignKey:IncidentReportID"`
	Documentation    Documentation  `gorm:"foreignKey:DocumentationID"`
}
