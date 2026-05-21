package models

import (
	"time"

	"gorm.io/gorm"
)

type HazardStatus string

const (
	HazardStatusSubmit     HazardStatus = "submit"
	HazardStatusInProgress HazardStatus = "in_progress"
	HazardStatusPending    HazardStatus = "pending"
	HazardStatusClosed     HazardStatus = "closed"
	HazardStatusCancelled  HazardStatus = "cancelled"
)

type Hazard struct {
	gorm.Model

	// Relasi Kategori & Risiko
	EventCategoryID uint          `json:"event_category_id"`
	EventCategory   EventCategory `gorm:"foreignKey:EventCategoryID"`

	RiskMatrixID uint       `json:"risk_matrix_id"`
	RiskMatrix   RiskMatrix `gorm:"foreignKey:RiskMatrixID;references:ID"`

	// Status Hazard
	Status HazardStatus `gorm:"type:enum('submit','in_progress','pending','closed','cancelled');default:'submit'" json:"status"`

	// Relasi Scat Option
	ScatOptionID uint       `gorm:"column:scat_option_id;after:event_category_id"`
	ScatOption   ScatOption `gorm:"foreignKey:ScatOptionID;references:ID"`

	// Lokasi Kejadian
	LocationID       uint     `json:"location_id"`
	Location         Location `gorm:"foreignKey:LocationID"`
	LocationSpecific string   `gorm:"type:varchar(255)" json:"location_specific"`

	// Waktu & Deskripsi
	TanggalWaktu time.Time `gorm:"not null" json:"tanggal_waktu"`
	Deskripsi    string    `gorm:"type:text;not null" json:"deskripsi"`

	// Tindakan Perbaikan
	CorrectiveAction string `gorm:"type:text" json:"corrective_action"`

	// Pelapor (Dinamis)
	ReportByID *uint `json:"report_by_id"`
	ReportBy   *User `gorm:"foreignKey:ReportByID"`

	ReporterManual string `gorm:"type:varchar(255)" json:"reporter_manual"`

	// Penanggung Jawab (Divisi/Unit)
	DepartmentID *uint       `json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID"`

	ContractorID *uint       `json:"contractor_id"`
	Contractor   *Contractor `gorm:"foreignKey:ContractorID"`

	// PIC (Berdasarkan relasi User ke Dept/Cont)
	PicID uint `gorm:"not null" json:"pic_id"`
	PIC   User `gorm:"foreignKey:PicID"`

	// SCAT Options (Multiple Selection / Many-to-Many)
	ScatOptions []ScatOption `gorm:"many2many:hazard_scat_options;" json:"scat_options"`

	Documentations    []HazardDocumentation    `gorm:"foreignKey:HazardID" json:"documentations"`
	Audits            []HazardAudit            `gorm:"foreignKey:HazardID"`
	CorrectiveActions []CorrectiveActionHazard `gorm:"foreignKey:HazardID"`
}
