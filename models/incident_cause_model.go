package models

import "gorm.io/gorm"

// IncidentCause menyimpan data Bagian 6 (Penyebab Langsung & Dasar)
type IncidentCause struct {
	gorm.Model

	IncidentReportID uint            `gorm:"index" json:"incident_report_id"`
	IncidentReport   *IncidentReport `gorm:"foreignKey:IncidentReportID;constraint:OnDelete:CASCADE;" json:"-"`

	// Kategori: "unsafe_condition", "unsafe_act", "personal_factor", "job_factor", "control_weakness"
	CategoryType string `gorm:"type:varchar(50);index" json:"category_type"`

	// ID dari pilihan dropdown (Master Data ScatOption)
	ScatOptionID *uint `json:"scat_option_id"`

	// Jika Anda sudah punya model master ScatOption, buka komentar di bawah ini:
	// ScatOption    *ScatOption     `gorm:"foreignKey:ScatOptionID" json:"-"`

	// Penjelasan manual
	Description string `gorm:"type:text" json:"description"`
}
