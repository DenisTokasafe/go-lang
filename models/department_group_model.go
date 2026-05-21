package models

import "gorm.io/gorm"

type DepartmentGroup struct {
	DepartmentID uint       `gorm:"not null;index;uniqueIndex:idx_department_group"`
	GroupID      uint       `gorm:"not null;index;uniqueIndex:idx_department_group"`
	Department   Department `gorm:"foreignKey:DepartmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Group        Group      `gorm:"foreignKey:GroupID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	gorm.Model
}

func GetAllDepartmentGroups(db *gorm.DB) ([]DepartmentGroup, error) {
	var rows []DepartmentGroup
	tx := db.Preload("Department").Preload("Group").Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return rows, nil
}

func CreateDepartmentGroup(db *gorm.DB, departmentID uint, groupID uint) error {
	return db.Create(&DepartmentGroup{
		DepartmentID: departmentID,
		GroupID:      groupID,
	}).Error
}

func DeleteDepartmentGroup(db *gorm.DB, id string) error {
	return db.Unscoped().Delete(&DepartmentGroup{}, id).Error
}

func UpdateDepartmentGroup(db *gorm.DB, id string, departmentID uint, groupID uint) error {
	return db.Model(&DepartmentGroup{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"department_id": departmentID,
			"group_id":      groupID,
		}).Error
}

func GetDepartmentGroupByID(db *gorm.DB, id string) (DepartmentGroup, error) {
	var row DepartmentGroup
	tx := db.Preload("Department").Preload("Group").First(&row, "id = ?", id)
	if tx.Error != nil {
		return DepartmentGroup{}, tx.Error
	}
	return row, nil
}
