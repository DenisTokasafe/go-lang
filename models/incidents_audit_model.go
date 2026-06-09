package models

import "time"

type IncidentReportedAudit struct {
	ID uint `gorm:"primaryKey"`

	// PERBAIKAN: Ganti HazardID menjadi IncidentReportID
	IncidentReportID uint `json:"incident_report_id"`

	Action string `gorm:"size:20"`

	Before string `gorm:"type:longtext"`
	After  string `gorm:"type:longtext"`

	ChangedBy uint
	ChangedAt time.Time

	User User `gorm:"foreignKey:ChangedBy"`
}
