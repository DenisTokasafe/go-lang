package models

import (
	"time"

	"gorm.io/gorm"
)

// InvestigationParticipant menyimpan data tim investigasi
type InvestigationParticipant struct {
	ID               uint `gorm:"primarykey" json:"id"`
	IncidentReportID uint `json:"incident_report_id"` // Foreign Key ke IncidentReport

	// Peran dalam investigasi ("Pemimpin Investigasi", "Facilitator", "Anggota")
	Role string `gorm:"type:varchar(50);not null" json:"role"`

	// Data personal sesuai mockup form Anda
	EmployeID    *uint       `json:"employee_id"`
	ReportBy     *User       `gorm:"foreignKey:EmployeID"`
	Jabatan      string      `gorm:"type:varchar(100)" json:"jabatan"`
	WorkType     string      `gorm:"type:varchar(50)" json:"work_type"`
	DepartmentID *uint       `json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID"`

	ContractorID *uint       `json:"contractor_id"`
	Contractor   *Contractor `gorm:"foreignKey:ContractorID"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
