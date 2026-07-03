package models

import (
	"time"

	"gorm.io/gorm"
)

type CorrectiveActionIncident struct {
	gorm.Model
	IncidentReportID uint `json:"incident_report_id"`

	// Gunakan gorm:"type:text" untuk kolom TEXT
	Action           string `json:"action" gorm:"type:text"`
	ControlHierarchy string `json:"control_hierarchy" gorm:"type:text"`

	// Relasi ke User (PIC)
	UserID *uint `json:"user_id"`
	User   *User `json:"user" gorm:"foreignKey:UserID"`

	DueDate *time.Time `json:"due_date"`
}
