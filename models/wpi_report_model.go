package models

import (
	"time"

	"gorm.io/gorm"
)

type WpiReport struct {
	gorm.Model
	// Reference dibuat otomatis saat laporan dibuat dan tidak boleh berubah saat edit.
	Reference  string    `gorm:"uniqueIndex;size:50;not null" json:"reference"`
	TanggalJam time.Time `gorm:"not null" json:"tanggal_jam"`

	LocationID       *uint     `json:"location_id"`
	Location         *Location `gorm:"foreignKey:LocationID"`
	LocationSpecific string    `gorm:"type:varchar(255)" json:"location_specific"`

	SiteName string `gorm:"type:varchar(100)" json:"site_name"`
	Area     string `gorm:"type:varchar(100)" json:"area"`

	CompanyID *uint    `json:"company_id"`
	Company   *Company `gorm:"foreignKey:CompanyID"`

	DepartmentID *uint       `json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID"`

	ContractorID *uint       `json:"contractor_id"`
	Contractor   *Contractor `gorm:"foreignKey:ContractorID"`

	ReviewerID *uint `json:"reviewer_id"`
	Reviewer   *User `gorm:"foreignKey:ReviewerID"`

	ReviewDate *time.Time `gorm:"type:date" json:"review_date"`

	// Relasi ke detail
	Inspectors []WpiInspector `gorm:"foreignKey:WpiReportID" json:"inspectors"`
	Items      []WpiItem      `gorm:"foreignKey:WpiReportID" json:"items"`
}
