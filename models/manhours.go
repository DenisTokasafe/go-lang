package models

import (
	"time"

	"gorm.io/gorm"
)

type Manhours struct {
	gorm.Model
	Month             time.Time       `gorm:"not null"`
	EntityType        string          `gorm:"not null;type:enum('mine_company','contractor');default:'mine_company'"`
	BusinessUnitID    *uint           `gorm:"index"`
	BusinessUnit      BusinessUnit    `gorm:"foreignKey:BusinessUnitID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	ContractorID      *uint           `gorm:"index"`
	Contractor        Contractor      `gorm:"foreignKey:ContractorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	DepartmentGroupID *uint           `gorm:"index"`
	DepartmentGroup   DepartmentGroup `gorm:"foreignKey:DepartmentGroupID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CustodianID       *uint           `gorm:"index"`
	Custodian         Custodian       `gorm:"foreignKey:CustodianID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	SupervisorHours   float64         `gorm:"default:0"`
	SupervisorCount   uint            `gorm:"default:0"`
	OperationalHours  float64         `gorm:"default:0"`
	OperationalCount  uint            `gorm:"default:0"`
	AdminHours        float64         `gorm:"default:0"`
	AdminCount        uint            `gorm:"default:0"`
	Notes             string          `gorm:"type:text"`
}

func GetAllManhours(db *gorm.DB) ([]Manhours, error) {
	var rows []Manhours
	err := db.
		Preload("BusinessUnit.Company").
		Preload("Contractor").
		Preload("DepartmentGroup.Department").
		Preload("DepartmentGroup.Group").
		Preload("Custodian.Department").
		Preload("Custodian.Contractor").
		Find(&rows).Error
	return rows, err
}

func CreateManhours(db *gorm.DB, month time.Time, entityType string, businessUnitID *uint, contractorID *uint, departmentGroupID *uint, custodianID *uint, supervisorHours float64, supervisorCount uint, operationalHours float64, operationalCount uint, adminHours float64, adminCount uint, notes string) error {
	return db.Create(&Manhours{
		Month:             month,
		EntityType:        entityType,
		BusinessUnitID:    businessUnitID,
		ContractorID:      contractorID,
		DepartmentGroupID: departmentGroupID,
		CustodianID:       custodianID,
		SupervisorHours:   supervisorHours,
		SupervisorCount:   supervisorCount,
		OperationalHours:  operationalHours,
		OperationalCount:  operationalCount,
		AdminHours:        adminHours,
		AdminCount:        adminCount,
		Notes:             notes,
	}).Error
}

func DeleteManhours(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&Manhours{}, id).Error
}
