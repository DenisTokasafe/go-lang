package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name       string `gorm:"type:varchar(255);not null" json:"name"`
	Username   string `gorm:"type:varchar(100);unique" json:"username"`
	Email      string `gorm:"type:varchar(255);unique" json:"email"`
	Password   string `gorm:"type:varchar(255);not null" json:"-"`
	EmployeeID string `gorm:"type:varchar(255);not null" json:"employee_id"`

	// --- TAMBAHAN BARU ---
	DateOfBirth *time.Time `gorm:"type:date" json:"date_of_birth"`
	Gender      string     `gorm:"type:char(1)" json:"gender"` // 'L' atau 'P'

	// Relasi ke Role
	RoleID uint `gorm:"not null" json:"role_id"`
	Role   Role `gorm:"foreignKey:RoleID" json:"role"`
	IsPIC  bool `gorm:"default:false" json:"is_pic"`

	// HUBUNGAN KERJA (Explicit Nullable Fields)
	ContractorID *uint      `json:"contractor_id"`
	Contractor   Contractor `gorm:"foreignKey:ContractorID" json:"contractor"`

	DepartmentID        *uint           `json:"department_id"`
	Department          Department      `gorm:"foreignKey:DepartmentID" json:"department"`
	ModeratedCategories []EventCategory `gorm:"many2many:user_event_categories;" json:"moderated_categories"`
}

// Hook BeforeCreate: Set default role ke 'guest'
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.RoleID == 0 {
		var role Role
		if err := tx.Where("name = ?", "guest").First(&role).Error; err == nil {
			u.RoleID = role.ID
		}
	}
	return
}

// Helper Method: Menampilkan Unit Kerja secara dinamis di Template/Navbar
func (u User) GetWorkUnitName() string {
	if u.ContractorID != nil && u.Contractor.ID != 0 {
		return u.Contractor.Name
	}
	if u.DepartmentID != nil && u.Department.ID != 0 {
		return u.Department.Name
	}
	return "No Unit Assigned"
}

// GetUsersByContractor mengambil semua User (PIC) yang bekerja di kontraktor tertentu
// GetUsersByContractor mengambil semua User yang memang ditunjuk sebagai PIC di kontraktor tersebut
func GetUsersByContractor(db *gorm.DB, contractorID uint) ([]User, error) {
	var users []User
	err := db.Where("contractor_id = ? AND is_pic = ?", contractorID, true).
		Order("name asc").
		Find(&users).Error
	return users, err
}

// GetUsersByDepartment mengambil semua User yang memang ditunjuk sebagai PIC di departemen tersebut
func GetUsersByDepartment(db *gorm.DB, deptID uint) ([]User, error) {
	var users []User
	err := db.Where("department_id = ? AND is_pic = ?", deptID, true).
		Order("name asc").
		Find(&users).Error
	return users, err
}
