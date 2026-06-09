// File: models/involved_party.go
package models

import (
	"gorm.io/gorm"
)

// InvolvedParty mewakili BAGIAN 2 – Pihak Terlibat Langsung (Child Table)
type InvolvedParty struct {
	gorm.Model
	// Foreign Key yang menghubungkan ke tabel IncidentReport
	IncidentReportID uint `gorm:"not null;index" json:"incident_report_id"`

	// Detil Personil Lapangan yang diinput dari Form Dinamis
	UserID   *uint `json:"user_id"`
	ReportBy *User `gorm:"foreignKey:UserID"`

	ReporterManual string      `gorm:"type:varchar(255)" json:"reporter_manual"`
	DepartmentID   *uint       `json:"department_id"`
	Department     *Department `gorm:"foreignKey:DepartmentID"` // ID Karyawan atau Nama Perusahaan/Kontraktor
	ContractorID   *uint       `json:"contractor_id"`
	Contractor     *Contractor `gorm:"foreignKey:ContractorID"` // ID Karyawan atau Nama Perusahaan/Kontraktor
	Jabatan        string      `gorm:"type:varchar(100)" json:"jabatan"`
	Roster         string      `gorm:"type:varchar(50)" json:"roster"`        // Contoh: 4-2, 2-1, dll.
	Shift          string      `gorm:"type:varchar(50)" json:"shift"`         // Contoh: Day, Night
	Keterlibatan   string      `gorm:"type:varchar(150)" json:"keterlibatan"` // Saksi, Korban Cedera, Operator Unit, dll.
	Pengalaman     int         `gorm:"type:int;default:0" json:"pengalaman"`  // Pengalaman kerja dalam hitungan tahun
}
