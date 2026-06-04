package helpers

import (
	"fmt"
	"strings"

	"latihan1/cmd/web/config"
	"latihan1/models"

	"gorm.io/gorm"
)

// =====================================
// GET MODERATOR EMAILS
// =====================================
func GetModeratorEmails(
	db *gorm.DB,
	eventCategoryID uint,
) ([]string, error) {

	// =========================
	// GET CATEGORY
	// =========================
	var category models.EventCategory

	err := db.
		First(
			&category,
			eventCategoryID,
		).Error

	if err != nil {

		return nil, err
	}

	// =========================
	// GUNAKAN PARENT ID
	// JIKA ADA
	// =========================
	targetCategoryID := eventCategoryID

	if category.ParentID != nil {

		targetCategoryID = *category.ParentID
	}

	// =========================
	// GET MODERATORS
	// =========================
	var users []models.User

	err = db.
		Joins(
			"JOIN user_event_categories uec ON uec.user_id = users.id",
		).
		Where(
			"uec.event_category_id = ?",
			targetCategoryID,
		).
		Find(&users).Error

	if err != nil {

		return nil, err
	}

	// =========================
	// UNIQUE EMAILS
	// =========================
	emailMap := make(map[string]bool)

	for _, user := range users {

		email := strings.TrimSpace(
			user.Email,
		)

		if email != "" {

			emailMap[email] = true
		}
	}

	var emails []string

	for email := range emailMap {

		emails = append(
			emails,
			email,
		)
	}

	return emails, nil
}

// =====================================
// GET PIC EMAIL
// =====================================
func GetPICEmail(
	db *gorm.DB,
	picID uint,
) (string, error) {

	var user models.User

	err := db.
		First(&user, picID).Error

	if err != nil {

		return "", err
	}

	return strings.TrimSpace(
		user.Email,
	), nil
}

// =====================================
// MERGE EMAILS
// =====================================
func MergeEmails(
	emailGroups ...[]string,
) []string {

	emailMap := make(map[string]bool)

	for _, group := range emailGroups {

		for _, email := range group {

			email = strings.TrimSpace(
				email,
			)

			if email != "" {

				emailMap[email] = true
			}
		}
	}

	var result []string

	for email := range emailMap {

		result = append(
			result,
			email,
		)
	}

	return result
}

// =====================================
// SEND EMAIL TO MANY USERS
// =====================================
func SendEmailToMany(
	to []string,
	subject string,
	htmlBody string,
) error {

	if len(to) == 0 {

		return fmt.Errorf(
			"email tujuan kosong",
		)
	}

	return config.SendEmail(
		to,
		subject,
		htmlBody,
	)
}
