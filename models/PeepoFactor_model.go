package models

import (
	"time"

	"gorm.io/gorm"
)

type PeepoFactor struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	IncidentReportID uint           `json:"incident_report_id"`
	Factor           string         `gorm:"type:varchar(50);not null" json:"factor"` // Orang, Peralatan, Lingkungan, Prosedur, Organisasi
	Findings         string         `gorm:"type:text" json:"findings"`
	Description      string         `gorm:"type:text" json:"description"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}
