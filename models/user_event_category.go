package models

type UserEventCategory struct {
	UserID          uint `gorm:"primaryKey"`
	EventCategoryID uint `gorm:"primaryKey"`

	User          User
	EventCategory EventCategory
}
