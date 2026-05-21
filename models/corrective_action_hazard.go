package models

import (
	"time"

	"gorm.io/gorm"
)

type CorrectiveActionHazard struct {
	gorm.Model

	// =========================
	// RELASI HAZARD
	// =========================
	HazardID uint   `gorm:"not null" json:"hazard_id"`
	Hazard   Hazard `gorm:"foreignKey:HazardID"`

	// =========================
	// FOLLOW UP ACTION
	// =========================
	FollowupAction string `gorm:"type:text" json:"followup_action"`

	// =========================
	// RELASI DEPARTMENT
	// =========================
	DepartmentTerkaitID *uint       `json:"department_terkait_id"`
	DepartmentTerkait   *Department `gorm:"foreignKey:DepartmentTerkaitID"`

	// =========================
	// RELASI CONTRACTOR
	// =========================
	ContractorTerkaitID *uint       `json:"contractor_terkait_id"`
	ContractorTerkait   *Contractor `gorm:"foreignKey:ContractorTerkaitID"`

	// =========================
	// RELASI PIC TERKAIT
	// =========================
	PicTerkaitID *uint `json:"pic_terkait_id"`

	PICTerkait *User `gorm:"foreignKey:PicTerkaitID"`

	// =========================
	// TARGET & COMPLETED
	// =========================
	DueDate *time.Time `gorm:"type:date" json:"due_date"`

	CompletedOn *time.Time `gorm:"type:date" json:"completed_on"`
	// =========================
	// TEMP DATA (NOT DB)
	// =========================
	Pics []User `gorm:"-" json:"pics"`
}
