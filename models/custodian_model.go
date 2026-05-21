package models

import "gorm.io/gorm"

type Custodian struct {
	DepartmentID uint       `gorm:"not null;index;uniqueIndex:idx_dept_contractor"`
	ContractorID uint       `gorm:"not null;index;uniqueIndex:idx_dept_contractor"`
	Department   Department `gorm:"foreignKey:DepartmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Contractor   Contractor `gorm:"foreignKey:ContractorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	gorm.Model
}

func GetAllCustodians(db *gorm.DB) ([]Custodian, error) {
	var custodians []Custodian

	tx := db.
		Preload("Department").
		Preload("Contractor").
		Find(&custodians)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return custodians, nil
}

func CreateCustodian(db *gorm.DB, departmentID uint, contractorID uint) error {
	return db.Create(&Custodian{
		DepartmentID: departmentID,
		ContractorID: contractorID,
	}).Error
}

func DeleteCustodian(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&Custodian{}, id).Error
}

func UpdateCustodian(db *gorm.DB, id string, departmentID uint, contractorID uint) error {
	return db.Model(&Custodian{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"department_id": departmentID,
			"contractor_id": contractorID,
		}).Error
}

func GetCustodianByID(db *gorm.DB, id string) (Custodian, error) {
	var custodian Custodian

	tx := db.
		Preload("Department").
		Preload("Contractor").
		First(&custodian, "id = ?", id)

	if tx.Error != nil {
		return Custodian{}, tx.Error
	}

	return custodian, nil
}
