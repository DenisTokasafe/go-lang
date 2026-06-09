package models

import "gorm.io/gorm"

type IncidentDocumentation struct {
	gorm.Model
	// Menghubungkan ke tabel IncidentReport
	IncidentReportID uint           `json:"incident_report_id"`
	IncidentReport   IncidentReport `gorm:"foreignKey:IncidentReportID"`

	// Menghubungkan ke tabel master Documentation
	DocumentationID uint          `json:"documentation_id"`
	Documentation   Documentation `gorm:"foreignKey:DocumentationID"`
}
