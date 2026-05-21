package config

import (
	"fmt"
	"latihan1/models"
	"latihan1/seeders"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {

	dsn := "root:@tcp(127.0.0.1:3306)/go_crud_db?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal menghubungkan ke database:", err)
	}

	db.AutoMigrate(
		&models.Company{},
		&models.Role{},
		&models.Contractor{},
		&models.Department{},
		&models.BusinessUnit{},
		&models.Custodian{},
		&models.Group{},
		&models.DepartmentGroup{},
		&models.Manhours{},
		&models.Location{},
		&models.RiskLikelihood{},
		&models.RiskConsequence{},
		&models.RiskAssessmentCode{},
		&models.RiskMatrix{},
		&models.ScatOption{},
		&models.BodyPart{},
		&models.EventCategory{},
		&models.User{},
		&models.Hazard{},
		&models.Documentation{},
		&models.HazardDocumentation{},
		&models.HazardAudit{},
		&models.UserEventCategory{},
		&models.CorrectiveActionHazard{},
	)

	fmt.Println("Database terkoneksi & Migrasi berhasil!")

	seeders.SeedScatOptions(db)
	seeders.SeedAdminUser(db)

	return db
}
