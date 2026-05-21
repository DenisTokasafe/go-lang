package models

import "time"

type HazardAudit struct {
	ID uint `gorm:"primaryKey"`

	HazardID uint

	Action string `gorm:"size:20"`

	Before string `gorm:"type:longtext"`
	After  string `gorm:"type:longtext"`

	ChangedBy uint
	ChangedAt time.Time

	User User `gorm:"foreignKey:ChangedBy"`
}
