package models

import (
	"gorm.io/gorm"
)

type HazardDocumentation struct {
	gorm.Model

	HazardID        uint
	DocumentationID uint

	DocType string `gorm:"type:varchar(20)" json:"doc_type"`
	// contoh: desc / corrective / evidence

	// =========================
	// RELATION
	// =========================
	Hazard        Hazard        `gorm:"foreignKey:HazardID"`
	Documentation Documentation `gorm:"foreignKey:DocumentationID"`
}
