package models

import (
	"gorm.io/gorm"
)

type EventCategory struct {
	gorm.Model
	// ParentID menggunakan pointer *uint agar bisa bernilai NULL (untuk kategori utama)
	ParentID      *uint          `gorm:"index;default:null" json:"parent_id"`
	Parent        *EventCategory `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	CategoryGroup string         `gorm:"type:varchar(50);not null" json:"category_group"` // 'lead' atau 'incident'
	Name          string         `gorm:"type:varchar(255);not null" json:"name"`
	Code          string         `gorm:"type:varchar(255);unique;not null" json:"code"`
	Status        string         `gorm:"type:varchar(50);default:'enabled'" json:"status"` // 'enabled' atau 'disabled'

	// Relasi untuk mengambil anak (Sub-types) secara otomatis jika diperlukan
	SubCategories []EventCategory `gorm:"foreignKey:ParentID" json:"sub_categories,omitempty"`
	Moderators    []User          `gorm:"many2many:user_event_categories;" json:"moderators,omitempty"`
}

// Menentukan nama tabel secara eksplisit
func (EventCategory) TableName() string {
	return "event_categories"
}

// --- Fungsi Bantu CRUD ---

// CreateEventCategory menambahkan data kategori baru (bisa induk atau anak)
func CreateEventCategory(db *gorm.DB, parentID *uint, group, name, code, status string) error {
	newCat := EventCategory{
		ParentID:      parentID,
		CategoryGroup: group,
		Name:          name,
		Code:          code,
		Status:        status,
	}
	return db.Create(&newCat).Error
}

// UpdateEventCategory mengupdate data kategori yang sudah ada
func UpdateEventCategory(db *gorm.DB, id interface{}, parentID *uint, group, name, code, status string) error {
	return db.Model(&EventCategory{}).Where("id = ?", id).Updates(map[string]interface{}{
		"parent_id":      parentID,
		"category_group": group,
		"name":           name,
		"code":           code,
		"status":         status,
	}).Error
}

// DeleteEventCategory menghapus data secara permanen
func DeleteEventCategory(db *gorm.DB, id interface{}) error {
	return db.Unscoped().Delete(&EventCategory{}, id).Error
}

// GetEventCategoriesByGroup mengambil data berdasarkan grup (lead/incident) dan filter search
func GetEventCategoriesByGroup(db *gorm.DB, group, search string, parentOnly bool) ([]EventCategory, error) {
	var categories []EventCategory
	query := db.Where("category_group = ?", group)

	if parentOnly {
		query = query.Where("parent_id IS NULL")
	}

	if search != "" {
		s := "%" + search + "%"
		query = query.Where("(name LIKE ? OR code LIKE ?)", s, s)
	}

	err := query.Order("LENGTH(code) ASC, code ASC").Find(&categories).Error
	return categories, err
}
